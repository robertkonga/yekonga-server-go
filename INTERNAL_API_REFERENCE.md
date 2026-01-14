# YekongaServer - Internal API Reference

Complete reference for internal APIs and internal-use structures.

## Table of Contents

1. [Core Server APIs](#core-server-apis)
2. [Request/Response APIs](#requestresponse-apis)
3. [Data Model APIs](#data-model-apis)
4. [Query Builder APIs](#query-builder-apis)
5. [Cloud Functions APIs](#cloud-functions-apis)
6. [Context APIs](#context-apis)
7. [Middleware APIs](#middleware-apis)
8. [Database Connection APIs](#database-connection-apis)

---

## Core Server APIs

### YekongaData - Main Server Instance

#### Initialization

```go
// Initialize and return new server
app := yekonga.ServerConfig(configFile, databaseFile)

// Initialize and set global Server variable
yekonga.ServerLoad(configFile, databaseFile)

// Access global server
app := yekonga.Server
```

#### Routing Methods

```go
// Register GET route
app.Get(path string, handler Handler)

// Register POST route
app.Post(path string, handler Handler)

// Register PUT route
app.Put(path string, handler Handler)

// Register PATCH route
app.Patch(path string, handler Handler)

// Register DELETE route
app.Delete(path string, handler Handler)

// Register OPTIONS route
app.Options(path string, handler Handler)

// Register HEAD route
app.Head(path string, handler Handler)
```

Example with path parameters:

```go
// Route with parameters
app.Get("/users/:id", func(req *yekonga.Request, res *yekonga.Response) {
    id := req.Param("id")
    res.Json(map[string]string{"id": id})
})
```

#### Middleware Management

```go
// Register global middleware (executes for all requests)
app.Use(middleware Middleware)

// Register typed middleware
app.Middleware(middleware Middleware, middlewareType MiddlewareType)

// Available middleware types:
// - yekonga.GlobalMiddleware (default)
// - yekonga.InitMiddleware
// - yekonga.PreloadMiddleware
```

#### Static File Serving

```go
app.Static(config StaticConfig)

// StaticConfig structure:
type StaticConfig struct {
    Directory   string   // Root directory
    PathPrefix  string   // URL prefix (e.g., "/public")
    IndexFile   string   // Default file (e.g., "index.html")
    Extensions  []string // Allowed extensions
    CacheMaxAge int      // Cache seconds
}
```

#### Server Lifecycle

```go
// Start server on port
app.Start(address string)  // e.g., ":8080"

// Stop server
app.Stop()

// Get application home directory
homePath := app.HomeDirectory()  // returns: "/path/to/app"

// Get application base path
basePath := app.ApplicationPath()

// Get static assets path
assetsPath := app.AssetsPath()  // returns: "/path/to/app/assets"
```

#### Model Management

```go
// Get data model by name
model := app.Model("User")  // returns: *DataModel

// Access all models
models := app.Models()  // returns: map[string]*DataModel

// Create query builder for model
query := app.ModelQuery("User")  // returns: *ModelQuery
```

#### Configuration Access

```go
// Get configuration
config := app.Config  // returns: *config.YekongaConfig

// Access config properties
appName := app.Config.AppName
environment := app.Config.Environment
debug := app.Config.Debug
protocol := app.Config.Protocol
domain := app.Config.Domain
```

#### Cloud Functions

```go
// Define cloud function
app.Define(name string, function CloudFunction)

// Define GraphQL action
app.RegisterGraphqlAction(modelName, actionName string, function ActionCloudFunction)

// Database triggers
app.BeforeCreate(modelName string, function TriggerCloudFunction)
app.AfterCreate(modelName string, function TriggerCloudFunction)
app.BeforeUpdate(modelName string, function TriggerCloudFunction)
app.AfterUpdate(modelName string, function TriggerCloudFunction)
app.BeforeDelete(modelName string, function TriggerCloudFunction)
app.AfterDelete(modelName string, function TriggerCloudFunction)
app.BeforeFind(modelName string, function TriggerCloudFunction)
app.AfterFind(modelName string, function TriggerCloudFunction)
```

#### Cron Jobs

```go
// Register cron job with interval
app.RegisterCronjob(name string, interval time.Duration, handler CronjobHandler)

// Register cron job at specific time
app.RegisterCronjobAt(name string, frequency JobFrequency, startTime time.Time, handler CronjobHandler)

// Available frequencies:
// - yekonga.JobFrequencyMinute
// - yekonga.JobFrequencyHourly
// - yekonga.JobFrequencyDaily
// - yekonga.JobFrequencyWeekly
// - yekonga.JobFrequencyMonthly
```

---

## Request/Response APIs

### Request Object

#### URL and Method Information

```go
// Get HTTP method
method := req.Method()  // returns: "GET", "POST", etc.

// Get request URL/path
url := req.URL()  // returns: "/api/users/123?filter=active"

// Get request path (without query string)
path := req.Path()  // returns: "/api/users/123"

// Get hostname
host := req.Host()  // returns: "example.com"

// Get protocol
protocol := req.Protocol()  // returns: "https"

// Get remote address
remoteAddr := req.RemoteAddr()  // returns: "192.168.1.1"
```

#### Headers

```go
// Get single header
value := req.Header("Content-Type")  // returns: "application/json" or ""

// Get all headers
headers := req.Headers()  // returns: map[string][]string

// Check header exists
if req.Header("Authorization") != "" {
    // Header present
}
```

#### Parameters

```go
// Get URL path parameter
id := req.Param("id")  // from route /users/:id

// Get query string parameter
filter := req.Query("filter")  // from /users?filter=active

// Get form/body parameter
username := req.FormValue("username")

// Get all query parameters
params := req.QueryParams()  // returns: url.Values
```

#### Body Parsing

```go
// Get body as map
data := req.Body()  // returns: map[string]interface{}

// Get raw body bytes
rawBody := req.RawBody()  // returns: []byte

// Get body as string
bodyStr := req.BodyString()  // returns: string

// Parse JSON body to struct
var user User
json.Unmarshal(req.RawBody(), &user)
```

#### Authentication

```go
// Get current user from token
auth := req.Auth()  // returns: *AuthPayload or nil

// Get bearer token
token := req.Token()  // returns: "eyJhbGc..."

// Check if authenticated
if auth != nil {
    userId := auth.ID
    role := auth.Role
}
```

#### Context Management

```go
// Store value in request context
req.SetContext("key", "value")

// Retrieve value from context
value := req.GetContext("key")  // returns: interface{}

// Store object in context
req.SetContextObject("userData", userData)

// Retrieve object from context
userData := req.GetContextObject("userData")  // returns: interface{}
```

#### Utilities

```go
// Get original http.Request
httpReq := req.HttpRequest  // returns: *http.Request

// Get application instance
app := req.App  // returns: *YekongaData

// Get client info
client := req.ClientInfo()  // returns: *ClientPayload

// Get request ID (for logging)
requestId := req.RequestID()
```

### Response Object

#### Status and Headers

```go
// Set HTTP status code
res.Status(200)

// Set response header
res.Header("Content-Type", "application/json")

// Set multiple headers
res.Headers(map[string]string{
    "X-Custom": "value",
    "Cache-Control": "no-cache",
})

// Set cookie
res.SetCookie(&http.Cookie{
    Name: "sessionId",
    Value: "abc123",
})
```

#### Response Body

```go
// Send JSON response
res.Json(map[string]interface{}{
    "status": "success",
    "data": user,
})

// Send plain text
res.Text("Hello World")

// Send HTML
res.Html("<h1>Hello</h1>")

// Send file
res.File("path/to/file.pdf")

// Send file with custom name
res.Download("path/to/file.pdf", "myfile.pdf")
```

#### Error Responses

```go
// Send error response (sets status code)
res.Error("User not found", 404)

// Send error with details
res.Error("Validation failed", 400)  // status 400
res.Error("Unauthorized", 401)       // status 401
res.Error("Forbidden", 403)          // status 403
res.Error("Server error", 500)       // status 500
```

#### Redirects

```go
// Redirect to URL
res.Redirect("https://example.com")

// Redirect with status
res.Redirect("https://example.com/new-path")  // 302 default
```

#### Streaming

```go
// Flush response (for streaming)
res.Flush()

// Write bytes directly
res.Write([]byte("data"))

// Write string directly
res.WriteString("data")
```

---

## Data Model APIs

### DataModel Structure

```go
type DataModel struct {
    // Identity
    Name            string
    Class           string
    Collection      string
    
    // Naming
    Variable        string
    VariableSingle  string
    VariablePlural  string
    
    // Keys
    PrimaryKey      string
    PrimaryName     string
    ForeignKey      string
    
    // Field categories
    Required        []string
    Protected       []string
    DateFields      []string
    BooleanFields   []string
    NumberFields    []string
    FloatFields     []string
    FileFields      []string
    OptionFields    []string
    ValidFields     []string
    
    // Relationships
    ParentKeys      []string
    
    // References
    Fields          map[string]*DataModelField
    App             *YekongaData
    Config          *config.YekongaConfig
    DBConnect       *DatabaseConnections
}
```

### DataModelField Structure

```go
type DataModelField struct {
    PrimaryKey      bool
    Name            string
    Kind            DataModelFieldType
    Required        bool
    Protected       bool
    DefaultValue    interface{}
    ID              bool
    
    ForeignKey      DataModelFieldForeignKey
    Options         []DataModelFieldOptions
}

type DataModelFieldForeignKey struct {
    Model           *DataModel
    ModelName       string
    PrimaryKey      string
    ForeignKey      string
}

type DataModelFieldOptions struct {
    Value           string
    Label           string
}
```

### Model Access Methods

```go
// Get field from model
field := model.Fields["username"]  // returns: *DataModelField

// Check field type
if field.Kind == yekonga.DataModelString {
    // String field
}

// Check field constraints
if field.Required {
    // Field is required
}

// Get primary key field
pkField := model.Fields[model.PrimaryKey]
```

---

## Query Builder APIs

### ModelQuery Interface

#### Chainable Methods

```go
// Filter by field value and operator
query.Where(field, operator, value) *ModelQuery

// Operators supported:
// "=" (equals)
// "!=" (not equals)
// ">" (greater than)
// ">=" (greater than or equal)
// "<" (less than)
// "<=" (less than or equal)
// "contains" (substring)
// "starts" (starts with)
// "ends" (ends with)
// "in" (in array)
// "notIn" (not in array)

// Sort results
query.OrderBy(field, direction) *ModelQuery
// Directions: "asc" or "desc"

// Limit results
query.Take(limit int) *ModelQuery

// Skip results
query.Skip(offset int) *ModelQuery

// Select specific fields
query.Select(fields ...string) *ModelQuery

// Group by field
query.GroupBy(field string) *ModelQuery

// Include related data
query.With(relations ...string) *ModelQuery

// Distinct values
query.Distinct(field string) *ModelQuery
```

#### Execution Methods

```go
// Execute query and return all results
results := query.Find(selectors map[uint][]string) *[]datatype.DataMap

// Find single record
record := query.FindOne(filter datatype.DataMap) *datatype.DataMap

// Find by primary key
record := query.FindById(id interface{}) *datatype.DataMap

// Count matching records
count := query.Count() int64

// Get first field value
firstValue := query.First(field string) interface{}

// Get last field value
lastValue := query.Last(field string) interface{}

// Get distinct field values
values := query.Pluck(field string) []interface{}
```

#### Aggregation Methods

```go
// Sum of field values
total := query.Sum(field string) float64

// Average of field values
avg := query.Average(field string) float64

// Maximum value
max := query.Max(field string) interface{}

// Minimum value
min := query.Min(field string) interface{}

// Count occurrences
count := query.Count() int64

// Get aggregation summary
summary := query.Summary() *datatype.DataMap
// Returns: {"count": 100, "sum": 500, "avg": 5, "max": 50, "min": 1}
```

#### CRUD Methods

```go
// Create single record
created, err := query.Create(data datatype.DataMap) (*datatype.DataMap, error)

// Create multiple records
created, err := query.CreateMany(data []datatype.DataMap) (*[]datatype.DataMap, error)

// Update single record
updated, err := query.Update(id interface{}, data datatype.DataMap) (*datatype.DataMap, error)

// Update multiple records
updated, err := query.UpdateMany(data datatype.DataMap) (*[]datatype.DataMap, error)

// Delete single record
deleted, err := query.Delete(id interface{}) (interface{}, error)

// Delete multiple records (with where conditions)
query.Where("status", "=", "inactive")
deleted, err := query.DeleteMany() (interface{}, error)
```

#### Pagination

```go
// Get paginated results
paginated := query.Paginate(page, perPage int) *datatype.DataMap
// Returns: {"data": [...], "current_page": 1, "per_page": 10, "total": 100}

// Get pagination info
paginationInfo := query.Pagination() *datatype.DataMap
```

#### Graph/Chart Data

```go
// Get data for charting
graphData := query.Graph() *datatype.DataMap
// Returns grouped and aggregated data for charts

// Get data grouped by field
grouped := query.GroupBy("status").Find(nil)
```

---

## Cloud Functions APIs

### Function Definition

```go
// Define basic cloud function
app.Define("functionName", func(data interface{}, rc *yekonga.RequestContext) (interface{}, error) {
    // Process data
    result := processData(data)
    return result, nil
})

// Call cloud function from handler
result, err := app.CallFunction("functionName", inputData)
```

### RequestContext in Cloud Functions

```go
type RequestContext struct {
    Auth          *AuthPayload      // Current user
    App           *YekongaData      // Server instance
    Request       *Request          // Original request
    Client        *ClientPayload    // Client info
    TokenPayload  *TokenPayload     // JWT claims
    QuerySelectors map[uint][]string
    QueryRelatedData datatype.JsonObject
    QueryWhereData   datatype.JsonObject
}
```

### Database Triggers

```go
// Before create
app.BeforeCreate("User", func(rc *yekonga.RequestContext, qc *yekonga.QueryContext) (interface{}, error) {
    // qc.Data = data to be inserted
    // Modify or validate data
    return qc.Data, nil
})

// After create
app.AfterCreate("User", func(rc *yekonga.RequestContext, qc *yekonga.QueryContext) (interface{}, error) {
    // qc.Data = newly created record
    // Send notification, log event, etc.
    return qc.Data, nil
})

// Before update
app.BeforeUpdate("User", func(rc *yekonga.RequestContext, qc *yekonga.QueryContext) (interface{}, error) {
    // Validate update data
    return qc.Data, nil
})

// After update
app.AfterUpdate("User", func(rc *yekonga.RequestContext, qc *yekonga.QueryContext) (interface{}, error) {
    // qc.Data = updated record
    // qc.PreviousData = old record (if available)
    return qc.Data, nil
})

// Similarly for: BeforeDelete, AfterDelete, BeforeFind, AfterFind
```

### QueryContext Structure

```go
type QueryContext struct {
    Data         interface{}              // Record data
    PreviousData interface{}              // Old data (for updates)
    Model        *DataModel               // Model definition
    Selection    []string                 // Selected fields
    RelatedData  datatype.JsonObject     // Joined data
}
```

### GraphQL Actions

```go
app.RegisterGraphqlAction("User", "activate", 
    func(rc *yekonga.RequestContext, qc *yekonga.QueryContext) (yekonga.GraphqlActionResult, error) {
        // qc.Data contains action parameters
        
        return yekonga.GraphqlActionResult{
            Data:    result,
            Success: true,
            Status:  true,
            Message: "User activated",
        }, nil
    },
)
```

---

## Context APIs

### RequestContext Methods

```go
// Set value in context
ctx.SetContext("key", "value")

// Get value from context
value := ctx.GetContext("key")  // returns: interface{}

// Set object in context
ctx.SetContextObject("userData", userData)

// Get object from context
userData := ctx.GetContextObject("userData")

// Clear context
ctx.ClearContext()

// Get context data
contextData := ctx.GetContextData()  // returns: map[string]interface{}
```

### Common Context Keys

```go
// Token and authentication
TokenKey           // Bearer token
TokenPayloadKey    // JWT claims
UserInfoPayloadKey // Current user data
ClientPayloadKey   // Client information

// Query building
QuerySelectorsKey      // GraphQL field selection
QueryRelatedDataKey    // Joined data
QueryWhereDataKey      // Filter conditions
```

---

## Middleware APIs

### Middleware Function Signature

```go
type Middleware func(*Request, *Response) error

// Implementation example
func MyMiddleware(req *yekonga.Request, res *yekonga.Response) error {
    // Process request
    if err := validateRequest(req); err != nil {
        res.Error("Invalid request", 400)
        return err
    }
    
    // Store data for handler
    req.SetContext("validated", true)
    
    // Return nil to continue, error to stop
    return nil
}
```

### Middleware Registration

```go
// Global middleware (all requests)
app.Use(MyMiddleware)

// Init middleware (before route handler)
app.Middleware(MyMiddleware, yekonga.InitMiddleware)

// Preload middleware (before route handler, after init)
app.Middleware(MyMiddleware, yekonga.PreloadMiddleware)
```

### Built-in Middleware Functions

```go
// Token extraction and validation
yekonga.TokenMiddleware(req, res)

// User info loading
yekonga.UserInfoMiddleware(req, res)

// App key validation
yekonga.ApplicationKeyMiddleware(req, res)

// CORS handling
yekonga.CORSMiddleware(req, res)

// Request logging
yekonga.LoggingMiddleware(req, res)
```

---

## Database Connection APIs

### DatabaseConnections Methods

```go
// Get MongoDB client
client := dbConnect.GetMongoClient() *mongo.Client

// Get local DB client
localDb := dbConnect.GetLocalClient() *localDB.DB

// Check connection status
isConnected := dbConnect.IsConnected() bool

// Close all connections
err := dbConnect.Close()

// Reconnect to database
err := dbConnect.Reconnect()
```

### Connection Configuration

```go
// Configuration structure
type DatabaseConfig struct {
    Kind         string  // "mongodb", "mysql", "sql", "local"
    Host         string  // Database host
    Port         string  // Database port
    DatabaseName string  // Database/schema name
    Username     string  // Connection username
    Password     string  // Connection password
    Options      string  // Connection options
}

// Access from config
dbConfig := app.Config.Database
dbType := dbConfig.Kind
dbHost := dbConfig.Host
```

---

## Data Types Reference

### DataMap

```go
// Flexible key-value storage
type DataMap map[string]interface{}

// Usage
var data yekonga.DataMap = make(yekonga.DataMap)
data["username"] = "john_doe"
data["age"] = 25
data["email"] = "john@example.com"

// Conversion helpers
helper.ToJson(data)      // Convert to JSON string
helper.ToMap(data)       // Convert to map
```

### Response Structures

```go
// Standard API response
type ApiResponse struct {
    Status  bool        `json:"status"`
    Message string      `json:"message"`
    Data    interface{} `json:"data"`
}

// Error response
type ErrorResponse struct {
    Status  bool   `json:"status"`
    Message string `json:"message"`
    Code    int    `json:"code"`
}

// Paginated response
type PaginatedResponse struct {
    Data        interface{} `json:"data"`
    Total       int64       `json:"total"`
    Page        int         `json:"page"`
    PerPage     int         `json:"per_page"`
    TotalPages  int         `json:"total_pages"`
}
```

---

## Complete Request/Response Example

```go
app.Post("/api/users", func(req *yekonga.Request, res *yekonga.Response) {
    // Get authenticated user
    auth := req.Auth()
    if auth == nil {
        res.Error("Unauthorized", 401)
        return
    }
    
    // Parse request body
    data := req.Body()
    
    // Validate data
    if data["username"] == "" {
        res.Error("Username is required", 400)
        return
    }
    
    // Create user
    user, err := req.App.ModelQuery("User").Create(data)
    if err != nil {
        res.Error("Failed to create user", 500)
        return
    }
    
    // Set response headers
    res.Header("Content-Type", "application/json")
    res.Status(201)
    
    // Send response
    res.Json(map[string]interface{}{
        "status": "success",
        "data": user,
    })
})
```

---

## Index of All APIs by Module

| Module | Key APIs |
|--------|----------|
| **Server** | Get, Post, Put, Delete, Use, Define, Start, ModelQuery |
| **Request** | Param, Query, Body, Auth, Token, Header, SetContext |
| **Response** | Json, Error, Status, Header, Redirect, File |
| **DataModel** | Fields, PrimaryKey, Required, Protected |
| **Query** | Where, OrderBy, Take, Skip, Find, Create, Update |
| **Middleware** | Use, Middleware, error handling |
| **Context** | SetContext, GetContext, SetContextObject |
| **Cloud Functions** | Define, BeforeCreate, AfterCreate, RegisterGraphqlAction |

This reference is the source of truth for internal API usage. Refer to INTERNAL_ARCHITECTURE.md for conceptual understanding.
