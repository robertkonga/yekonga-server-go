package Yekonga

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/robertkonga/yekonga-server/helper"
	"github.com/robertkonga/yekonga-server/helper/logger"
	"github.com/robertkonga/yekonga-server/plugins/graphql"
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

	y.All(y.Config.Graphql.ApiAuthRoute, func(req *Request, res *Response) {
		requestString := req.Query("query")
		requestBody := req.Body()
		variableValues := map[string]interface{}{}
		operationName := ""

		graphqlContext := RequestContext{
			Auth:         req.Auth(),
			App:          y,
			Request:      req,
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

	y.All(y.Config.Graphql.ApiRoute, func(req *Request, res *Response) {
		requestString := req.Query("query")
		requestBody := req.Body()
		variableValues := map[string]interface{}{}
		operationName := ""

		graphqlContext := RequestContext{
			Auth:         req.Auth(),
			App:          y,
			Request:      req,
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

func NewDatabaseStructure(file string) *DatabaseStructureType {
	if !helper.FileExists(file) {
		file = helper.GetPath(file)
	}

	var databaseStructure = DefaultDatabaseStructure
	var extraDatabaseStructure map[string]map[string]map[string]interface{}
	data, err := helper.LoadJSONFile(file)

	if err != nil {
		fmt.Println(err)
	}

	json.Unmarshal(helper.ToByte(data), &extraDatabaseStructure)

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
