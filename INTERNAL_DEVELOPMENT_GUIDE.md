# YekongaServer - Development Guidelines & Best Practices

Internal development standards and best practices for the YekongaServer team.

## Table of Contents

1. [Code Organization](#code-organization)
2. [Development Workflow](#development-workflow)
3. [Naming Conventions](#naming-conventions)
4. [Error Handling](#error-handling)
5. [Performance Guidelines](#performance-guidelines)
6. [Testing Strategies](#testing-strategies)
7. [Logging Standards](#logging-standards)
8. [Security Practices](#security-practices)
9. [Database Design](#database-design)
10. [Common Patterns](#common-patterns)

---

## Code Organization

### Package Structure

```
YekongaServer_Go/
├── yekonga/              # Main framework package
│   ├── main.go           # ~581 lines: Server core
│   ├── request.go        # ~326 lines: Request handling
│   ├── response.go       # Response writing
│   ├── model.go          # ~403 lines: Data model definitions
│   ├── model_query.go    # ~500+ lines: Query builder
│   ├── middleware.go     # ~123 lines: Middleware system
│   ├── graphql.go        # ~1883 lines: GraphQL implementation
│   ├── cloud_functions.go # Function registry and execution
│   ├── cronjob.go        # Scheduled tasks
│   ├── database_structure.go # Schema parsing
│   ├── dbconnect.go      # Database abstraction (~184 lines)
│   ├── dbconnect_mongodb.go # MongoDB driver
│   ├── dbconnect_mysql.go   # MySQL driver
│   ├── dbconnect_sql.go     # SQL driver
│   └── dbconnect_local.go   # Local JSON DB
├── config/               # Configuration management
├── datatype/             # Custom data types
├── helper/               # 120+ utility functions
├── plugins/              # Database drivers and extensions
└── [root files]          # Config, README, API docs
```

### File Size Guidelines

- **Small files**: < 200 lines (focused responsibility)
- **Medium files**: 200-500 lines (main logic)
- **Large files**: 500-1000 lines (acceptable for complex modules)
- **Very large files**: > 1000 lines (graphql.go - consider splitting)

### Package Conventions

- **One responsibility per package**
- **Exported types are capitalized** (YekongaData, DataModel)
- **Unexported helpers are lowercase** (serverConfig, parseQuery)
- **Group related functions** together
- **Use clear package names** (not "util" or "helper" when possible)

---

## Development Workflow

### Setting Up Development Environment

```bash
# 1. Clone repository
git clone <repo>
cd YekongaServer_Go

# 2. Install dependencies
go mod download
go mod verify

# 3. Build the project
go build ./...

# 4. Run tests
go test ./...

# 5. Run linter
golangci-lint run ./...
```

### Local Development Setup

```bash
# Create development config
cp config.json config.development.json

# Update dev config
{
  "environment": "development",
  "debug": true,
  "database": {
    "kind": "local"  # Use local JSON DB for development
  }
}

# Run server
go run main.go
```

### Branch Strategy

```
main
├─ production-ready
└─ release candidate only

dev
├─ development branch
└─ all feature branches merge here

feature/feature-name
├─ individual feature branches
└─ merge to dev when complete

bugfix/bug-name
├─ bug fix branches
└─ merge to dev and main if critical
```

### Commit Message Format

```
[TYPE] Brief description

Detailed explanation of the change.

Types:
feat    - New feature
fix     - Bug fix
docs    - Documentation changes
refactor - Code refactoring
perf    - Performance improvement
test    - Test additions/changes
chore   - Build, config changes
security - Security updates

Example:
[feat] Add pagination support to ModelQuery

Implement Take() and Skip() methods for efficient
pagination on large datasets. Supports MongoDB and MySQL.
```

### Code Review Checklist

Before submitting PR:

- [ ] Code follows naming conventions
- [ ] All new public functions documented
- [ ] Error handling implemented
- [ ] Thread safety verified (mutex usage)
- [ ] No race conditions (use `go test -race`)
- [ ] Performance impact assessed
- [ ] Security implications reviewed
- [ ] Tests added/updated
- [ ] No debug code left in

---

## Naming Conventions

### Type Names

```go
// Server types
YekongaData              // Main server instance
Request                 // HTTP request wrapper
Response                // HTTP response wrapper
RequestContext          // Per-request context
DataModel              // Model definition
ModelQuery             // Query builder

// Types are capitalized and descriptive
DatabaseConnections    // Connection manager
GraphqlAutoBuild       // GraphQL schema builder
AuthPayload            // Authentication data

// Avoid
util                    // ✗ Too generic
handler                 // ✗ Use more specific name
data                    // ✗ Ambiguous
```

### Function Names

```go
// Package-level functions (exported)
ServerConfig()         // Initialize server
ServerLoad()           // Load global server
NewDatabaseStructure() // Create new structure
NewSystemModels()      // Create model set

// Method names are descriptive
req.Auth()            // Get authenticated user
query.Where()         // Add filter condition
res.Json()            // Send JSON response

// Helper methods (unexported)
parseQuery()          // ✓ Lowercase
buildSchema()         // ✓ Clear purpose
validateData()        // ✓ Action verb

// Avoid
get()                 // ✗ Too vague
do()                  // ✗ Non-descriptive
process()             // ✗ Ambiguous
```

### Variable Names

```go
// Single letter for loop counters
for i := 0; i < len(items); i++ { }

// Descriptive names for other variables
user := app.ModelQuery("User").FindById(id)
filter := req.Query("filter")
config := app.Config

// Plural for collections
users := app.ModelQuery("User").Find(nil)
models := app.Models()

// Interface types
type dataModelQueryStructure interface { }  // Suffix with "er" or type name
type CloudFunction func(...)               // Descriptive function type

// Avoid
u := user               // ✗ Unnecessary abbreviation
tmp := tempValue        // ✗ Temporary is vague
x := data              // ✗ Not descriptive
a, b, c := ...         // ✗ Use meaningful names
```

### Constant Names

```go
// Exported constants: PascalCase
const (
    DataModelID     = "id"
    DataModelString = "string"
    DefaultPageSize = 50
    MaxPageSize     = 1000
)

// Unexported constants: camelCase
const (
    defaultTimeout  = 30 * time.Second
    maxRetries      = 3
    bufferSize      = 1024
)

// Configuration keys
const (
    TokenKey           = "token"
    UserInfoPayloadKey = "userInfo"
)
```

---

## Error Handling

### Error Return Conventions

```go
// Function that might fail
func Create(data map[string]interface{}) (*DataMap, error) {
    // Return (result, nil) on success
    if valid {
        return &result, nil
    }
    
    // Return (nil, error) on failure
    return nil, fmt.Errorf("validation failed: %w", err)
}

// Middleware functions
func MyMiddleware(req *Request, res *Response) error {
    // Return nil if processing should continue
    if isValid {
        return nil
    }
    
    // Return error to stop request pipeline
    res.Error("Invalid request", 400)
    return fmt.Errorf("validation failed")
}
```

### Error Response Standards

```go
// Response to client
res.Error("User not found", 404)

// Response structure (automatic)
{
    "status": false,
    "message": "User not found",
    "code": 404
}

// Custom error response
res.Status(400)
res.Json(map[string]interface{}{
    "status": false,
    "message": "Validation failed",
    "errors": map[string]string{
        "email": "Invalid email format",
        "age": "Must be >= 18",
    },
})
```

### Error Logging

```go
// Use logger for errors
import "github.com/robertkonga/yekonga-server/helper/logger"

logger.Error("Database connection failed: %v", err)
logger.Warn("Slow query detected: %dms", duration)

// Log with context
logger.Error("User creation failed for %s: %v", userId, err)

// Don't use log.Fatal in middleware - return error instead
if err != nil {
    // ✓ Good: Return error from middleware
    return fmt.Errorf("database error: %w", err)
}

// ✗ Bad: Use log.Fatal
if err != nil {
    log.Fatal(err)  // Stops entire server
}
```

### Panics

```go
// Use panic only for truly unrecoverable situations
// (configuration loading, type assertion failures during init)

// ✗ Don't panic on user input
user := req.Body()
if user["id"] == nil {
    res.Error("ID required", 400)  // ✓ Correct
    return
}

// ✓ OK to panic on logic errors during startup
if config == nil {
    panic("Configuration not loaded")
}
```

---

## Performance Guidelines

### Database Query Optimization

```go
// ✓ Good: Select only needed fields
users := app.ModelQuery("User")
    .Select("id", "username", "email")
    .Where("status", "=", "active")
    .Find(nil)

// ✗ Bad: Fetch all fields
users := app.ModelQuery("User")
    .Find(nil)  // May include large text fields, images, etc.

// ✓ Good: Use pagination for large datasets
users := app.ModelQuery("User")
    .Take(20)
    .Skip(page * 20)
    .Find(nil)

// ✗ Bad: Fetch millions of records
users := app.ModelQuery("User").Find(nil)

// ✓ Good: Add where conditions before executing
count := app.ModelQuery("User")
    .Where("status", "=", "active")
    .Count()

// ✗ Bad: Filter in application code
allUsers := app.ModelQuery("User").Find(nil)
active := filter(allUsers, func(u) bool { return u["status"] == "active" })
```

### Middleware Performance

```go
// ✓ Good: Cache expensive operations
func MyMiddleware(req *Request, res *Response) error {
    // Check if already in context
    if user := req.GetContextObject("cachedUser"); user != nil {
        return nil
    }
    
    // Only fetch once per request
    user := app.ModelQuery("User").FindById(req.Auth().ID)
    req.SetContextObject("cachedUser", user)
    
    return nil
}

// ✗ Bad: Repeat expensive operations
func MyMiddleware(req *Request, res *Response) error {
    user := app.ModelQuery("User").FindById(req.Auth().ID)  // Query each time
    // ...
}

// ✓ Good: Order middlewares by cost
app.Use(AuthMiddleware)         // Cheap: Parse token
app.Use(UserMiddleware)         // Moderate: Database query
app.Use(PermissionMiddleware)   // Expensive: Complex checking

// ✗ Bad: Expensive middleware first
app.Use(PermissionMiddleware)   // Expensive first
app.Use(UserMiddleware)         // Moderate next
app.Use(AuthMiddleware)         // Cheap last
```

### Goroutine Usage

```go
// ✓ Good: Use goroutines for I/O-bound work
app.RegisterCronjob("cleanup", 24*time.Hour, func(app *YekongaData, t time.Time) {
    // This runs in background, doesn't block
    app.ModelQuery("Session").Where("expires", "<", time.Now()).Delete()
})

// ✓ Good: Async operations with goroutines
go func() {
    SendNotification(user)  // Don't block response
}()

// ✗ Bad: Sync blocking operations in handler
handler := func(req *Request, res *Response) {
    // Slow: Blocks response for 30 seconds
    time.Sleep(30 * time.Second)
    res.Json(result)
}

// ✓ Correct: Return immediately, do work async
handler := func(req *Request, res *Response) {
    go SendEmail(user)  // Non-blocking
    res.Json(map[string]string{"status": "sent"})
}
```

### Caching Strategies

```go
// ✓ Good: Cache computed values
var (
    cachedModels map[string]*DataModel
    cacheMutex   sync.RWMutex
)

func GetModel(name string) *DataModel {
    cacheMutex.RLock()
    defer cacheMutex.RUnlock()
    return cachedModels[name]
}

// ✓ Good: Use context-level caching
req.SetContext("userId", req.Auth().ID)
// Later in handler or middleware
userId := req.GetContext("userId")  // No re-computation

// ✗ Bad: Expensive recomputation
for i := 0; i < 1000; i++ {
    user := app.ModelQuery("User").FindById(id)  // Query 1000x times
}

// ✓ Correct: Cache in loop
user := app.ModelQuery("User").FindById(id)
for i := 0; i < 1000; i++ {
    process(user)  // Reuse same user object
}
```

---

## Testing Strategies

### Unit Test Structure

```go
package yekonga

import (
    "testing"
)

func TestModelQuery_Where(t *testing.T) {
    // Arrange: Set up test data
    app := ServerConfig("./test_config.json", "./test_database.json")
    
    // Act: Perform the action
    query := app.ModelQuery("User").Where("status", "=", "active")
    
    // Assert: Verify the result
    if query == nil {
        t.Error("Expected query, got nil")
    }
}

// Naming convention: Test<FunctionName>_<Scenario>
func TestRequest_Body_ValidJSON(t *testing.T) { }
func TestRequest_Body_InvalidJSON(t *testing.T) { }
func TestResponse_Json_WithNilData(t *testing.T) { }
```

### Integration Test Example

```go
func TestEndToEnd_UserCreation(t *testing.T) {
    // Setup
    app := ServerConfig("./test_config.json", "./test_database.json")
    defer app.Stop()
    
    // Create user through API
    req := httptest.NewRequest("POST", "/api/users", 
        strings.NewReader(`{"username":"john","email":"john@test.com"}`))
    rec := httptest.NewRecorder()
    
    // Verify response
    if rec.Code != http.StatusCreated {
        t.Errorf("Expected 201, got %d", rec.Code)
    }
    
    // Verify database
    user := app.ModelQuery("User").FindOne(map[string]interface{}{
        "username": "john",
    })
    if user == nil {
        t.Error("User not created in database")
    }
}
```

### Test Data Management

```go
// Use fixtures for consistent test data
func setupTestDatabase(t *testing.T) *YekongaData {
    app := ServerConfig("./test_config.json", "./test_database.json")
    
    // Clear existing data
    app.ModelQuery("User").DeleteMany()
    
    // Insert test data
    app.ModelQuery("User").CreateMany([]map[string]interface{}{
        {"username": "alice", "email": "alice@test.com"},
        {"username": "bob", "email": "bob@test.com"},
    })
    
    return app
}

func TestUserQuery(t *testing.T) {
    app := setupTestDatabase(t)
    
    users := app.ModelQuery("User").Find(nil)
    if len(*users) != 2 {
        t.Errorf("Expected 2 users, got %d", len(*users))
    }
}
```

### Race Condition Testing

```bash
# Run tests with race detector
go test -race ./...

# Detects concurrent access to shared variables
# Essential for mutex-protected code
```

---

## Logging Standards

### Logging Levels

```go
// Error: Something failed that the application needs to handle
logger.Error("Database connection failed: %v", err)
logger.Error("User creation failed for %s", userId)

// Warning: Something unexpected but not necessarily wrong
logger.Warn("Slow query: %dms", queryTime)
logger.Warn("Retry attempt %d for %s", retryCount, operation)

// Info: Important events
logger.Info("Server started on port %d", port)
logger.Info("User %s logged in", userId)

// Debug: Detailed information for debugging
logger.Debug("Query: %s", sqlQuery)
logger.Debug("Response: %+v", response)
```

### Structured Logging

```go
// ✓ Good: Structured context
logger.Info("User login",
    "userId", userId,
    "email", userEmail,
    "duration", loginDuration,
)

// ✓ Good: Request correlation
requestID := req.RequestID()
logger.Info("Processing request", "requestId", requestID)

// ✗ Bad: Unstructured log message
logger.Info(fmt.Sprintf("User %s at %s logged in after %d seconds", 
    userId, time.Now(), duration))
```

### Error Logging with Context

```go
// Include relevant context in error logs
if err := query.Execute(); err != nil {
    logger.Error("Query execution failed",
        "query", query.String(),
        "model", query.Model.Name,
        "error", err,
    )
}
```

---

## Security Practices

### SQL Injection Prevention

```go
// ✓ Good: Use parameterized queries
query.Where("email", "=", userInput)  // Automatically parameterized

// ✗ Bad: String concatenation
// query.Where("email = '" + userInput + "'")  // VULNERABLE!

// The framework handles this automatically through model query builder
```

### Authentication

```go
// ✓ Good: Always validate authentication
app.Get("/protected", func(req *Request, res *Response) {
    if req.Auth() == nil {
        res.Error("Unauthorized", 401)
        return
    }
    // Handle request
})

// ✗ Bad: Assume user is authenticated
app.Get("/protected", func(req *Request, res *Response) {
    userId := req.Auth().ID  // Panics if nil!
})
```

### Password Hashing

```go
// Use configured hashing algorithm
config := app.Config.Authentication
// Algorithm: bcrypt (recommended)
// SaltRound: 10 (configurable)

// Helper for hashing (if available)
hashedPassword := helper.HashPassword(plainPassword)

// Verify password
isMatch := helper.VerifyPassword(plainPassword, hashedPassword)
```

### Rate Limiting

```go
// Implement rate limiting middleware
app.Use(func(req *Request, res *Response) error {
    clientIP := req.RemoteAddr()
    requestCount := cache.Increment(clientIP)
    
    if requestCount > maxRequestsPerMinute {
        res.Error("Too many requests", 429)
        return fmt.Errorf("rate limit exceeded")
    }
    
    return nil
})
```

### Environment Variables

```go
// ✓ Good: Sensitive config in environment
protocol := os.Getenv("APP_PROTOCOL")
appKey := os.Getenv("APP_KEY")

// ✗ Bad: Hardcoded secrets
const AppKey = "secret123"  // NEVER!

// Store in .env file (not in git)
APP_KEY=your_secret_key_here
DB_PASSWORD=your_password_here
```

### CORS Configuration

```go
// Configure CORS in middleware
func CORSMiddleware(req *Request, res *Response) error {
    res.Header("Access-Control-Allow-Origin", os.Getenv("ALLOWED_ORIGINS"))
    res.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
    res.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
    
    return nil
}

app.Use(CORSMiddleware)
```

---

## Database Design

### Model Definition Best Practices

```json
{
  "models": [
    {
      "name": "User",
      "fields": {
        "id": {
          "type": "id",
          "primary": true,
          "required": true
        },
        "username": {
          "type": "string",
          "required": true,
          "unique": true
        },
        "email": {
          "type": "string",
          "required": true,
          "unique": true
        },
        "status": {
          "type": "string",
          "options": [
            {"value": "active", "label": "Active"},
            {"value": "inactive", "label": "Inactive"}
          ]
        },
        "createdAt": {
          "type": "date",
          "required": true,
          "default": "now"
        },
        "updatedAt": {
          "type": "date"
        }
      }
    }
  ]
}
```

### Field Type Guidelines

```go
// Use appropriate field types for data
"id"         // Auto-generated IDs (use this, not string)
"string"     // Text (< 255 chars)
"text"       // Long text (> 255 chars)
"number"     // Integers
"float"      // Decimal numbers
"bool"       // Boolean (true/false)
"date"       // Timestamps
"object"     // JSON objects
"array"      // JSON arrays
"file"       // File uploads
```

### Relationship Design

```go
// Foreign key relationships
{
  "name": "Post",
  "fields": {
    "userId": {
      "type": "string",
      "foreignKey": "User"  // Reference to User model
    }
  }
}

// Load related data
posts := app.ModelQuery("Post")
    .With("User")  // Include user data
    .Find(nil)
```

### Indexing Strategy

```go
// Add indexes to frequently queried fields
{
  "name": "User",
  "fields": {
    "email": {
      "type": "string",
      "unique": true,  // Also creates index
      "index": true
    },
    "status": {
      "type": "string",
      "index": true  // For WHERE clauses
    },
    "createdAt": {
      "type": "date",
      "index": true  // For sorting
    }
  }
}
```

---

## Common Patterns

### Request Handler Pattern

```go
app.Post("/api/users", func(req *Request, res *Response) {
    // 1. Authentication
    auth := req.Auth()
    if auth == nil {
        res.Error("Unauthorized", 401)
        return
    }
    
    // 2. Parse input
    data := req.Body()
    
    // 3. Validation
    if data["email"] == "" {
        res.Error("Email required", 400)
        return
    }
    
    // 4. Business logic
    user, err := app.ModelQuery("User").Create(data)
    if err != nil {
        res.Error("Failed to create user", 500)
        return
    }
    
    // 5. Response
    res.Status(201)
    res.Json(user)
})
```

### Cloud Function Pattern

```go
app.Define("SendWelcomeEmail", 
    func(data interface{}, rc *RequestContext) (interface{}, error) {
        // Type assertion
        userData, ok := data.(map[string]interface{})
        if !ok {
            return nil, fmt.Errorf("invalid input")
        }
        
        // Extract fields
        email := userData["email"].(string)
        name := userData["name"].(string)
        
        // Perform action
        err := sendEmail(email, "Welcome "+name)
        if err != nil {
            return nil, fmt.Errorf("email failed: %w", err)
        }
        
        // Return result
        return map[string]interface{}{
            "status": "sent",
            "timestamp": time.Now(),
        }, nil
    },
)
```

### Middleware Pattern

```go
func AuthorizationMiddleware(req *Request, res *Response) error {
    // Check permission
    auth := req.Auth()
    if auth == nil {
        return fmt.Errorf("not authenticated")
    }
    
    // Check role
    if auth.Role != "admin" {
        res.Error("Forbidden", 403)
        return fmt.Errorf("insufficient permissions")
    }
    
    // Continue
    return nil
}

app.Use(AuthorizationMiddleware)
```

### Query Builder Pattern

```go
// Complex query with multiple conditions
results := app.ModelQuery("Post")
    .Where("status", "=", "published")
    .Where("views", ">", 100)
    .OrderBy("createdAt", "desc")
    .Take(10)
    .Skip((pageNumber - 1) * 10)
    .Select("id", "title", "content", "viewCount")
    .Find(nil)
```

---

## Code Quality Standards

### Go Formatting

```bash
# Format all files
go fmt ./...

# Use gofmt for consistency
gofmt -w .

# Check formatting
gofmt -l .
```

### Linting

```bash
# Install linter
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run ./...

# Fix issues
golangci-lint run --fix ./...
```

### Documentation

Every exported function should have a comment:

```go
// ServerConfig initializes and returns a new server instance.
// configFile: path to config.json
// databaseFile: path to database.json
// Returns: configured YekongaData instance ready to use
func ServerConfig(configFile string, databaseFile string) *YekongaData {
    // Implementation
}
```

---

## Summary

Key development principles:

1. **Write clear, readable code** - Use descriptive names
2. **Handle errors properly** - Don't ignore or panic unnecessarily
3. **Optimize strategically** - Measure before optimizing
4. **Test thoroughly** - Use tests, especially for concurrent code
5. **Document well** - API docs and comments
6. **Keep security first** - Validate input, protect secrets
7. **Follow conventions** - Consistent with Go standards
8. **Review code carefully** - Before merging to main

These guidelines ensure maintainable, scalable, and secure code across the YekongaServer framework.
