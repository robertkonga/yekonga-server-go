package yekonga

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/helper/jwt"
)

// Define custom middleware keys
type MiddlewareType string

const (
	GlobalMiddleware  MiddlewareType = "global"
	InitMiddleware    MiddlewareType = "init"
	PreloadMiddleware MiddlewareType = "preload"
	CatchMiddleware   MiddlewareType = "catch"
)

func BillingMiddleware(req *Request, res *Response) (int, error) {
	return http.StatusOK, nil
}

// Middleware to set client detail
func ClientMiddleware(req *Request, res *Response) (int, error) {
	r := req.HttpRequest
	protoList := strings.Split(strings.ToLower(r.Proto), "/")
	hostList := strings.Split(strings.ToLower(r.Host), ":")
	proto := protoList[0]
	host := hostList[0]
	port := ""
	ipAddress := helper.GetClientIP(r)
	origin := r.Header.Get("origin")
	altProto := r.Header.Get("X-Forwarded-Proto")

	if len(hostList) > 1 {
		port = hostList[len(hostList)-1]
	}

	if helper.IsNotEmpty(altProto) {
		proto = altProto
	}

	if helper.IsEmpty(origin) {
		referer := r.Header.Get("referer")
		origin = helper.ExtractDomain(referer)

		if helper.IsEmpty(origin) {
			origin = "" + host
		}
	}

	origin = proto + "://" + helper.ExtractDomain(origin)

	client := ClientPayload{
		Host:      host,
		Proto:     proto,
		Port:      port,
		Path:      r.URL.Path,
		Method:    strings.ToLower(r.Method),
		Origin:    origin,
		UserAgent: r.UserAgent(),
		IpAddress: ipAddress,
	}

	req.SetContext(string(ClientPayloadKey), client)

	return http.StatusOK, nil
}

func TenantCatchMiddleware(req *Request, res *Response) (int, error) {
	tenantModelName := "Tenant"
	client := *req.Client()
	host := client.OriginDomain()
	// console.Log("host", client)

	var tenantId interface{}

	if req.App.Config.HasTenant {
		tenant := req.App.ModelQuery(tenantModelName).SkipTenant().SkipBeforeCommit().FindOne(datatype.DataMap{
			"OR": []datatype.DataMap{
				{"domain": host},
				{"subdomain": host},
				{"customDomain": host},
				{"customSubdomain": host},
			},
		})

		if helper.IsNotEmpty(tenant) {
			tenantId = helper.GetValueOfString(tenant, "_id")
		}

		if helper.IsEmpty(tenantId) && req.App.Config.TenantOnly {
			return http.StatusBadRequest, errors.New("Tenant not found for the request")
		}

		if helper.IsNotEmpty(tenantId) {
			req.SetTenantId(tenantId)

			tenantConfig := req.App.GetTenantConfig(req)
			// console.Log("tenantConfig", tenantConfig)

			if helper.IsNotEmpty(tenantConfig) {
				req.SetTenant(*tenantConfig)
			}

		}
	} else if req.App.Config.HasTenantCatch {
		tenant, err := req.App.FetchTenantByDomain(host, req, res)

		if err == nil {
			if helper.IsNotEmpty(tenant) {
				tenantId = helper.GetValueOf(tenant, "tenantId")
			}
		}
	}

	if req.App.Config.HasTenant || req.App.Config.HasTenantCatch {
		mainDomain := helper.GetMainDomain(host)
		// console.Error("Tenant", "host:", host, "mainDomain:", mainDomain)

		if helper.IsNotEmpty(mainDomain) {
			if host != *mainDomain {
				if helper.IsEmpty(tenantId) {
					return http.StatusNotFound, errors.New("Tenant not found")
				}
			}
		}
	}

	if helper.IsNotEmpty(tenantId) {
		client.TenantId = tenantId

		req.SetTenantId(tenantId)
		req.SetContext(string(ClientPayloadKey), client)
	}

	return http.StatusOK, nil
}

// Middleware to add token as a string
func TokenMiddleware(req *Request, res *Response) (int, error) {
	app := req.App
	config := req.App.Config
	clientPayload := req.Client()
	moduleName := req.Param("moduleName")
	masterKey := req.GetContext(string(MasterKey))
	domain := clientPayload.OriginDomain()
	isJson := (strings.Contains(res.request.GetHeader("content-type"), "json"))

	if !isJson {
		isJson = (strings.Contains(res.request.GetHeader("accept"), "json"))
	}

	currentPath := req.HttpRequest.URL.Path
	ignorePaths := []string{
		app.AppendBaseUrl("me"),
		app.AppendBaseUrl("logout"),
		app.AppendBaseUrl("refresh"),
		app.AppendBaseUrl("languages"),
		app.AppendBaseUrl("excel-to-csv"),
		app.AppendBaseUrl("upload"),
		app.AppendBaseUrl("upload-files"),
		app.AppendBaseUrl("config/data"),
		app.AppendBaseUrl("config/report"),
		app.AppendBaseUrl("permissions"),
		app.AppendBaseUrl("config"),
		app.AppendBaseUrl("tenant"),
		app.AppendBaseUrl("tenant-config"),
		app.AppendBaseUrl("theme.css"),
		app.AppendBaseUrl("custom-style.css"),
		app.AppendBaseUrl(config.RestApi),
		app.AppendBaseUrl(config.RestAuthApi),
		app.AppendBaseUrl(config.Graphql.ApiRoute),
		app.AppendBaseUrl(config.Graphql.ApiAuthRoute),
	}
	mandatoryValidToken := (!helper.Contains(ignorePaths, currentPath) &&
		!strings.HasPrefix(currentPath, app.AppendBaseUrl("me/")) &&
		!strings.HasPrefix(currentPath, app.AppendBaseUrl("refresh/")) &&
		!strings.HasPrefix(currentPath, app.AppendBaseUrl("download/")) &&
		!strings.HasPrefix(currentPath, app.AppendBaseUrl("translations/")))

	var isValid bool
	var accessToken string
	var refreshToken string
	var tokenPayloadMap datatype.DataMap
	var tokenPayload TokenPayload

	accessCookie, err := req.HttpRequest.Cookie(string(AccessTokenKey))
	if err == nil {
		accessToken = accessCookie.Value
	}

	refreshCookie, err := req.HttpRequest.Cookie(string(RefreshTokenKey))
	if err == nil {
		refreshToken = refreshCookie.Value
	}

	if helper.IsEmpty(accessToken) {
		var values []string = req.HttpRequest.Header["Authorization"]

		if len(values) > 0 {
			accessToken = strings.Trim(strings.Split(values[0], "Bearer")[1], " ")
		}
	}

	logoutUrl := "logout"
	if helper.IsNotEmpty(moduleName) {
		logoutUrl = "logout/" + moduleName
	}

	if helper.IsNotEmpty(accessToken) {
		isValid, tokenPayloadMap = jwt.DecodeJWT(accessToken, config.Authentication.SecretToken)

		if !isValid || helper.IsEmpty(tokenPayloadMap) {
			if mandatoryValidToken {
				if !isJson {
					return http.StatusTemporaryRedirect, errors.New(helper.GetBaseUrl(logoutUrl, domain))
				}

				return http.StatusTemporaryRedirect, errors.New("Access token invalid")
			}
		}

		json.Unmarshal([]byte(helper.ToJson(tokenPayloadMap)), &tokenPayload)

		if tokenPayload.ExpiresAt.Before(helper.GetTimestamp(nil)) {
			if mandatoryValidToken {
				if !isJson {
					return http.StatusTemporaryRedirect, errors.New(helper.GetBaseUrl("refresh", domain))
				}

				return http.StatusUnauthorized, errors.New("Token expired")
			}
		}

		if domain != tokenPayload.Domain && helper.IsNotEmpty(tokenPayload.Domain) {
			if !isJson {
				return http.StatusTemporaryRedirect, errors.New(helper.GetBaseUrl(logoutUrl, domain))
			}

			return http.StatusUnauthorized, errors.New("Domain mismatch expired")
		}

		if req.App.Config.TenantOnly {
			if helper.IsNotEmpty(tokenPayload) && helper.IsEmpty(tokenPayload.TenantId) {
				return http.StatusBadRequest, errors.New("tenant not found for the request")
			}
		}

		req.SetContext(string(AccessTokenKey), accessToken)
		req.SetContext(string(TokenPayloadKey), tokenPayload)

		if helper.IsNotEmpty(tokenPayload.TenantId) && helper.IsEmpty(req.TenantId()) {
			req.SetTenantId(tokenPayload.TenantId)
		}
	}

	if helper.IsNotEmpty(refreshToken) {
		req.SetContext(string(RefreshTokenKey), refreshToken)
	}

	if config.AuthorizedOnly {
		if !(helper.IsNotEmpty(masterKey) && masterKey == config.MasterKey) && (mandatoryValidToken && helper.IsEmpty(accessToken)) {
			if isJson {
				return http.StatusUnauthorized, errors.New("Must be authorized/login")
			}
		}
	}

	return http.StatusOK, nil
}

// Middleware to add user info as a map
func UserInfoMiddleware(req *Request, res *Response) (int, error) {
	var userInfo *datatype.DataMap
	const userModelName = "User"
	tokenPayload := req.GetContext(string(TokenPayloadKey))

	// console.Error("tokenPayload", tokenPayload)

	if helper.IsNotEmpty(tokenPayload) {
		if payload, ok := tokenPayload.(TokenPayload); ok {
			id := payload.UserId

			if helper.IsNotEmpty(id) {
				if req.App.Config.IsAuthorizationServer {
					userInfo = req.App.ModelQuery(userModelName).SkipBeforeCommit().FindOne(datatype.DataMap{
						"_id": helper.ObjectID(id),
					})
				} else {
					data := helper.ToDataMap(payload.ToMap())
					data["_id"] = id
					data["id"] = id

					userInfo = &data
				}

			}
		}
	}

	// console.Error("userInfo", userInfo)

	if helper.IsNotEmpty(userInfo) {
		req.SetContext(string(UserInfoPayloadKey), *userInfo)
	}

	return http.StatusOK, nil
}

func ApplicationIDMiddleware(req *Request, res *Response) (int, error) {
	if req.App.Config.EnableAppKey {

		var appKey string = req.GetHeader("application-key")
		var path string = req.HttpRequest.URL.Path
		var extension string = filepath.Ext(path)
		var allowed = []string{
			".css",
			".js",
			".ico",
			".pdf",
			".flv",
			".jpg",
			".jpeg",
			".png",
			".gif",
			".webp",
			".woff2",
			".woff",
			".ttf",
			".eot",
		}

		if helper.IsEmpty(appKey) {
			appKey = req.GetHeader("X-Core-Application-Id")
		}

		if !helper.Contains(allowed, extension) {
			if helper.IsEmpty(appKey) {
				appKey = req.Query("application-key")

				if helper.IsEmpty(appKey) {
					appKey = req.Query("app-key")
				}
			}

			if helper.IsNotEmpty(appKey) {
				if appKey != req.App.Config.AppKey {
					return http.StatusUnauthorized, errors.New("application key invalid")
				}
			} else {
				return http.StatusUnauthorized, errors.New("application key not provided")
			}
		}
	}

	return http.StatusOK, nil
}

func MasterKeyMiddleware(req *Request, res *Response) (int, error) {
	var masterKey string = req.GetHeader("master-key")

	if helper.IsEmpty(masterKey) {
		masterKey = req.GetHeader("X-Core-Master-Key")
	}

	if helper.IsEmpty(masterKey) {
		masterKey = req.Query("master-key")
	}

	if helper.IsNotEmpty(masterKey) {
		req.SetContext(string(MasterKey), masterKey)

		if masterKey != req.App.Config.MasterKey {
			return http.StatusUnauthorized, errors.New("master key invalid")
		}
	}

	return http.StatusOK, nil
}
