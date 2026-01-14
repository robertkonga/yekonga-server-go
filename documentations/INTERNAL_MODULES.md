# YekongaServer - Module Documentation

Deep dive into each major module and internal components.

## Table of Contents

1. [Core Module Structure](#core-module-structure)
2. [Request/Response Processing](#requestresponse-processing)
3. [Database Connections](#database-connections)
4. [Model Query System](#model-query-system)
5. [GraphQL Integration](#graphql-integration)
6. [Cloud Functions System](#cloud-functions-system)
7. [Helper Functions](#helper-functions)
8. [Plugin System](#plugin-system)

---

## Core Module Structure

### Package: `yekonga` (Main)

Core server implementation residing in `/yekonga` directory.

#### Key Files

| File | Purpose | Exports |
|------|---------|---------|
| `main.go` | Server initialization and routing | `YekongaData`, `ServerConfig()`, `ServerLoad()` |
| `request.go` | HTTP request abstraction | `Request`, `RequestContext`, `AuthPayload` |
| `response.go` | HTTP response abstraction | `Response` |
| `middleware.go` | Middleware functions | Built-in middleware functions |
| `model.go` | Data model definitions | `DataModel`, `DataModelField` |
| `model_query.go` | Query builder | `ModelQuery`, query execution methods |
| `graphql.go` | GraphQL implementation | `GraphqlAutoBuild` |
| `cloud_functions.go` | Function registry and execution | Function management |
| `cronjob.go` | Scheduled tasks | `Cronjob`, cron registration |
| `database_structure.go` | Schema parsing | `DatabaseStructureType` |
| `dbconnect.go` | Database abstraction | `DatabaseConnections` |
| `dbconnect_mongodb.go` | MongoDB driver | MongoDB-specific implementation |
| `dbconnect_mysql.go` | MySQL driver | MySQL-specific implementation |
| `dbconnect_sql.go` | SQL driver | Generic SQL implementation |
| `dbconnect_local.go` | Local JSON DB | File-based database |

### Initialization Flow

```
ServerConfig(configFile, databaseFile)
├── Load config.json → YekongaConfig
├── Load database.json → DatabaseStructureType
├── Parse models → DataModel objects
├── Create system models → SystemModels
├── Initialize DB connections → DatabaseConnections
├── Build GraphQL schema → GraphqlAutoBuild
├── Create YekongaData instance
├── Set global Server variable
└── Return *YekongaData
```

### Startup Example

```go
// /yekonga/main.go - ServerConfig implementation
func ServerConfig(configFile string, databaseFile string) *YekongaData {
    // 1. Print logo
    logger.Logo()
    
    // 2. Parse database schema
    databaseStructure := NewDatabaseStructure(databaseFile)
    
    // 3. Load configuration
    config := config.NewYekongaConfig(configFile)
    
    // 4. Create system models from schema
    systemModels := NewSystemModels(config, databaseStructure)
    
    // 5. Initialize database connections
    dbConnect := NewDatabaseConnections(config)
    
    // 6. Build resolver groups for charting
    resolverChartGroupData := SetDataGroups(systemModels)
    
    // 7. Create server instance
    Server = &YekongaData{
        Config: config,
        models: systemModels,
        databaseStructure: databaseStructure,
        routes: make(map[string][]Route),
        functions: make(map[string]CloudFunction),
        // ... initialize other fields
    }
    
    // 8. Set DB connection app path
    dbConnect.appPath = Server.HomeDirectory()
    
    // 9. Configure database for models
    SetSystemModelDBconnection(Server, &systemModels)
    
    // 10. Build GraphQL schema
    Server.graphqlBuild = NewGraphqlAutoBuild(Server, systemModels)
    
    return Server
}
```

---

## Request/Response Processing

### Request Lifecycle Details

#### Request Object Fields

```go
type Request struct {
    // Core HTTP request
    HttpRequest *http.Request
    
    // Request metadata
    Method           string
    URL              string
    Path             string
    QueryString      string
    
    // Parsed data
    parsedQuery      url.Values          // Query parameters
    parsedBody       map[string]interface{} // Body as map
    rawBody          []byte              // Raw body bytes
    
    // Context and authentication
    context          map[string]interface{} // Request context
    contextObject    map[string]interface{} // Object storage
    tokenPayload     *TokenPayload       // JWT claims
    auth             *AuthPayload        // User authentication
    
    // Application reference
    App              *YekongaData        // Server instance
    
    // Synchronization
    mu               sync.RWMutex        // Thread safety
}
```

#### Request Creation Flow

```go
// In route handler registration
app.Get("/path", handler)

// When request arrives:
1. HTTP server multiplexes to handler
2. Create Request wrapper:
   request := &Request{
       HttpRequest: httpRequest,
       App: app,
       Method: httpRequest.Method,
       URL: httpRequest.URL.String(),
   }
3. Parse body (if POST/PUT):
   request.rawBody = ioutil.ReadAll(httpRequest.Body)
   request.parsedBody = parseJSON(rawBody)
4. Initialize context maps
5. Pass to middleware pipeline
```

#### Body Parsing

```go
// In request.go - Body() method
func (r *Request) Body() map[string]interface{} {
    if r.parsedBody == nil {
        // Read raw body
        r.rawBody, _ = ioutil.ReadAll(r.HttpRequest.Body)
        
        // Parse JSON
        json.Unmarshal(r.rawBody, &r.parsedBody)
        
        // Reset body for potential re-reading
        r.HttpRequest.Body = ioutil.NopCloser(
            bytes.NewBuffer(r.rawBody),
        )
    }
    return r.parsedBody
}
```

### Response Object

#### Response Implementation

```go
type Response struct {
    // Underlying response writer
    writer http.ResponseWriter
    
    // Status tracking
    status  int
    written bool  // Track if headers written
    
    // Headers management
    headers map[string][]string
    
    // Request reference
    request *Request
}
```

#### Writing Response

```go
// Response writing flow
1. res.Status(200)        // Set status code
2. res.Header(...)        // Set headers
3. res.Json(data)         // Write body
4. Flush to client

// Status must be set BEFORE writing body
// Headers must be set BEFORE writing body
```

#### JSON Response Marshal

```go
func (r *Response) Json(data interface{}) {
    // Set content type if not set
    if r.header("Content-Type") == "" {
        r.writer.Header().Set("Content-Type", "application/json")
    }
    
    // Write status
    if !r.written {
        r.writer.WriteHeader(r.status)
        r.written = true
    }
    
    // Marshal and write JSON
    jsonBytes, _ := json.Marshal(data)
    r.writer.Write(jsonBytes)
}
```

### Context Management

#### Context Scope

```go
// RequestContext is per-request
type RequestContext struct {
    Auth             *AuthPayload          // Authenticated user
    App              *YekongaData          // Server reference
    Request          *Request              // Request object
    Client           *ClientPayload        // Client info
    TokenPayload     *TokenPayload         // JWT claims
    
    // Query building context
    QuerySelectors   map[uint][]string     // GraphQL selections
    QueryRelatedData datatype.JsonObject   // Joined data
    QueryWhereData   datatype.JsonObject   // Filter conditions
    
    mut              sync.RWMutex
}
```

#### Context Isolation

Each request gets isolated context:

```go
// In middleware/handler
func MyMiddleware(req *yekonga.Request, res *yekonga.Response) error {
    // This context is ONLY for this request
    req.SetContext("userId", "123")
    
    // Not visible to other requests
    return nil
}

// Each request gets its own map:
request1.SetContext("key", "value1")  // Request 1 context
request2.SetContext("key", "value2")  // Request 2 context
// Different values, isolated storage
```

---

## Database Connections

### Multi-Database Support Architecture

#### Connection Strategy Pattern

```
DatabaseConnections (abstraction)
├── MongoDB Driver
│   ├── mongodbClient *mongo.Client
│   ├── mongodbConnect()
│   └── mongodbClose()
├── MySQL Driver
│   ├── mysqlClient *interface{}
│   ├── mysqlConnect()
│   └── mysqlClose()
├── SQL Driver
│   ├── sqlClient *interface{}
│   ├── sqlConnect()
│   └── sqlClose()
└── Local File Database
    ├── localClient *localDB.DB
    ├── localConnect()
    └── localClose()
```

### Connection Initialization

```go
// In dbconnect.go - connect() method
func (dc *DatabaseConnections) connect() {
    switch dc.config.Database.Kind {
    case config.DBTypeMongodb:
        dc.mongodbConnect()  // MongoDB Atlas/Community
    case config.DBTypeMysql:
        dc.mysqlConnect()    // MySQL 5.7+
    case config.DBTypeSql:
        dc.sqlConnect()      // Generic SQL
    default:
        dc.localConnect()    // Local JSON DB
    }
}
```

### MongoDB Connection Details

Located in `dbconnect_mongodb.go`:

```go
func (dc *DatabaseConnections) mongodbConnect() {
    // Build connection string
    uri := fmt.Sprintf(
        "mongodb+srv://%s:%s@%s/%s",
        dc.config.Database.Username,
        dc.config.Database.Password,
        dc.config.Database.Host,
        dc.config.Database.DatabaseName,
    )
    
    // Create client with options
    client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
    
    // Test connection
    err = client.Ping(context.Background(), nil)
    
    dc.mongodbClient = client
}
```

### MySQL Connection Details

Located in `dbconnect_mysql.go`:

```go
func (dc *DatabaseConnections) mysqlConnect() {
    // Build DSN
    dsn := fmt.Sprintf(
        "%s:%s@tcp(%s:%s)/%s",
        dc.config.Database.Username,
        dc.config.Database.Password,
        dc.config.Database.Host,
        dc.config.Database.Port,
        dc.config.Database.DatabaseName,
    )
    
    // Open connection with connection pooling
    db, err := sql.Open("mysql", dsn)
    
    // Set pool parameters
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    
    dc.mysqlClient = &db
}
```

### Local Database Connection

Located in `dbconnect_local.go`:

```go
func (dc *DatabaseConnections) localConnect() {
    // Local file database for development
    dbPath := filepath.Join(dc.appPath, "database.json")
    
    // Initialize local DB
    localDB := localDB.New(dbPath)
    
    dc.localClient = &localDB
}
```

### Query Execution Flow

```
ModelQuery → DataModel → DatabaseConnections
                         ↓
                    Check db.kind
                         ↓
        ┌────────┬────────┬────────┐
        ↓        ↓        ↓        ↓
    MongoDB   MySQL    SQL    LocalDB
        ↓        ↓        ↓        ↓
    Execute Native Queries
        ↓        ↓        ↓        ↓
    Return Results (DataMap)
        ↓
    Transform to DataModel format
        ↓
    Return to Handler
```

---

## Model Query System

### Query Builder Architecture

#### ModelQuery Structure

```go
type ModelQuery struct {
    // Model reference
    model *DataModel
    
    // Query parameters
    whereConditions []WhereCondition
    orderBy         map[string]string  // field -> direction
    limit           int
    offset          int
    
    // Field selection
    selectedFields  []string
    
    // Aggregation
    groupByField    string
    aggregations    map[string]string
    
    // Relationships
    relatedModels   []string
    
    // Database connection
    dbConnect       *DatabaseConnections
}

type WhereCondition struct {
    Field    string
    Operator string
    Value    interface{}
}
```

### Method Chaining Pattern

```go
// Fluent interface - each method returns *ModelQuery for chaining
app.ModelQuery("User")
    .Where("status", "=", "active")      // returns *ModelQuery
    .Where("age", ">", 18)               // returns *ModelQuery
    .OrderBy("createdAt", "desc")        // returns *ModelQuery
    .Take(10)                            // returns *ModelQuery
    .Skip(20)                            // returns *ModelQuery
    .Find(nil)                           // Terminal: executes query
```

### Query Compilation

```go
// Compilation to database-specific query

ModelQuery → Intermediate Representation
    ↓
    ├─ MongoDB Query
    │  ├─ Filter: {status: "active", age: {$gt: 18}}
    │  ├─ Sort: {createdAt: -1}
    │  ├─ Skip: 20
    │  └─ Limit: 10
    │
    ├─ MySQL Query
    │  └─ SELECT * FROM users 
    │     WHERE status = 'active' AND age > 18 
    │     ORDER BY createdAt DESC 
    │     LIMIT 10 OFFSET 20
    │
    └─ Local DB Query
       └─ Filter JSON array in memory
```

### Where Condition Translation

```go
// Condition: Where("age", ">", 18)

MongoDB:    {age: {$gt: 18}}
MySQL:      age > 18
Local DB:   item["age"] > 18
GraphQL:    age: {greaterThan: 18}
```

### Supported Operators

```
"="               Equal to
"!="              Not equal
">"               Greater than
">="              Greater than or equal
"<"               Less than
"<="              Less than or equal
"contains"        String contains
"starts"          String starts with
"ends"            String ends with
"in"              Value in array
"notIn"           Value not in array
"exists"          Field exists
"regex"           Regex match
```

---

## GraphQL Integration

### GraphQL Auto-Build System

Located in `graphql.go` and `graphql_types.go`:

#### Schema Generation Flow

```
database.json
    ↓
DatabaseStructureType (parsed schema)
    ↓
SystemModels (DataModel objects)
    ↓
GraphqlAutoBuild
├── BuildQueryType()       → Query root type
├── BuildMutationType()    → Mutation root type
├── BuildSubscriptionType()→ Subscription root type
├── BuildDataTypes()       → Model types
├── BuildEnumTypes()       → Enum types
└── BuildSchema()          → Final schema
    ↓
graphql.Schema (executable schema)
```

#### GraphQL Type Mapping

```go
type GraphqlAutoBuild struct {
    yekonga             *YekongaData
    GraphqlSubscription map[string]map[string]GraphqlSubscription
    Database            map[string]*DataModel
    EnumTypes           map[string]*graphql.Enum         // Select field enums
    QueryTypes          map[string]*graphql.Object      // Query fields
    MutationTypes       map[string]*graphql.InputObject // Mutation inputs
    Schema              graphql.Schema                  // Executable schema
    AuthSchema          graphql.Schema                  // Auth-protected schema
    mut                 sync.RWMutex
}
```

#### Type Generation Example

```
DataModel (User)
    ├─ Fields
    │  ├─ id: DataModelID         → GraphQL: ID!
    │  ├─ username: DataModelString → GraphQL: String!
    │  ├─ email: DataModelString   → GraphQL: String!
    │  ├─ status: DataModelString  → GraphQL: Enum (active|inactive)
    │  ├─ age: DataModelNumber     → GraphQL: Int
    │  └─ createdAt: DataModelDate → GraphQL: DateTime!
    │
    └─ Generated GraphQL Type
       type User {
           id: ID!
           username: String!
           email: String!
           status: UserStatus!
           age: Int
           createdAt: DateTime!
       }
```

#### Query Execution

```go
// GraphQL query execution flow

GraphQL Query Input
    ↓
Validate against schema
    ↓
Authorize (if auth required)
    ↓
Extract query parameters
    ↓
Build ModelQuery
    ↓
Execute database query
    ↓
Apply field selection
    ↓
Load relationships (if included)
    ↓
Serialize to GraphQL response
    ↓
Return to client
```

#### Subscription Handling

```
GraphQL Subscription
    ↓
Create WebSocket connection
    ↓
Subscribe to changes
    ↓
Store in GraphqlSubscription map
    ↓
On database change:
    ├─ Trigger event
    ├─ Notify subscribers
    └─ Send data over WebSocket
```

---

## Cloud Functions System

### Function Registry

Located in `cloud_functions.go`:

#### Registry Structure

```go
type YekongaData struct {
    // Function registry
    functions              map[string]CloudFunction
    
    // Primary functions (pre-defined)
    primaryFunctions       map[PrimaryCloudKey]CloudFunction
    
    // Database trigger functions
    triggerFunctions       map[string]map[TriggerAction]map[string]TriggerCloudFunction
    // Structure: [modelName][action][functionName]TriggerCloudFunction
    
    // GraphQL action functions
    graphqlActionFunctions map[string]map[string]map[string]ActionCloudFunction
    // Structure: [modelName][actionName][operationName]ActionCloudFunction
}
```

### Trigger Execution Points

```
CREATE Operation
    ├─ BeforeCreate triggers
    ├─ INSERT into database
    ├─ AfterCreate triggers
    └─ Return created record

READ Operation
    ├─ BeforeFind triggers
    ├─ SELECT from database
    ├─ AfterFind triggers (transform records)
    └─ Return records

UPDATE Operation
    ├─ BeforeUpdate triggers (validate)
    ├─ UPDATE in database
    ├─ AfterUpdate triggers (log change)
    └─ Return updated record

DELETE Operation
    ├─ BeforeDelete triggers (check references)
    ├─ DELETE from database
    ├─ AfterDelete triggers (cleanup)
    └─ Return deletion status
```

### Trigger Context

```go
// BeforeCreate/Update passes original data
func(rc *RequestContext, qc *QueryContext) {
    // qc.Data = record data to insert/update
    // qc.PreviousData = old record (for updates)
    // Modify qc.Data to change what gets saved
}

// AfterCreate/Update/Delete passes processed data
func(rc *RequestContext, qc *QueryContext) {
    // qc.Data = record that was created/updated/deleted
    // Read-only, changes won't affect database
    // Use for: notifications, logging, analytics
}
```

### Function Calling

```go
// Explicit function call
result, err := app.CallFunction("functionName", inputData)

// Function with context
result, err := app.CallFunctionWithContext(
    "functionName", 
    inputData, 
    requestContext,
)

// Async function execution
go app.CallFunctionAsync("functionName", inputData)
```

---

## Helper Functions

### Helper Package Organization

Located in `/helper` directory:

#### Main Module (`helpers.go`)

Provides 120+ utility functions:

```go
// Type conversion
helper.ToInt(value)
helper.ToFloat(value)
helper.ToBool(value)
helper.ToString(value)
helper.ToJson(data)

// String manipulation
helper.ToSlug(str)           // "Hello World" → "hello-world"
helper.ToCamelCase(str)      // "hello_world" → "helloWorld"
helper.ToUnderscore(str)     // "helloWorld" → "hello_world"
helper.ToTitle(str)          // "hello world" → "Hello World"
helper.Capitalize(str)       // "hello" → "Hello"

// Validation
helper.IsEmail(value)
helper.IsPhone(value)
helper.IsURL(value)
helper.IsEmpty(value)

// Array/Slice operations
helper.Contains(slice, value)
helper.Index(slice, value)
helper.Remove(slice, value)
helper.Unique(slice)

// Date/Time
helper.Today()
helper.Tomorrow()
helper.Now()
helper.WeekStart(time.Time)
helper.MonthEnd(time.Time)

// File operations
helper.FileExists(path)
helper.ReadFile(path)
helper.WriteFile(path, data)
helper.LoadJSONFile(path)
helper.SaveToFile(data, path)

// ID/Encryption
helper.UUID()
helper.MD5(data)
helper.SHA256(data)

// Network
helper.GetLocalIP()
helper.GetPublicIP()
```

#### Specialized Modules

```
helper/
├── jwt/
│   ├── jwt.go              # JWT token operations
│   ├── GenerateJWT()       # Create token
│   └── DecodeJWT()         # Parse token
├── console/
│   └── log.go              # Console logging
├── logger/
│   └── log.go              # Application logging
├── idle_time/
│   ├── idle_time_darwin.go  # macOS idle time
│   ├── idle_time_linux.go   # Linux idle time
│   └── idle_time_windows.go # Windows idle time
└── command.go              # Command execution
```

### Helper Integration

```go
// Helpers used throughout the framework

// In request parsing
helper.ToJson(body)

// In validation
if !helper.IsEmail(email) {
    return errors.New("invalid email")
}

// In data transformation
slug := helper.ToSlug(name)

// In JWT handling
helper.GenerateJWT(claims)

// In file operations
data := helper.LoadJSONFile("config.json")
```

---

## Plugin System

### Plugin Architecture

Located in `/plugins` directory:

#### Database Plugins

```
plugins/database/
├── db/              # Local JSON database
├── dberr/           # Database error types
├── gommap/          # Memory-mapped operations
└── tdlog/           # Transaction logging
```

#### GraphQL Plugin

```
plugins/graphql/
├── graphql.go       # Core GraphQL
├── schema.go        # Schema building
├── types.go         # Type system
├── validator.go     # Validation
├── executor.go      # Query execution
├── language/        # Query language parsing
├── gqlerrors/       # Error handling
└── examples/        # Usage examples
```

#### Database Drivers

```
plugins/
├── mongo-driver/    # MongoDB official driver
├── mysql/          # MySQL driver
├── redis/          # Redis caching (if present)
└── websocket/      # WebSocket support
```

### Plugin Integration

Plugins are transparent to the framework:

```go
// Framework automatically selects plugin based on config
if config.Database.Kind == "mongodb" {
    import "mongo-driver"  // Auto-imported in dbconnect.go
}

if config.Database.Kind == "mysql" {
    import "mysql"  // Auto-imported in dbconnect.go
}
```

---

## Module Interactions

### Data Flow Through Modules

```
Client Request
    ↓
HTTP Server (net/http)
    ↓
Router (main.go)
    ↓
Request Wrapper Creation (request.go)
    ↓
Middleware Pipeline
├─ Init Middlewares
├─ Preload Middlewares
└─ Global Middlewares
    ↓
Route Handler
    ↓
ModelQuery Creation (model_query.go)
    ↓
DatabaseConnections (dbconnect.go)
    ├─ MongoDB (dbconnect_mongodb.go)
    ├─ MySQL (dbconnect_mysql.go)
    ├─ SQL (dbconnect_sql.go)
    └─ Local (dbconnect_local.go)
    ↓
Query Execution
    ↓
Cloud Function Triggers (cloud_functions.go)
    ├─ Before Triggers
    └─ After Triggers
    ↓
Response Creation (response.go)
    ↓
HTTP Response to Client
```

### Typical Request Lifecycle

```
1. Client sends HTTP request
2. Server receives on port 8080/8443
3. Router matches path pattern
4. Request wrapper created
5. TokenMiddleware extracts JWT
6. UserInfoMiddleware loads user
7. Custom middlewares execute
8. Route handler invoked
9. Handler creates ModelQuery
10. Query compiled to database-specific format
11. Database query executed
12. Results transformed to DataMap
13. AfterFind triggers execute
14. Response serialized (JSON/GraphQL)
15. Response sent to client
```

---

## Performance Characteristics

### Latency Points

| Operation | Typical Latency | Optimization |
|-----------|-----------------|--------------|
| Middleware pipeline | 0.1-1ms | Minimize middleware complexity |
| Model query build | 0.5-2ms | Cache frequently used queries |
| Database query | 10-100ms | Add indexes, optimize query |
| JSON serialization | 1-10ms | Use field selection |
| Network I/O | 10-1000ms | Client location dependent |

### Memory Usage

- Per request: ~50-200KB (varies with payload)
- Server static: ~10-50MB (models, schema, connections)
- Connection pool: 10-20MB (database connections)

### Concurrency

- All modules use sync.RWMutex for thread safety
- Request context is isolated per goroutine
- Database connections are pooled and reused

---

This module documentation should be combined with INTERNAL_ARCHITECTURE.md for complete understanding of the framework's internal structure.
