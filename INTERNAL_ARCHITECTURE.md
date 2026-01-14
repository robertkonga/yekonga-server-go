# YekongaServer - Internal Architecture Documentation

## Overview

YekongaServer is a production-ready Go framework built on a modular, layered architecture designed for scalability and extensibility. This document provides internal development team members with deep architectural insights.

## Table of Contents

1. [Core Architecture](#core-architecture)
2. [Layer Model](#layer-model)
3. [Request/Response Lifecycle](#requestresponse-lifecycle)
4. [Data Model System](#data-model-system)
5. [Database Abstraction Layer](#database-abstraction-layer)
6. [Middleware Pipeline](#middleware-pipeline)
7. [Type System](#type-system)

---

## Core Architecture

### High-Level Structure

```
┌─────────────────────────────────────────────────────┐
│         HTTP Server (Net/HTTP)                      │
└────────────────────┬────────────────────────────────┘
                     │
     ┌───────────────┼───────────────┐
     │               │               │
┌────▼─────┐   ┌─────▼──────┐  ┌──────▼────┐
│  REST    │   │  GraphQL   │  │ WebSocket │
│  Router  │   │  Executor  │  │  Server   │
└────┬─────┘   └─────┬──────┘  └──────┬────┘
     │               │               │
     └───────────────┼───────────────┘
                     │
         ┌───────────▼───────────┐
         │  Middleware Pipeline  │
         └───────────┬───────────┘
                     │
      ┌──────────────┼──────────────┐
      │              │              │
 ┌────▼─────┐  ┌────▼─────┐  ┌──────▼────┐
 │ Request  │  │ Response │  │ Context   │
 │ Handler  │  │ Writer   │  │ Storage   │
 └────┬─────┘  └──────────┘  └───────────┘
      │
      │    ┌──────────────────────┐
      └───▶│ Cloud Functions      │
           │ (Triggers/Actions)   │
           └──────────┬───────────┘
                      │
      ┌───────────────┼───────────────┐
      │               │               │
┌─────▼──────┐  ┌────▼──────┐  ┌────▼─────┐
│Data Model  │  │  Query    │  │  Database│
│Definitions │  │  Builder  │  │Connection│
└────────────┘  └───────────┘  └──────────┘
      │               │               │
      └───────────────┼───────────────┘
                      │
         ┌────────────▼──────────┐
         │  Multi-DB Support    │
         │  (MongoDB/MySQL/SQL) │
         └──────────────────────┘
```

### Main Components

**YekongaData** (yekonga/main.go)
- Central server instance holding all state
- Routes management (HTTP methods and patterns)
- Middleware pipelines
- Cloud functions registry
- Database connections
- Configuration and models

**Request/Response** (yekonga/request.go, yekonga/response.go)
- Abstracts Go's http.Request/ResponseWriter
- Provides context management
- Authentication payload handling
- Query parameter parsing

**DataModel** (yekonga/model.go)
- Schema representation for database tables
- Field type definitions and constraints
- Foreign key relationships
- Computed properties and validation rules

---

## Layer Model

### 1. Transport Layer
- **HTTP Server**: Standard Go net/http with custom routing
- **WebSocket**: Real-time bi-directional communication
- **Protocols**: HTTP, HTTPS, WSS support

### 2. API Layer
- **REST Router**: Pattern-based routing with parameter extraction
- **GraphQL Executor**: Full GraphQL query/mutation/subscription support
- **Route Handlers**: Function-based request handling

### 3. Middleware Layer
Three distinct middleware chains:

| Chain | Type | Purpose | Timing |
|-------|------|---------|--------|
| **Init Middleware** | `InitMiddleware` | Initialize request context | Before route handler |
| **Preload Middleware** | `PreloadMiddleware` | Preload data/resources | Before route handler |
| **Global Middleware** | `GlobalMiddleware` | Common logic (auth, logging) | Before route handler |

### 4. Business Logic Layer
- **Cloud Functions**: User-defined functions with trigger/action support
- **Data Models**: Domain entity definitions
- **Query Builder**: Fluent interface for database queries
- **Validation**: Field and data validation

### 5. Data Access Layer
- **DatabaseConnections**: Unified database connection management
- **Model Query Interface**: Database agnostic query execution
- **Connection Pooling**: Efficient resource management

### 6. Database Layer
- **MongoDB Driver**: Async, modern Go driver
- **MySQL Driver**: Standard SQL support
- **Local DB**: File-based JSON database
- **SQL Support**: Generic SQL implementation

---

## Request/Response Lifecycle

### Incoming Request Flow

```
1. HTTP Request arrives at Server
   ↓
2. Server multiplexes to appropriate handler (REST/GraphQL/WebSocket)
   ↓
3. Request wrapper created (*Request object)
   ↓
4. Init Middlewares executed sequentially
   ↓
5. Preload Middlewares executed sequentially
   ↓
6. Global Middlewares (including auth) executed sequentially
   ↓
7. Route Handler invoked with (*Request, *Response)
   ↓
8. Handler can:
   - Query database via ModelQuery
   - Call cloud functions
   - Invoke middleware functions
   - Write to response
   ↓
9. Cloud Function Triggers executed (if registered)
   ↓
10. Response written to client
```

### Request Context Flow

```
RequestContext
├── Auth: User authentication data
├── App: Reference to YekongaData
├── Request: Original request object
├── Client: Client information
├── TokenPayload: JWT token claims
├── QuerySelectors: GraphQL field selection
├── QueryRelatedData: Related record data
└── QueryWhereData: Where clause conditions
```

### Key Request Methods

```go
// Parameter extraction
req.Param("id")              // Route parameter
req.Query("filter")          // Query string parameter
req.Body()                   // Request body as map

// Authentication
req.Auth() *AuthPayload      // Current user
req.Token() string           // Bearer token

// Context management
req.SetContext(key, value)   // Store context value
req.GetContext(key)          // Retrieve context value

// Utilities
req.Method()                 // HTTP method
req.URL()                    // Request URL
req.Header(name)             // Get header value
```

---

## Data Model System

### Model Definition Structure

```go
type DataModel struct {
    // Identity
    Name            string              // Model name ("User")
    Class           string              // Class name
    Collection      string              // Database collection/table
    
    // Naming variants
    Variable        string              // user
    VariableSingle  string              // user
    VariablePlural  string              // users
    
    // Schema information
    Fields          []DataModelField    // Model fields
    PrimaryKey      string              // Primary key field
    PrimaryName     string              // Primary key name
    
    // Field categories
    Required        []string            // Required fields
    Protected       []string            // Non-updatable fields
    DateFields      []string            // DateTime fields
    BooleanFields   []string            // Boolean fields
    NumberFields    []string            // Integer fields
    FloatFields     []string            // Float fields
    FileFields      []string            // File upload fields
    OptionFields    []string            // Select/enum fields
    
    // Relationships
    ParentKeys      []string            // Foreign keys
}
```

### Field Type System

```go
const (
    DataModelID     = "id"      // Auto ID
    DataModelString = "string"  // Text field
    DataModelNumber = "number"  // Integer field
    DataModelFloat  = "float"   // Decimal field
    DataModelDate   = "date"    // DateTime field
    DataModelBool   = "bool"    // Boolean field
    DataModelObject = "object"  // JSON object
    DataModelArray  = "array"   // JSON array
    DataModelFile   = "file"    // File upload
)
```

### Model Initialization

Models are initialized from `database.json`:

```json
{
  "models": [
    {
      "name": "User",
      "fields": {
        "id": {"type": "id", "primary": true},
        "username": {"type": "string", "required": true, "unique": true},
        "email": {"type": "string", "required": true},
        "status": {"type": "string", "options": [
          {"value": "active", "label": "Active"},
          {"value": "inactive", "label": "Inactive"}
        ]},
        "createdAt": {"type": "date"}
      }
    }
  ]
}
```

The `NewSystemModels()` function parses this and creates `DataModel` instances stored in `YekongaData.models` map.

---

## Database Abstraction Layer

### DatabaseConnections Architecture

```go
type DatabaseConnections struct {
    config        *config.YekongaConfig  // Database configuration
    appPath       string                 // Application path
    mongodbClient *mongo.Client          // MongoDB connection
    localClient   *localDB.DB            // Local JSON DB
    mysqlClient   *interface{}           // MySQL connection
    sqlClient     *interface{}           // SQL connection
}
```

### Connection Strategy

The database abstraction uses the **Strategy Pattern**:

```
DatabaseConnections
├── mongodbConnect()   → MongoDB Atlas/Community
├── mysqlConnect()     → MySQL 5.7+ / MariaDB
├── sqlConnect()       → Generic SQL databases
└── localConnect()     → File-based JSON (Development)
```

Selected based on `config.database.kind`:
- `"mongodb"` → MongoDB driver
- `"mysql"` → MySQL driver
- `"sql"` → Generic SQL driver
- `"local"` → Local file database

### Query Interface

All databases implement `dataModelQueryStructure` interface:

```go
type dataModelQueryStructure interface {
    findOne() *datatype.DataMap        // Single record
    findAll() *[]datatype.DataMap      // Multiple records
    find() *[]datatype.DataMap         // With filtering
    pagination() *datatype.DataMap     // Paginated results
    summary() *datatype.DataMap        // Aggregated data
    count() int64                      // Record count
    max(field) interface{}             // Maximum value
    min(field) interface{}             // Minimum value
    sum(field) float64                 // Sum of values
    average(field) float64             // Average value
    graph() *datatype.DataMap          // Chart data
    
    create(data) (*datatype.DataMap, error)
    createMany(data) (*[]datatype.DataMap, error)
    update(data) (*datatype.DataMap, error)
    updateMany(data) (*[]datatype.DataMap, error)
    delete() (interface{}, error)
}
```

### ModelQuery Builder

The `ModelQuery` provides a fluent interface:

```go
app.ModelQuery("User")
    .Where("status", "=", "active")
    .Where("age", ">", 18)
    .OrderBy("createdAt", "desc")
    .Take(10)
    .Skip(20)
    .Find(nil)  // Executes query
```

**Query methods** (chainable):
- `Where(field, operator, value)` - Add filter
- `OrderBy(field, direction)` - Sort results
- `Take(limit)` - Limit results
- `Skip(offset)` - Skip N records
- `Select(fields...)` - Choose fields

**Execution methods** (terminal):
- `Find(selectors)` - Execute query
- `FindOne(filter)` - Single record
- `FindById(id)` - By primary key
- `Count()` - Record count
- `Create(data)` - Insert record
- `Update(id, data)` - Modify record
- `Delete(id)` - Remove record

---

## Middleware Pipeline

### Middleware Execution Order

```
Global Middlewares (in order registered)
    ↓
Init Middlewares (in order registered)
    ↓
Preload Middlewares (in order registered)
    ↓
Route Handler
    ↓
Cloud Function Triggers (if any)
```

### Built-in Middlewares

1. **TokenMiddleware** (middleware.go)
   - Extracts Bearer token from Authorization header
   - Decodes JWT payload
   - Stores in context under `TokenKey`

2. **UserInfoMiddleware** (middleware.go)
   - Fetches user record from token payload
   - Stores in context under `UserInfoPayloadKey`
   - Available to handlers via `req.GetContext(UserInfoPayloadKey)`

3. **ApplicationKeyMiddleware** (middleware.go)
   - Validates API key authentication
   - Extracts from `X-App-Key` header
   - Checks against configured `appKey`

### Custom Middleware Implementation

```go
// Define middleware function
func MyMiddleware(req *yekonga.Request, res *yekonga.Response) error {
    // Pre-processing logic
    value := req.Header("X-Custom-Header")
    
    // Store in context for handler access
    req.SetContext("customKey", value)
    
    // Return error to stop pipeline
    if err != nil {
        res.Error("Unauthorized", 401)
        return fmt.Errorf("unauthorized")
    }
    
    return nil  // Continue to next middleware/handler
}

// Register middleware
app.Use(MyMiddleware)  // Global
app.Middleware(MyMiddleware, yekonga.InitMiddleware)  // Init
app.Middleware(MyMiddleware, yekonga.PreloadMiddleware)  // Preload
```

### Middleware Context Storage

Middlewares communicate with handlers via context:

```go
// In middleware
req.SetContext("userId", "123")
req.SetContextObject("user", userData)

// In handler
handler := func(req *yekonga.Request, res *yekonga.Response) {
    userId := req.GetContext("userId")
    user := req.GetContextObject("user")
}
```

---

## Type System

### Data Type Definitions (datatype package)

```go
// DataMap - Flexible key-value storage
type DataMap map[string]interface{}

// JsonObject - Structured JSON data
type JsonObject map[string]interface{}

// DataGroup - Grouped data aggregation
type DataGroup struct {
    GroupKey   string
    FieldName  string
    GroupData  interface{}
}
```

### Context Keys (Type-safe)

```go
type ContextKey string

const (
    TokenKey              ContextKey = "token"
    TokenPayloadKey       ContextKey = "tokenPayload"
    UserInfoPayloadKey    ContextKey = "userInfo"
    ClientPayloadKey      ContextKey = "client"
)
```

### Cloud Function Types

```go
// Basic function: receives data, returns result
type CloudFunction func(interface{}, *RequestContext) (interface{}, error)

// Trigger function: executes after database operation
type TriggerCloudFunction func(*RequestContext, *QueryContext) (interface{}, error)

// Action function: GraphQL mutation handler
type ActionCloudFunction func(*RequestContext, *QueryContext) (GraphqlActionResult, error)
```

### Query Context

```go
type QueryContext struct {
    Data          interface{}    // Record data
    Model         *DataModel     // Model definition
    Selection     []string       // Selected fields
    RelatedData   datatype.JsonObject
    PreviousData  interface{}    // For updates
}
```

---

## Concurrency and Thread Safety

### Synchronization Mechanisms

```go
// YekongaData uses RWMutex
type YekongaData struct {
    // ...
    mut sync.RWMutex  // Protects concurrent access
}

// RequestContext uses RWMutex
type RequestContext struct {
    // ...
    mut sync.RWMutex  // Protects context map
}

// GraphqlAutoBuild uses RWMutex
type GraphqlAutoBuild struct {
    // ...
    mut sync.RWMutex  // Protects schema mutations
}
```

### Thread-Safe Operations

```go
// Safe read
server.mut.RLock()
model := server.models["User"]
server.mut.RUnlock()

// Safe write
server.mut.Lock()
server.models["NewModel"] = newModel
server.mut.Unlock()
```

---

## Extension Points

### 1. Custom Route Handlers
```go
app.Get("/custom", func(req *yekonga.Request, res *yekonga.Response) {
    // Custom logic
})
```

### 2. Cloud Functions
```go
app.Define("customFunction", func(data interface{}, rc *yekonga.RequestContext) (interface{}, error) {
    // Custom logic
    return result, nil
})
```

### 3. Database Triggers
```go
app.BeforeCreate("User", func(rc *yekonga.RequestContext, qc *yekonga.QueryContext) (interface{}, error) {
    // Pre-insert logic
    return qc.Data, nil
})
```

### 4. Custom Middleware
```go
app.Use(func(req *yekonga.Request, res *yekonga.Response) error {
    // Middleware logic
    return nil
})
```

### 5. GraphQL Actions
```go
app.RegisterGraphqlAction("User", "activate", func(rc *yekongaRequestContext, qc *yekonga.QueryContext) (yekonga.GraphqlActionResult, error) {
    // Action logic
    return yekonga.GraphqlActionResult{...}, nil
})
```

---

## Performance Considerations

### 1. Connection Pooling
- MongoDB: Configured in driver options
- MySQL: Connection pool managed by driver
- Local DB: Single file handle

### 2. Caching Strategies
- Models cached in memory at startup
- Database connections reused
- GraphQL schema built once

### 3. Query Optimization
- Use `Select()` to limit fields
- Use `Take()` for pagination
- Index database queries appropriately

### 4. Middleware Ordering
- Place expensive middlewares last
- Cache computation results in context
- Avoid repeated database lookups

---

## Error Handling Strategy

### Request/Response Error Handling

```go
// Write error response
res.Error("User not found", 404)

// Custom error with status
res.Status(400)
res.Json(map[string]string{"error": "Invalid input"})
```

### Cloud Function Error Propagation

```go
// In cloud function
if err != nil {
    return nil, fmt.Errorf("operation failed: %v", err)
}
```

### Middleware Error Handling

```go
// Stop request pipeline on error
if err != nil {
    return fmt.Errorf("validation failed")
}
```

---

## Configuration Precedence

1. **config.json** - Application configuration
2. **database.json** - Schema definitions
3. **Environment variables** - Runtime overrides
4. **Default values** - Built-in defaults

---

## Summary

YekongaServer's architecture balances:
- **Modularity**: Clear separation of concerns
- **Extensibility**: Multiple extension points
- **Type Safety**: Compile-time safety with Go
- **Performance**: Efficient resource management
- **Developer Experience**: Fluent APIs and clear conventions

The layered approach allows teams to work on different layers independently while maintaining consistency across the framework.
