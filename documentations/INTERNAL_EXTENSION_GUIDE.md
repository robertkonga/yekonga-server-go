# YekongaServer - Extension & Customization Guide

Guide for building extensions, plugins, and customizations for YekongaServer.

## Table of Contents

1. [Extension Points](#extension-points)
2. [Custom Middleware](#custom-middleware)
3. [Cloud Functions](#cloud-functions)
4. [Custom Routes](#custom-routes)
5. [Database Triggers](#database-triggers)
6. [GraphQL Extensions](#graphql-extensions)
7. [Plugin Development](#plugin-development)
8. [Data Type Extensions](#data-type-extensions)
9. [Real-World Examples](#real-world-examples)

---

## Extension Points

### Available Hooks & Customization Areas

```
YekongaServer Extension Architecture
│
├── 1. HTTP Layer
│   ├─ Custom routes (GET, POST, PUT, DELETE, etc.)
│   └─ Static file serving configuration
│
├── 2. Middleware Layer
│   ├─ Request validation
│   ├─ Authentication/Authorization
│   ├─ Request logging
│   └─ Custom headers
│
├── 3. API Layer (REST & GraphQL)
│   ├─ Custom REST endpoints
│   ├─ GraphQL actions (mutations)
│   └─ Subscription handlers
│
├── 4. Business Logic Layer
│   ├─ Cloud functions
│   ├─ Before/After triggers
│   └─ Scheduled jobs (cron)
│
├── 5. Data Layer
│   ├─ Data validation
│   ├─ Data transformation
│   └─ Relationship loading
│
└── 6. Helper Functions
    ├─ Custom utilities
    └─ Addon functions
```

---

## Custom Middleware

### Basic Middleware

```go
// Create middleware function
func MyAuthMiddleware(req *yekonga.Request, res *yekonga.Response) error {
    // Extract custom header
    apiKey := req.Header("X-API-Key")
    
    if apiKey == "" {
        res.Error("API key required", 401)
        return fmt.Errorf("missing API key")
    }
    
    // Validate API key
    isValid := ValidateAPIKey(apiKey)
    if !isValid {
        res.Error("Invalid API key", 401)
        return fmt.Errorf("invalid API key")
    }
    
    // Store in context for later use
    req.SetContext("apiKey", apiKey)
    
    // Continue to next middleware/handler
    return nil
}

// Register middleware
func main() {
    app := yekonga.ServerConfig("config.json", "database.json")
    
    // Register as global middleware (runs on all routes)
    app.Use(MyAuthMiddleware)
    
    // Or register as specific type
    app.Middleware(MyAuthMiddleware, yekonga.InitMiddleware)
    
    app.Start(":8080")
}
```

### Conditional Middleware

```go
// Only apply to certain routes
func ConditionalMiddleware(req *yekonga.Request, res *yekonga.Response) error {
    // Check if route needs protection
    protectedRoutes := []string{"/admin", "/api/protected"}
    
    path := req.Path()
    isProtected := false
    for _, route := range protectedRoutes {
        if strings.HasPrefix(path, route) {
            isProtected = true
            break
        }
    }
    
    if !isProtected {
        return nil  // Skip middleware for other routes
    }
    
    // Apply protection
    auth := req.Auth()
    if auth == nil {
        res.Error("Unauthorized", 401)
        return fmt.Errorf("not authenticated")
    }
    
    return nil
}

app.Use(ConditionalMiddleware)
```

### Middleware Chain

```go
// Multiple middlewares execute in registration order

// 1. Logging middleware (first)
app.Use(func(req *yekonga.Request, res *yekonga.Response) error {
    startTime := time.Now()
    req.SetContext("startTime", startTime)
    return nil
})

// 2. Authentication middleware
app.Use(func(req *yekonga.Request, res *yekonga.Response) error {
    if req.Auth() == nil {
        res.Error("Unauthorized", 401)
        return fmt.Errorf("not authenticated")
    }
    return nil
})

// 3. Authorization middleware
app.Use(func(req *yekonga.Request, res *yekonga.Response) error {
    role := req.Auth().Role
    if role != "admin" {
        res.Error("Forbidden", 403)
        return fmt.Errorf("insufficient permissions")
    }
    return nil
})

// 4. Request validation
app.Use(func(req *yekonga.Request, res *yekonga.Response) error {
    contentType := req.Header("Content-Type")
    if req.Method() == "POST" && contentType != "application/json" {
        res.Error("Content-Type must be application/json", 400)
        return fmt.Errorf("invalid content type")
    }
    return nil
})

// Handler only executes if all middlewares return nil
app.Post("/api/data", func(req *yekonga.Request, res *yekonga.Response) {
    // All validations passed
    data := req.Body()
    res.Json(data)
})
```

### Context-Based Middleware Communication

```go
// Store computed values in context for reuse
app.Use(func(req *yekonga.Request, res *yekonga.Response) error {
    // Expensive operation
    user := app.ModelQuery("User").FindById(req.Auth().ID)
    req.SetContextObject("currentUser", user)
    return nil
})

// Later middleware can access
app.Use(func(req *yekonga.Request, res *yekonga.Response) error {
    user := req.GetContextObject("currentUser")
    userRole := (*user.(*yekonga.DataMap))["role"]
    
    if userRole != "admin" {
        return fmt.Errorf("admin only")
    }
    return nil
})

// Handler can also access
app.Get("/profile", func(req *yekonga.Request, res *yekonga.Response) {
    user := req.GetContextObject("currentUser")
    res.Json(user)
})
```

---

## Cloud Functions

### Basic Function Definition

```go
// Define cloud function
app.Define("ProcessData", 
    func(data interface{}, rc *yekonga.RequestContext) (interface{}, error) {
        // Type assertion
        input, ok := data.(map[string]interface{})
        if !ok {
            return nil, fmt.Errorf("invalid input type")
        }
        
        // Process data
        result := ProcessDataLogic(input)
        
        // Return result
        return map[string]interface{}{
            "success": true,
            "data": result,
        }, nil
    },
)
```

### Function with Context

```go
// Use request context to access authenticated user
app.Define("CreateUserPost",
    func(data interface{}, rc *yekonga.RequestContext) (interface{}, error) {
        // Get authenticated user
        auth := rc.Auth
        if auth == nil {
            return nil, fmt.Errorf("not authenticated")
        }
        
        // Get application
        app := rc.App
        
        // Parse input
        postData, ok := data.(map[string]interface{})
        if !ok {
            return nil, fmt.Errorf("invalid input")
        }
        
        // Add user ID
        postData["userId"] = auth.ID
        
        // Create post
        post, err := app.ModelQuery("Post").Create(postData)
        if err != nil {
            return nil, fmt.Errorf("failed to create post: %w", err)
        }
        
        return post, nil
    },
)
```

### Function Calling

```go
// Call from route handler
app.Get("/data/process", func(req *yekonga.Request, res *yekonga.Response) {
    // Prepare input
    input := req.Body()
    
    // Call function
    result, err := app.CallFunction("ProcessData", input)
    if err != nil {
        res.Error(err.Error(), 500)
        return
    }
    
    res.Json(result)
})

// Or call function with automatic context
app.Post("/posts/create", func(req *yekonga.Request, res *yekonga.Response) {
    input := req.Body()
    
    // Framework automatically builds context with auth info
    result, err := app.CallFunctionWithContext("CreateUserPost", input, 
        &yekonga.RequestContext{
            Auth: req.Auth(),
            App: req.App,
            Request: req,
        },
    )
    
    if err != nil {
        res.Error(err.Error(), 400)
        return
    }
    
    res.Json(result)
})

// Async execution (fire and forget)
go app.CallFunctionAsync("SendEmail", emailData)
```

---

## Custom Routes

### Basic Custom Routes

```go
// Simple GET route
app.Get("/hello", func(req *yekonga.Request, res *yekonga.Response) {
    res.Text("Hello World")
})

// GET with path parameters
app.Get("/users/:id", func(req *yekonga.Request, res *yekonga.Response) {
    id := req.Param("id")
    user := app.ModelQuery("User").FindById(id)
    
    if user == nil {
        res.Error("User not found", 404)
        return
    }
    
    res.Json(user)
})

// GET with query parameters
app.Get("/search", func(req *yekonga.Request, res *yekonga.Response) {
    query := req.Query("q")
    
    results := app.ModelQuery("Post")
        .Where("title", "contains", query)
        .Find(nil)
    
    res.Json(results)
})
```

### POST/PUT/DELETE Routes

```go
// POST - Create resource
app.Post("/api/users", func(req *yekonga.Request, res *yekonga.Response) {
    data := req.Body()
    
    // Validate
    if data["email"] == nil {
        res.Error("Email is required", 400)
        return
    }
    
    // Create
    user, err := app.ModelQuery("User").Create(data)
    if err != nil {
        res.Error("Failed to create user", 500)
        return
    }
    
    res.Status(201)
    res.Json(user)
})

// PUT - Update resource
app.Put("/api/users/:id", func(req *yekonga.Request, res *yekonga.Response) {
    id := req.Param("id")
    data := req.Body()
    
    user, err := app.ModelQuery("User").Update(id, data)
    if err != nil {
        res.Error("Update failed", 500)
        return
    }
    
    res.Json(user)
})

// DELETE - Delete resource
app.Delete("/api/users/:id", func(req *yekonga.Request, res *yekonga.Response) {
    id := req.Param("id")
    
    _, err := app.ModelQuery("User").Delete(id)
    if err != nil {
        res.Error("Delete failed", 500)
        return
    }
    
    res.Json(map[string]string{"status": "deleted"})
})
```

### Advanced Routing

```go
// Pattern with multiple parameters
app.Get("/users/:userId/posts/:postId", func(req *yekonga.Request, res *yekonga.Response) {
    userId := req.Param("userId")
    postId := req.Param("postId")
    
    post := app.ModelQuery("Post")
        .Where("_id", "=", postId)
        .Where("userId", "=", userId)
        .FindOne(nil)
    
    if post == nil {
        res.Error("Post not found", 404)
        return
    }
    
    res.Json(post)
})

// File download
app.Get("/download/:fileId", func(req *yekonga.Request, res *yekonga.Response) {
    fileId := req.Param("fileId")
    filePath := fmt.Sprintf("./uploads/%s", fileId)
    
    if !helper.FileExists(filePath) {
        res.Error("File not found", 404)
        return
    }
    
    res.Download(filePath, fileId)
})

// File upload
app.Post("/upload", func(req *yekonga.Request, res *yekonga.Response) {
    // Parse multipart form
    req.HttpRequest.ParseMultipartForm(10 << 20)  // 10MB max
    
    file, handler, err := req.HttpRequest.FormFile("file")
    if err != nil {
        res.Error("No file uploaded", 400)
        return
    }
    defer file.Close()
    
    // Save file
    dst, err := os.Create(fmt.Sprintf("./uploads/%s", handler.Filename))
    if err != nil {
        res.Error("Failed to save file", 500)
        return
    }
    defer dst.Close()
    
    io.Copy(dst, file)
    res.Json(map[string]string{"filename": handler.Filename})
})
```

---

## Database Triggers

### Create Triggers

```go
// Before create - validate/transform data
app.BeforeCreate("User", 
    func(rc *yekonga.RequestContext, qc *yekonga.QueryContext) (interface{}, error) {
        data := qc.Data.(*yekonga.DataMap)
        
        // Validate required fields
        if (*data)["username"] == "" || (*data)["email"] == "" {
            return nil, fmt.Errorf("username and email required")
        }
        
        // Hash password before insert
        if password, ok := (*data)["password"]; ok {
            hashedPassword := HashPassword(password.(string))
            (*data)["password"] = hashedPassword
        }
        
        // Add timestamp
        (*data)["createdAt"] = time.Now()
        
        return data, nil
    },
)

// After create - side effects
app.AfterCreate("User",
    func(rc *yekonga.RequestContext, qc *yekonga.QueryContext) (interface{}, error) {
        user := qc.Data.(*yekonga.DataMap)
        email := (*user)["email"].(string)
        
        // Send welcome email
        go SendWelcomeEmail(email)
        
        // Log event
        logger.Info("User created", "email", email)
        
        // Don't modify data - just side effects
        return user, nil
    },
)
```

### Update Triggers

```go
// Before update - validate changes
app.BeforeUpdate("User",
    func(rc *yekonga.RequestContext, qc *yekonga.QueryContext) (interface{}, error) {
        newData := qc.Data.(*yekonga.DataMap)
        oldData := qc.PreviousData.(*yekonga.DataMap)
        
        // Prevent certain fields from changing
        if (*oldData)["id"] != (*newData)["id"] {
            return nil, fmt.Errorf("cannot change user ID")
        }
        
        // Track change for audit
        rc.Request.SetContext("changes", map[string]interface{}{
            "old": oldData,
            "new": newData,
        })
        
        return newData, nil
    },
)

// After update - track changes
app.AfterUpdate("User",
    func(rc *yekonga.RequestContext, qc *yekonga.QueryContext) (interface{}, error) {
        user := qc.Data.(*yekonga.DataMap)
        changes := rc.Request.GetContext("changes")
        
        // Log audit trail
        LogAuditTrail("user_updated", rc.Auth.ID, changes)
        
        return user, nil
    },
)
```

### Delete Triggers

```go
// Before delete - check references
app.BeforeDelete("User",
    func(rc *yekonga.RequestContext, qc *yekonga.QueryContext) (interface{}, error) {
        userId := qc.Data.(*yekonga.DataMap)
        
        // Check if user has posts
        postCount := rc.App.ModelQuery("Post")
            .Where("userId", "=", userId)
            .Count()
        
        if postCount > 0 {
            return nil, fmt.Errorf("user has %d posts, cannot delete", postCount)
        }
        
        return qc.Data, nil
    },
)

// After delete - cleanup
app.AfterDelete("User",
    func(rc *yekonga.RequestContext, qc *yekonga.QueryContext) (interface{}, error) {
        userId := qc.Data.(*yekonga.DataMap)
        
        // Delete related data
        rc.App.ModelQuery("UserSettings").Where("userId", "=", userId).Delete()
        
        // Log deletion
        logger.Info("User deleted", "userId", userId)
        
        return qc.Data, nil
    },
)
```

### Find Triggers

```go
// After find - transform results
app.AfterFind("User",
    func(rc *yekonga.RequestContext, qc *yekonga.QueryContext) (interface{}, error) {
        users := qc.Data.(*[]yekonga.DataMap)
        
        // Remove sensitive fields
        for _, user := range *users {
            delete(user, "password")
            delete(user, "internalNotes")
        }
        
        // Add computed fields
        for _, user := range *users {
            userId := user["id"]
            postCount := rc.App.ModelQuery("Post")
                .Where("userId", "=", userId)
                .Count()
            user["postCount"] = postCount
        }
        
        return users, nil
    },
)
```

---

## GraphQL Extensions

### Custom GraphQL Action

```go
// Register action on User model
app.RegisterGraphqlAction("User", "activate",
    func(rc *yekonga.RequestContext, qc *yekonga.QueryContext) (yekonga.GraphqlActionResult, error) {
        // qc.Data contains action parameters
        userId := qc.Data.(*yekonga.DataMap)
        
        // Execute action
        user, err := app.ModelQuery("User").Update(userId, map[string]interface{}{
            "status": "active",
        })
        
        if err != nil {
            return yekonga.GraphqlActionResult{
                Success: false,
                Message: "Failed to activate user",
            }, err
        }
        
        return yekonga.GraphqlActionResult{
            Data: user,
            Success: true,
            Message: "User activated successfully",
        }, nil
    },
)

// GraphQL mutation available as:
// mutation {
//   User {
//     activate(id: "123") {
//       success
//       message
//       data { id username email }
//     }
//   }
// }
```

### Subscription Handler

```go
// Handle GraphQL subscriptions
app.RegisterGraphqlSubscription("User", "onUserCreated",
    func(rc *yekonga.RequestContext) (interface{}, error) {
        // Return data to send to subscriber
        return map[string]interface{}{
            "type": "user_created",
            "timestamp": time.Now(),
        }, nil
    },
)
```

---

## Plugin Development

### Creating a Custom Plugin

Plugin structure:

```
plugins/my-plugin/
├── plugin.go          # Main plugin logic
├── types.go           # Type definitions
└── README.md          # Documentation
```

Example plugin:

```go
// plugins/custom-auth/auth.go
package customauth

import (
    "fmt"
    yekonga "github.com/robertkonga/yekonga-server-go/yekonga"
)

// CustomAuthProvider provides custom authentication
type CustomAuthProvider struct {
    secretKey string
}

// NewCustomAuthProvider creates new instance
func NewCustomAuthProvider(secretKey string) *CustomAuthProvider {
    return &CustomAuthProvider{
        secretKey: secretKey,
    }
}

// Authenticate validates credentials
func (cap *CustomAuthProvider) Authenticate(username, password string) (*yekonga.AuthPayload, error) {
    // Custom authentication logic
    if username == "" || password == "" {
        return nil, fmt.Errorf("invalid credentials")
    }
    
    // Validate password (your custom logic)
    if !cap.ValidatePassword(username, password) {
        return nil, fmt.Errorf("invalid credentials")
    }
    
    // Return auth payload
    return &yekonga.AuthPayload{
        ID: username,
        Username: username,
        Role: "user",
    }, nil
}

// ValidatePassword checks password (implementation)
func (cap *CustomAuthProvider) ValidatePassword(username, password string) bool {
    // Your validation logic
    return true
}
```

Using the plugin:

```go
import customauth "github.com/yourorg/yekonga-plugins/custom-auth"

func main() {
    app := yekonga.ServerConfig("config.json", "database.json")
    
    // Initialize plugin
    authProvider := customauth.NewCustomAuthProvider("secret123")
    
    // Use in middleware
    app.Use(func(req *yekonga.Request, res *yekonga.Response) error {
        username := req.Header("X-Username")
        password := req.Header("X-Password")
        
        auth, err := authProvider.Authenticate(username, password)
        if err != nil {
            res.Error("Authentication failed", 401)
            return err
        }
        
        // Set in request
        req.SetContextObject("auth", auth)
        return nil
    })
    
    app.Start(":8080")
}
```

---

## Data Type Extensions

### Custom Data Types

```go
// Define custom type
type CustomData struct {
    ID    string
    Name  string
    Value interface{}
}

// Add custom methods
func (cd *CustomData) ToMap() map[string]interface{} {
    return map[string]interface{}{
        "id": cd.ID,
        "name": cd.Name,
        "value": cd.Value,
    }
}

// Use in handler
app.Get("/custom", func(req *yekonga.Request, res *yekonga.Response) {
    data := CustomData{
        ID: "123",
        Name: "test",
        Value: "some value",
    }
    
    res.Json(data.ToMap())
})
```

### Extending DataMap

```go
// Create helper for custom fields
func AddComputedField(dataMap *yekonga.DataMap, fieldName string, compute func() interface{}) {
    (*dataMap)[fieldName] = compute()
}

// Use in AfterFind trigger
app.AfterFind("Product",
    func(rc *yekonga.RequestContext, qc *yekonga.QueryContext) (interface{}, error) {
        products := qc.Data.(*[]yekonga.DataMap)
        
        for _, product := range *products {
            // Add computed price with tax
            originalPrice := product["price"].(float64)
            product["priceWithTax"] = originalPrice * 1.10
            
            // Add availability
            stock := product["stock"].(int)
            product["available"] = stock > 0
        }
        
        return products, nil
    },
)
```

---

## Real-World Examples

### Example 1: User Registration System

```go
// Register new user
app.Post("/auth/register", func(req *yekonga.Request, res *yekonga.Response) {
    data := req.Body()
    
    // Validation
    email := data["email"].(string)
    if !helper.IsEmail(email) {
        res.Error("Invalid email", 400)
        return
    }
    
    // Check duplicate
    existing := app.ModelQuery("User").FindOne(map[string]interface{}{
        "email": email,
    })
    if existing != nil {
        res.Error("Email already registered", 409)
        return
    }
    
    // Create user (BeforeCreate trigger will hash password)
    user, err := app.ModelQuery("User").Create(data)
    if err != nil {
        res.Error("Registration failed", 500)
        return
    }
    
    res.Status(201)
    res.Json(user)
})

// BeforeCreate trigger
app.BeforeCreate("User", func(rc *yekonga.RequestContext, qc *yekonga.QueryContext) (interface{}, error) {
    data := qc.Data.(*yekonga.DataMap)
    
    // Hash password
    password := (*data)["password"].(string)
    hashedPassword := hashPassword(password)
    (*data)["password"] = hashedPassword
    
    // Add defaults
    (*data)["status"] = "active"
    (*data)["createdAt"] = time.Now()
    
    return data, nil
})

// AfterCreate trigger
app.AfterCreate("User", func(rc *yekonga.RequestContext, qc *yekonga.QueryContext) (interface{}, error) {
    user := qc.Data.(*yekonga.DataMap)
    email := (*user)["email"].(string)
    
    // Send verification email
    go sendVerificationEmail(email, (*user)["id"].(string))
    
    return user, nil
})
```

### Example 2: Admin Dashboard with Authorization

```go
// Admin authorization middleware
func AdminMiddleware(req *yekonga.Request, res *yekonga.Response) error {
    auth := req.Auth()
    if auth == nil {
        res.Error("Not authenticated", 401)
        return fmt.Errorf("no auth")
    }
    
    if auth.Role != "admin" {
        res.Error("Admin access required", 403)
        return fmt.Errorf("insufficient permissions")
    }
    
    return nil
}

// Admin routes
app.Use(AdminMiddleware)

// Get all users
app.Get("/admin/users", func(req *yekonga.Request, res *yekonga.Response) {
    users := app.ModelQuery("User")
        .Select("id", "username", "email", "status", "createdAt")
        .OrderBy("createdAt", "desc")
        .Find(nil)
    
    res.Json(users)
})

// Deactivate user
app.Put("/admin/users/:id/deactivate", func(req *yekonga.Request, res *yekonga.Response) {
    userId := req.Param("id")
    
    user, err := app.ModelQuery("User").Update(userId, map[string]interface{}{
        "status": "inactive",
    })
    
    if err != nil {
        res.Error("Update failed", 500)
        return
    }
    
    res.Json(user)
})
```

### Example 3: E-Commerce Order Processing

```go
// Create order with automatic calculations
app.Post("/orders", func(req *yekonga.Request, res *yekonga.Response) {
    auth := req.Auth()
    if auth == nil {
        res.Error("Not authenticated", 401)
        return
    }
    
    data := req.Body()
    data["userId"] = auth.ID
    
    order, err := app.ModelQuery("Order").Create(data)
    if err != nil {
        res.Error("Failed to create order", 500)
        return
    }
    
    res.Status(201)
    res.Json(order)
})

// BeforeCreate - Calculate total
app.BeforeCreate("Order", func(rc *yekonga.RequestContext, qc *yekonga.QueryContext) (interface{}, error) {
    data := qc.Data.(*yekonga.DataMap)
    items := (*data)["items"].([]interface{})
    
    total := 0.0
    tax := 0.0
    
    // Calculate totals
    for _, item := range items {
        itemMap := item.(map[string]interface{})
        price := itemMap["price"].(float64)
        quantity := itemMap["quantity"].(float64)
        total += price * quantity
    }
    
    tax = total * 0.1  // 10% tax
    
    (*data)["subtotal"] = total
    (*data)["tax"] = tax
    (*data)["total"] = total + tax
    (*data)["status"] = "pending"
    (*data)["createdAt"] = time.Now()
    
    return data, nil
})

// AfterCreate - Trigger fulfillment
app.AfterCreate("Order", func(rc *yekonga.RequestContext, qc *yekonga.QueryContext) (interface{}, error) {
    order := qc.Data.(*yekonga.DataMap)
    orderId := (*order)["id"]
    
    // Queue for fulfillment
    go ProcessOrderFulfillment(orderId)
    
    // Send confirmation email
    user := rc.Request.GetContextObject("currentUser").(*yekonga.DataMap)
    email := (*user)["email"].(string)
    go SendOrderConfirmation(email, orderId)
    
    return order, nil
})
```

---

## Testing Extensions

### Testing Custom Middleware

```go
func TestCustomAuthMiddleware(t *testing.T) {
    // Create mock request
    req := &yekonga.Request{
        HttpRequest: &http.Request{
            Header: http.Header{
                "X-API-Key": []string{"valid-key"},
            },
        },
    }
    
    res := &yekonga.Response{}
    
    // Test middleware
    err := MyAuthMiddleware(req, res)
    
    if err != nil {
        t.Errorf("Expected no error, got %v", err)
    }
    
    // Verify context set
    key := req.GetContext("apiKey")
    if key != "valid-key" {
        t.Errorf("Expected apiKey in context")
    }
}
```

---

## Summary

Extension points overview:

| Extension Type | Use Case | Complexity |
|---|---|---|
| **Middleware** | Request validation, authentication | Easy |
| **Routes** | Custom endpoints | Easy |
| **Cloud Functions** | Business logic, async work | Medium |
| **Triggers** | Data validation, side effects | Medium |
| **GraphQL Actions** | Complex mutations | Medium |
| **Plugins** | Reusable modules | Hard |
| **Data Types** | Custom data structures | Easy |

The framework is designed to be highly extensible while maintaining security and performance standards.
