package yekonga

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/robertkonga/yekonga-server-go/config"
	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/gateway/setting"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/helper/logger"
)

const COOKIE_ENABLED_KEY = "YEKONGA_ENABLED"

// StaticConfig holds configuration for static file serving
type StaticConfig struct {
	Directory   string   // Root directory for static files
	PathPrefix  string   // URL path prefix for static files (e.g., "/public")
	IndexFile   string   // Default index file (e.g., "index.html")
	Extensions  []string // Allowed file extensions
	CacheMaxAge int      // Cache max age in seconds
}

// Route represents a route with its pattern and parameters
type Route struct {
	pattern    string
	paramNames []string
	handler    Handler
}

// Handler represents a function that handles an HTTP request
type Handler func(req *Request, res *Response)
type SystemHandler func(req *Request, res *Response) (interface{}, error)

// Middleware represents a function that can modify requests/responses
type Middleware func(req *Request, res *Response) error

type CloudFunction func(interface{}, *RequestContext) (interface{}, error)
type BackendCloudFunction func(interface{}) (*setting.SendResponse, error)
type TriggerFunction func(*RequestContext, *QueryContext) (interface{}, error)
type TriggerAllFunction func(*DataModel, *RequestContext, *QueryContext) (interface{}, error)
type ActionCloudFunction func(*RequestContext, *QueryContext) (GraphqlActionResult, error)

var Server *YekongaData

// Yekonga represents the main server structure
type YekongaData struct {
	routes                 map[string][]Route // method -> routes
	functions              map[string]CloudFunction
	systemFunctions        map[string]SystemHandler
	primaryFunctions       map[PrimaryCloudKey]BackendCloudFunction
	authTriggerFunctions   map[TriggerAction]TriggerFunction
	triggerFunctions       map[string]map[TriggerAction]map[string]TriggerFunction
	triggerAllFunctions    map[TriggerAction]TriggerAllFunction
	graphqlActionFunctions map[string]map[string]map[string]ActionCloudFunction
	middlewares            []Middleware
	initMiddlewares        []Middleware
	preloadMiddlewares     []Middleware
	models                 map[string]*DataModel
	resolverChartGroupData map[string]ResolverChartGroupData
	databaseStructure      *DatabaseStructureType
	graphqlBuild           *GraphqlAutoBuild
	Config                 *config.YekongaConfig
	socketServer           *SocketServer
	dbConnect              *DatabaseConnections
	staticConfig           []*StaticConfig
	logger                 *log.Logger
	cronjob                *Cronjob
	mut                    sync.RWMutex
}

// NewYekonga creates a new instance of Yekonga server
func ServerConfig(configFile string, databaseFile string) *YekongaData {
	logger.Logo()

	config := config.NewYekongaConfig(configFile)
	databaseStructure := NewDatabaseStructure(databaseFile, config)
	systemModels := NewSystemModels(config, databaseStructure)
	dbConnect := NewDatabaseConnections(config)
	resolverChartGroupData := SetDataGroups(systemModels)

	Server = &YekongaData{
		Config:                 config,
		dbConnect:              dbConnect,
		models:                 systemModels,
		resolverChartGroupData: resolverChartGroupData,
		databaseStructure:      databaseStructure,
		routes:                 make(map[string][]Route),
		middlewares:            make([]Middleware, 0, 5),
		initMiddlewares:        make([]Middleware, 0, 5),
		preloadMiddlewares:     make([]Middleware, 0, 5),
		functions:              make(map[string]CloudFunction),
		systemFunctions:        make(map[string]SystemHandler),
		primaryFunctions:       make(map[PrimaryCloudKey]BackendCloudFunction),
		graphqlActionFunctions: make(map[string]map[string]map[string]ActionCloudFunction),
		triggerFunctions:       make(map[string]map[TriggerAction]map[string]TriggerFunction),
		authTriggerFunctions:   make(map[TriggerAction]TriggerFunction),
		logger:                 &log.Logger{},
	}

	dbConnect.appPath = Server.HomeDirectory()
	SetSystemModelDBconnection(Server, &systemModels)
	graphqlBuild := NewGraphqlAutoBuild(Server, systemModels)
	graphqlBuild.initialize()

	dbConnect.connect()
	Server.graphqlBuild = graphqlBuild
	Server.initialize()
	Server.cronjob = NewCronjob(Server)
	Server.setNotification()

	return Server
}

func ServerLoad(configFile string, databaseFile string) {
	ServerConfig(configFile, databaseFile)
}

func (y *YekongaData) Model(name string) *DataModel {
	if mod, ok := y.models[name]; ok {
		return mod
	} else {
		logger.Error("" + name + " is not available")
	}

	return nil
}

func (y *YekongaData) ModelQuery(name string) *DataModelQuery {
	model := y.Model(name)

	if model != nil {
		return model.Query()
	}

	return nil
}

func (y *YekongaData) HomeDirectory() string {
	return helper.HomeDirectory(helper.ToSlug(y.Config.AppName))
}

// parseRoute parses a route pattern and extracts parameter names
func parseRoute(pattern string) ([]string, string) {
	parts := strings.Split(pattern, "/")
	params := []string{}
	normalized := []string{}

	for _, part := range parts {
		if strings.HasPrefix(part, ":") {
			params = append(params, part[1:])
			normalized = append(normalized, "([^/]+)")
		} else {
			normalized = append(normalized, part)
		}
	}

	return params, "^" + strings.Join(normalized, "/") + "$"
}

// matchRoute checks if a path matches a route pattern and extracts parameters
func matchRoute(pattern string, paramNames []string, path string) (bool, map[string]string) {
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	if len(patternParts) != len(pathParts) {
		return false, nil
	}

	params := make(map[string]string)
	for i, part := range patternParts {
		if strings.HasPrefix(part, ":") {
			paramName := part[1:]
			params[paramName] = pathParts[i]
		} else if part != pathParts[i] {
			return false, nil
		}
	}

	return true, params
}

// Yekonga methods
func (y *YekongaData) addRoute(method, pattern string, handler Handler) {
	y.mut.Lock()
	defer y.mut.Unlock()

	if y.routes[method] == nil {
		y.routes[method] = []Route{}
	}

	paramNames, _ := parseRoute(pattern)
	pattern = y.AppendBaseUrl(pattern)

	y.routes[method] = append(y.routes[method], Route{
		pattern:    pattern,
		paramNames: paramNames,
		handler:    handler,
	})
}

func (y *YekongaData) AppendBaseUrl(pattern string) string {
	if helper.IsNotEmpty(y.Config.BaseUrl) {
		baseUrl := y.Config.BaseUrl

		if baseUrl != "/" {
			baseUrl = strings.TrimSuffix(baseUrl, "/")
			pattern = strings.TrimPrefix(pattern, "/")

			pattern = baseUrl + "/" + pattern
		}
	}

	return pattern
}

func (y *YekongaData) findRoute(method, path string) (*Route, map[string]string) {
	y.mut.Lock()
	routes := y.routes[method]
	y.mut.Unlock()

	for _, route := range routes {
		if matches, params := matchRoute(route.pattern, route.paramNames, path); matches {
			return &route, params
		}
	}
	return nil, nil
}

func (y *YekongaData) Middleware(middleware Middleware, middlewareType MiddlewareType) {
	switch middlewareType {
	case GlobalMiddleware:
		y.middlewares = append(y.middlewares, middleware)
	case InitMiddleware:
		y.initMiddlewares = append(y.initMiddlewares, middleware)
	case PreloadMiddleware:
		y.preloadMiddlewares = append(y.preloadMiddlewares, middleware)
	default:
		y.logger.Printf("Unknown middlewares type: %s, select between global, init, or preload", middlewareType)
	}
}

func (y *YekongaData) Get(path string, handler Handler) {
	y.addRoute(http.MethodGet, path, handler)
}

func (y *YekongaData) Post(path string, handler Handler) {
	y.addRoute(http.MethodPost, path, handler)
}

func (y *YekongaData) Put(path string, handler Handler) {
	y.addRoute(http.MethodPut, path, handler)
}

func (y *YekongaData) Patch(path string, handler Handler) {
	y.addRoute(http.MethodPatch, path, handler)
}

func (y *YekongaData) Options(path string, handler Handler) {
	y.addRoute(http.MethodOptions, path, handler)
}

func (y *YekongaData) Delete(path string, handler Handler) {
	y.addRoute(http.MethodDelete, path, handler)
}

func (y *YekongaData) All(path string, handler Handler) {
	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodOptions,
		http.MethodDelete,
	}
	for _, method := range methods {
		y.addRoute(method, path, handler)
	}
}

// Static configures static file serving
func (y *YekongaData) Static(config StaticConfig) error {
	if y.staticConfig == nil {
		total := 10
		if y.Config != nil {
			total = len(y.Config.Public)
		}

		y.staticConfig = make([]*StaticConfig, total)
	}

	// Set default values if not provided
	if config.IndexFile == "" {
		config.IndexFile = "index.html"
	}
	if config.Extensions == nil {
		config.Extensions = DefaultExtensions[:]
	}
	if config.CacheMaxAge == 0 {
		config.CacheMaxAge = 86400 // 24 hours
	}

	// Verify directory exists
	if _, err := os.Stat(config.Directory); os.IsNotExist(err) {
		return err
	}

	y.mut.Lock()
	defer y.mut.Unlock()
	y.staticConfig = append(y.staticConfig, &config)
	return nil
}

// isStaticPath checks if the path is for static file serving
func (y *YekongaData) isStaticPath(path string) bool {
	if y.staticConfig == nil {
		y.staticConfig = make([]*StaticConfig, 10)
	}

	count := len(y.staticConfig)
	if count == 0 {
		return false
	}

	for i := 0; i < count; i++ {
		y.mut.RLock()
		static := y.staticConfig[i]
		y.mut.RUnlock()

		if static != nil && helper.Contains(static.Extensions, filepath.Ext(path)) {
			return true
		}
	}

	return false
}

// handleStaticFile serves static files
func (y *YekongaData) handleStaticFile(w http.ResponseWriter, r *http.Request) bool {
	if y.staticConfig == nil {
		y.staticConfig = make([]*StaticConfig, 10)
	}

	count := len(y.staticConfig)
	if count == 0 {
		return false
	}

	for i := 0; i < count; i++ {
		static := y.staticConfig[i]

		if static != nil {
			// Remove path prefix to get relative file path
			urlPath := strings.TrimPrefix(r.URL.Path, static.PathPrefix)
			urlPath = strings.TrimPrefix(urlPath, "/")

			// Construct full file path
			filePath := filepath.Join(static.Directory, urlPath)

			// Check if path is a directory
			fileInfo, err := os.Stat(filePath)
			if err == nil && fileInfo.IsDir() {
				filePath = filepath.Join(filePath, static.IndexFile)
			}

			// Verify file exists and extension is allowed
			if fileInfo, err := os.Stat(filePath); err == nil && !fileInfo.IsDir() {
				ext := strings.ToLower(path.Ext(filePath))
				for _, allowedExt := range static.Extensions {
					if ext == allowedExt {
						// Set cache headers
						w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", static.CacheMaxAge))

						// Serve the file
						http.ServeFile(w, r, filePath)
						return true
					}
				}
			}
		}
	}

	return false
}

func (y *YekongaData) RegisterCronjob(name string, frequency time.Duration, callback func(app *YekongaData, time time.Time)) {
	y.cronjob.registerJob(name, frequency, callback)
}

func (y *YekongaData) RegisterCronjobAt(name string, frequency JobFrequency, time time.Time, callback func(app *YekongaData, time time.Time)) {
	y.cronjob.registerJobAt(name, frequency, time, callback)
}

func (y *YekongaData) RegisterCronjobOn(name string, frequency JobFrequency, time time.Time, callback func(app *YekongaData, time time.Time)) {
	y.cronjob.registerJobAt(name, frequency, time, callback)
}

// ServeHTTP implements the http.Handler interface
func (y *YekongaData) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// logger.Info("ServeHTTP", "All routes pass")
	var err error
	var rawBody interface{}

	// Check for static file requests first
	if y.isStaticPath(r.URL.Path) {
		if y.handleStaticFile(w, r) {
			return
		}
	}

	origin := "*"
	originList := r.Header["Origin"]

	if len(originList) > 0 {
		origin = originList[0]
	} else {
		originList = r.Header["Referer"]

		if len(originList) > 0 {
			origin = originList[0]
		}

		if helper.IsEmpty(origin) {
			origin = r.Host
		}
	}

	if y.Config.Cors {
		w.Header().Add("access-control-allow-origin", origin)
	}

	w.Header().Add("access-control-allow-headers", "content-type, authorization, x-requested-with, x-csrf-token, timezone, upgrade-insecure-requests")
	w.Header().Add("access-control-allow-credentials", "true")
	w.Header().Add("access-control-allow-methods", "GET, POST, OPTIONS, PUT, PATCH, DELETE")
	w.Header().Add("Keep-Alive", "timeout=5, max=98")
	w.Header().Add("Connection", "Keep-Alive")

	cookieValue := y.Config.ConnectionID
	cookie, err := r.Cookie(COOKIE_ENABLED_KEY)
	if helper.IsEmpty(cookieValue) {
		cookieValue = "YEKONGA_CONNECTED"
	}

	if err != nil || helper.IsEmpty(cookie.Value) {
		http.SetCookie(w, &http.Cookie{
			Name:     COOKIE_ENABLED_KEY,
			Value:    cookieValue,
			Domain:   "",
			HttpOnly: true,
			Secure:   y.Config.SecureOnly,
			SameSite: http.SameSiteDefaultMode,
			MaxAge:   30 * 24 * 60 * 60, // 30 days
		})
	}

	json.NewDecoder(r.Body).Decode(&rawBody)

	route, params := y.findRoute(r.Method, r.URL.Path)
	if route == nil {
		var htmlPage []byte
		var err error
		var textPage []byte

		if r.URL.Path == "" || r.URL.Path == "/" || r.URL.Path == "/index.html" {
			textPage = []byte("Welcome to Yekonga Server")
			htmlPage, err = StaticFS.ReadFile("static/index.html")
		} else {
			w.WriteHeader(http.StatusNotFound)
			textPage = []byte("404 Page Not Found")
			htmlPage, err = StaticFS.ReadFile("static/404.html")
		}

		if err != nil {
			// If the file is missing or there's an error, send a default message
			w.Write(textPage)
			return
		}

		w.Write(htmlPage) // Send the custom 404 page content
		return
	}

	req := Request{
		HttpRequest: r,
		RawBody:     rawBody,
		Context:     datatype.Context{},
		Params:      params,
		App:         y,
	}

	res := Response{
		httpResponseWriter: &w,
		staticConfig:       y.staticConfig,
		request:            &req,
	}

	// Apply middlewares
	err = ApplicationKeyMiddleware(&req, &res)
	if err != nil {
		res.Abort(http.StatusForbidden, err.Error())
		return
	}

	// Apply middlewares
	for _, middleware := range y.preloadMiddlewares {
		if middleware != nil {
			err = middleware(&req, &res)
			if err != nil {
				res.Abort(http.StatusBadRequest, err.Error())
				return
			}
		}
	}

	// Apply middlewares
	err = ClientMiddleware(&req, &res)
	if err != nil {
		res.Abort(http.StatusBadRequest, err.Error())
		return
	}

	// Apply middlewares
	err = TokenMiddleware(&req, &res)
	if err != nil {
		res.Abort(http.StatusBadRequest, err.Error())
		return
	}

	// Apply middlewares
	err = UserInfoMiddleware(&req, &res)
	if err != nil {
		res.Abort(http.StatusBadRequest, err.Error())
		return
	}

	// Apply middlewares
	for _, middleware := range y.initMiddlewares {
		if middleware != nil {
			err = middleware(&req, &res)
			if err != nil {
				res.Abort(http.StatusBadRequest, err.Error())
				return
			}
		}
	}

	// Apply middlewares
	for _, middleware := range y.middlewares {
		if middleware != nil {
			err = middleware(&req, &res)
			if err != nil {
				res.Abort(http.StatusBadRequest, err.Error())
				return
			}
		}
	}

	// Execute route handler
	route.handler(&req, &res)
}

// Redirect HTTP to HTTPS
func redirectToHTTPS(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://"+r.Host+r.RequestURI, http.StatusMovedPermanently)
}

func (y *YekongaData) Stop() {
	if y.socketServer != nil {
		y.socketServer.Close()
	}
}

func (y *YekongaData) Start(address interface{}) {
	port := y.Config.Ports.Server

	if y.Config.Ports.Secure {
		port = y.Config.Ports.SSLServer
	}

	if helper.IsNotEmpty(address) {
		port = helper.ToInt(address)
	}

	y.Config.Ports.Server = port

	serverPort := fmt.Sprint(":", port)
	y.socketServer = NewSocketServer(y)

	defer y.socketServer.Close()

	logRunningServer(serverPort, y.Config.Ports.Secure)

	if y.Config.Ports.Secure {
		go func() {
			httpMux := http.NewServeMux()

			httpMux.HandleFunc("/", redirectToHTTPS)

			err := http.ListenAndServe(":"+fmt.Sprint(y.Config.Ports.Server), httpMux)
			if err != nil {
				fmt.Println("HTTP server error:", err)
			}
		}()

		if err := http.ListenAndServeTLS(serverPort, "certificate/cert.pem", "certificate/key.pem", y); err != nil {
			logger.Error("Error starting https server", err)
		}
	} else {
		if err := http.ListenAndServe(serverPort, y); err != nil {
			logger.Error("Error starting server", err)
		}
	}
}

func logRunningServer(port string, secure bool) {
	ips, err := helper.GetLocalIPS()

	if err != nil {
		ips = append(ips, "127.0.0.1")
	}

	for _, ip := range ips {
		serverAddress := fmt.Sprint(ip + port)
		if secure {
			logger.Success("HTTPS Server is running on", serverAddress)
		} else {
			logger.Success("Server is running on", serverAddress)
		}
	}
}
