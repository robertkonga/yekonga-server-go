package yekonga

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/robertkonga/yekonga-server-go/config"
	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/helper/console"
	"github.com/robertkonga/yekonga-server-go/helper/jwt"
	"github.com/robertkonga/yekonga-server-go/helper/logger"
	"github.com/robertkonga/yekonga-server-go/plugins/graphql"
)

// Allowed file extensions
var DefaultExtensions = [...]string{
	// Web Files
	".html", ".css", ".js",

	// Image Files
	".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico",
	".bmp", ".webp", ".tiff", ".tif", ".avif",

	// Font Files
	".ttf", ".otf", ".woff", ".woff2", ".eot",

	// documents
	".xlsx", ".pdf", ".doc", ".text", ".txt", ".csv",
	".docx", ".odt", ".rtf", ".md", ".xls", ".ods",
}

func (y *YekongaData) initialize() {
	logger.Info("App Name", y.Config.AppName)
	dir := y.HomeDirectory()
	logger.Info("Data Directory", dir)

	y.All("/health", func(req *Request, res *Response) {
		res.Text("Ok!")
	})

	y.All("/api-health", func(req *Request, res *Response) {
		res.Json(map[string]string{"status": "OK!"})
	})

	y.All("/check-connection", func(req *Request, res *Response) {
		id := y.Config.ConnectionID
		if helper.IsEmpty(id) {
			id = "YEKONGA_CONNECTED"
		}

		res.Text(id)
	})

	if y.Config.IsAuthorizationServer {
		y.Get("/me", y.authHandler)
		y.Post("/me", y.authHandler)

		y.Get("/refresh", y.refreshHandler)
		y.Post("/refresh", y.refreshHandler)
	}

	y.initializer_socket_routes()

	for _, public := range y.Config.Public {
		if !(strings.HasPrefix(public, "/") || strings.HasPrefix(public, "./")) {
			public = "./" + public
		}

		if !helper.FileExists(public) {
			public = helper.GetPath(public)
		}

		if helper.FileExists(public) {
			// Configure static file serving
			err := y.Static(StaticConfig{
				Directory:   public,       // Serve files from ./public directory
				PathPrefix:  "/",          // Access files at /public URL path
				IndexFile:   "index.html", // Default index file
				Extensions:  DefaultExtensions[:],
				CacheMaxAge: 86400, // Cache for 24 hours
			})

			if err != nil {
				logger.Error("Failed to configure static file serving:", err)
			}
		} else {
			logger.Error("Failed", "Public directory not exist", public)
		}
	}

	if y.Config.IsAuthorizationServer {
		y.All(y.Config.Graphql.ApiAuthRoute, func(req *Request, res *Response) {
			requestString := req.Query("query")
			requestBody := req.Body()
			variableValues := map[string]interface{}{}
			operationName := ""

			graphqlContext := RequestContext{
				Auth:         req.Auth(),
				App:          y,
				Request:      req,
				Response:     res,
				TokenPayload: req.TokenPayload(),
				Client:       req.Client(),
			}

			if body, oki := requestBody.(map[string]interface{}); oki {
				if data, ok := body["query"]; ok {
					if str, ok := data.(string); ok {
						requestString = str
					}
				}
				if data, ok := body["operationName"]; ok {
					if str, ok := data.(string); ok {
						operationName = str
					}
				}
				if data, ok := body["variables"]; ok {
					if str, ok := data.(map[string]interface{}); ok {
						variableValues = str
					}
				}
			}
			// requestStringMap := helper.ToMap(graphql.Parser(requestString))
			// querySelectors := helper.ExtractGraphqlQuery(requestStringMap, 0)
			// graphqlContext.QuerySelectors = querySelectors
			currentContext := context.WithValue(req.HttpRequest.Context(), RequestContextKey, &graphqlContext)

			// helper.SaveToFile(graphql.Parser(requestString), "graphql.data.json")
			// logger.Error(querySelectors)

			// start := time.Now()
			result := graphql.Do(graphql.Params{
				Schema:         y.graphqlBuild.AuthSchema,
				RequestString:  requestString,
				Context:        currentContext,
				VariableValues: variableValues,
				OperationName:  operationName,
			})

			// helper.TrackTime(&start, "Graphql query execute")
			res.Json(result)
			// helper.TrackTime(&start, "Json encode")
			// logger.Error("===== end ======")
		})
	}

	y.All(y.Config.Graphql.ApiRoute, func(req *Request, res *Response) {
		requestString := req.Query("query")
		requestBody := req.Body()
		variableValues := map[string]interface{}{}
		operationName := ""

		graphqlContext := RequestContext{
			Auth:         req.Auth(),
			App:          y,
			Request:      req,
			Response:     res,
			TokenPayload: req.TokenPayload(),
			Client:       req.Client(),
		}

		if body, oki := requestBody.(map[string]interface{}); oki {
			if data, ok := body["query"]; ok {
				if str, ok := data.(string); ok {
					requestString = str
				}
			}
			if data, ok := body["operationName"]; ok {
				if str, ok := data.(string); ok {
					operationName = str
				}
			}
			if data, ok := body["variables"]; ok {
				if str, ok := data.(map[string]interface{}); ok {
					variableValues = str
				}
			}
		}
		// requestStringMap := helper.ToMap(graphql.Parser(requestString))
		// querySelectors := helper.ExtractGraphqlQuery(requestStringMap, 0)
		// graphqlContext.QuerySelectors = querySelectors
		currentContext := context.WithValue(req.HttpRequest.Context(), RequestContextKey, &graphqlContext)

		// helper.SaveToFile(graphql.Parser(requestString), "graphql.data.json")
		// logger.Error(querySelectors)

		// start := time.Now()
		result := graphql.Do(graphql.Params{
			Schema:         y.graphqlBuild.Schema,
			RequestString:  requestString,
			Context:        currentContext,
			VariableValues: variableValues,
			OperationName:  operationName,
		})

		// helper.TrackTime(&start, "Graphql query execute")
		res.Json(result)
		// helper.TrackTime(&start, "Json encode")
		// logger.Error("===== end ======")
	})

	y.initializer_other_routes()
}

func (y *YekongaData) authHandler(req *Request, res *Response) {
	var user *datatype.DataMap
	auth := req.Auth()

	if helper.IsNotEmpty(auth) {
		requestContext := &RequestContext{
			App:      y,
			Auth:     req.Auth(),
			Client:   req.Client(),
			Request:  req,
			Response: res,
		}

		user = y.GetLoginData(requestContext, &LoginData{
			UserID:    auth.ID,
			ProfileID: auth.ProfileID,
		})

		if helper.IsNotEmpty(user) {
			(*user)["token"] = nil
		}

		res.Json(user)
		return
	}

	res.Status(http.StatusUnauthorized)
	res.Json(datatype.DataMap{
		"error": "Missing or Invalid token",
	})
}

func (y *YekongaData) refreshHandler(req *Request, res *Response) {
	result, status := y.refreshTokenProcess(req, res, nil)

	if !y.Config.SecureAuthentication {
		result["token"] = nil
	}

	res.Status(status)
	res.Json(result)
}

func (y *YekongaData) refreshTokenProcess(req *Request, res *Response, refreshToken interface{}) (datatype.DataMap, int) {
	var data *datatype.DataMap
	status := http.StatusUnauthorized
	result := datatype.DataMap{}

	if helper.IsEmpty(refreshToken) {
		refreshToken = req.GetContext(string(RefreshTokenKey))
	}

	if helper.IsNotEmpty(refreshToken) {
		hashedToken := helper.HashRefreshToken(helper.ToString(refreshToken))
		data = y.ModelQuery("RefreshToken").Where("tokenHash", hashedToken).FindOne(nil)

		if helper.IsNotEmpty(data) {
			revoked := helper.GetValueOfBoolean(data, "revoked")

			if revoked {
				result["error"] = "refresh_token is revoked"
			} else {
				expiresAt := helper.GetValueOfDate(data, "expiresAt")
				today := helper.GetTimestamp(nil)

				if expiresAt.After(today) {
					domain := req.Client().OriginDomain()
					tokenDomain := helper.GetValueOfString(data, "domain")
					userId := helper.GetValueOfString(data, "userId")
					tenantId := helper.GetValueOfString(data, "tenantId")
					profileId := helper.GetValueOfString(data, "profileId")
					adminId := helper.GetValueOfString(data, "adminId")

					if tokenDomain == domain {
						payload := TokenPayload{
							Domain:      domain,
							TenantId:    tenantId,
							ProfileId:   profileId,
							UserId:      userId,
							AdminId:     adminId,
							Roles:       make([]string, 0),
							Permissions: make([]string, 0),
							ExpiresAt:   today.Add(time.Minute * 15),
						}

						newAccessToken, _ := jwt.EncodeJWT(payload.ToMap(), y.Config.Authentication.SecretToken)
						newRefreshToken := y.getRefreshToken(*req.Client(), payload)

						status = http.StatusOK
						result[string(AccessTokenKey)] = newAccessToken
						result[string(RefreshTokenKey)] = newRefreshToken

						y.setAuthCookies(&RequestContext{
							App:      y,
							Request:  req,
							Response: res,
							Client:   req.Client(),
						}, newAccessToken, newRefreshToken)

						y.ModelQuery("RefreshToken").Where("tokenHash", hashedToken).Update(datatype.DataMap{
							"revoked": true,
						}, nil)
					} else {
						console.Log(req.Client())
						console.Log(data)
						result["error"] = "Domain mismatch"
					}
				} else {
					result["error"] = "refresh_token expired"
				}
			}
		} else {
			result["error"] = "Invalid refresh_token"
		}
	} else {
		result["error"] = "Empty refresh_token"
	}

	return result, status
}

func NewDatabaseStructure(file string, config *config.YekongaConfig) *DatabaseStructureType {
	if !helper.FileExists(file) {
		file = helper.GetPath(file)
	}

	var databaseAuthorizationStructure = DefaultAuthorizationDatabaseStructure
	var databaseStructure = DefaultDatabaseStructure
	var extraDatabaseStructure map[string]map[string]map[string]interface{}
	data, err := helper.LoadJSONFile(file)

	if err != nil {
		fmt.Println(err)
	}

	json.Unmarshal(helper.ToByte(data), &extraDatabaseStructure)

	if config.IsAuthorizationServer {
		for k, v := range databaseAuthorizationStructure {
			k = helper.ToCamelCase(helper.Pluralize(k))
			if _, ok := databaseStructure[k]; ok {
				for kn, vn := range extraDatabaseStructure[k] {
					databaseStructure[k][kn] = vn
				}
			} else {
				databaseStructure[k] = v
			}
		}
	}

	for k, v := range extraDatabaseStructure {
		k = helper.ToCamelCase(helper.Pluralize(k))
		if _, ok := databaseStructure[k]; ok {
			for kn, vn := range extraDatabaseStructure[k] {
				databaseStructure[k][kn] = vn
			}
		} else {
			databaseStructure[k] = v
		}
	}

	return &databaseStructure

}
