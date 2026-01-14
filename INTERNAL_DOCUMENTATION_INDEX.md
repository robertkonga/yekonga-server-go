# YekongaServer - Internal Documentation Index

Welcome to the comprehensive internal development documentation for YekongaServer. This index guides you through all available documentation.

## Quick Navigation

### 📋 Documentation Files

| Document | Purpose | Audience | Time to Read |
|----------|---------|----------|--------------|
| [INTERNAL_ARCHITECTURE.md](INTERNAL_ARCHITECTURE.md) | System design, layered architecture, core concepts | Architects, Lead Developers | 20-30 min |
| [INTERNAL_API_REFERENCE.md](INTERNAL_API_REFERENCE.md) | Complete API reference, all functions, interfaces | All Developers | 40-50 min |
| [INTERNAL_MODULES.md](INTERNAL_MODULES.md) | Deep dive into each module, implementation details | Technical Leads, Contributors | 30-40 min |
| [INTERNAL_DEVELOPMENT_GUIDE.md](INTERNAL_DEVELOPMENT_GUIDE.md) | Coding standards, best practices, guidelines | All Developers | 25-35 min |
| [INTERNAL_EXTENSION_GUIDE.md](INTERNAL_EXTENSION_GUIDE.md) | How to extend the framework, plugins, customization | Feature Developers | 30-40 min |

---

## Learning Paths

### 👤 For New Team Members

1. **Start here**: [INTERNAL_ARCHITECTURE.md](INTERNAL_ARCHITECTURE.md)
   - Understand system design and how components interact
   - Learn about the request/response lifecycle
   - Get familiar with core concepts

2. **Then read**: [INTERNAL_API_REFERENCE.md](INTERNAL_API_REFERENCE.md)
   - Learn the public API
   - Understand available methods
   - See practical examples

3. **Implementation**: [INTERNAL_DEVELOPMENT_GUIDE.md](INTERNAL_DEVELOPMENT_GUIDE.md)
   - Follow coding standards
   - Apply best practices
   - Learn naming conventions

### 🏗️ For Feature Developers

1. **Architecture**: [INTERNAL_ARCHITECTURE.md](INTERNAL_ARCHITECTURE.md) (Extension Points section)
2. **Implementation**: [INTERNAL_EXTENSION_GUIDE.md](INTERNAL_EXTENSION_GUIDE.md)
3. **Standards**: [INTERNAL_DEVELOPMENT_GUIDE.md](INTERNAL_DEVELOPMENT_GUIDE.md)

### 🔧 For Maintainers

1. **Module Deep Dive**: [INTERNAL_MODULES.md](INTERNAL_MODULES.md)
2. **API Reference**: [INTERNAL_API_REFERENCE.md](INTERNAL_API_REFERENCE.md)
3. **Development Guide**: [INTERNAL_DEVELOPMENT_GUIDE.md](INTERNAL_DEVELOPMENT_GUIDE.md)

### 📚 For Technical Writers

- All files contain detailed explanations suitable for documentation generation
- Use [INTERNAL_ARCHITECTURE.md](INTERNAL_ARCHITECTURE.md) for conceptual diagrams
- Use [INTERNAL_MODULES.md](INTERNAL_MODULES.md) for implementation details

---

## Document Summaries

### INTERNAL_ARCHITECTURE.md

**Covers:**
- High-level system architecture with diagrams
- Layer model (Transport, API, Middleware, Business Logic, Data, Database)
- Request/Response lifecycle with detailed flow
- Data model system and field types
- Database abstraction layer with multi-database support
- Middleware pipeline and execution order
- Type system and context keys
- Concurrency and thread safety mechanisms
- Extension points overview
- Performance considerations
- Error handling strategy

**Key Diagrams:**
- System architecture overview
- Layer model visualization
- Request/Response flow
- Data Model structure
- Database abstraction strategy

### INTERNAL_API_REFERENCE.md

**Covers:**
- Core server APIs (YekongaData)
  - Routing methods (Get, Post, Put, Delete, etc.)
  - Middleware management
  - Static file serving
  - Server lifecycle (Start, Stop, HomeDirectory)
  - Model management
  - Configuration access
  - Cloud functions
  - Cron jobs
- Request/Response APIs with detailed method listings
- Data Model APIs
- Query Builder APIs with chainable and execution methods
- Cloud Functions APIs (function definition, triggers, context)
- Context APIs (RequestContext)
- Middleware APIs
- Database Connection APIs
- Data type definitions

**Quick Reference Tables:**
- HTTP routing methods
- Query builder operators
- Aggregation methods
- Error response standards

### INTERNAL_MODULES.md

**Covers:**
- Core module structure and file organization
- File responsibilities and line counts
- Initialization flow with detailed steps
- Request/Response processing pipeline
- Context management and isolation
- Database connections architecture
  - Connection strategy pattern
  - MongoDB, MySQL, SQL, Local DB specifics
  - Query execution flow
- Model Query system architecture
  - Query builder patterns
  - Method chaining
  - Query compilation to database-specific formats
  - Supported operators
- GraphQL integration
  - Schema generation flow
  - Type mapping examples
  - Query execution pipeline
  - Subscription handling
- Cloud Functions system
  - Function registry structure
  - Trigger execution points
  - Function calling patterns
- Helper Functions overview
- Plugin System architecture
- Module interactions diagram

**Performance Characteristics:**
- Latency points for operations
- Memory usage estimates
- Concurrency model

### INTERNAL_DEVELOPMENT_GUIDE.md

**Covers:**
- Code organization principles
- Package structure and conventions
- Development workflow
  - Setting up environment
  - Local development setup
  - Branch strategy
  - Commit message format
  - Code review checklist
- Naming conventions
  - Type names
  - Function names
  - Variable names
  - Constant names
- Error handling
  - Error return conventions
  - Error response standards
  - Error logging
  - Panic usage
- Performance guidelines
  - Database query optimization
  - Middleware performance
  - Goroutine usage
  - Caching strategies
- Testing strategies
  - Unit test structure
  - Integration testing
  - Test data management
  - Race condition testing
- Logging standards
  - Logging levels
  - Structured logging
  - Error logging with context
- Security practices
  - SQL injection prevention
  - Authentication
  - Password hashing
  - Rate limiting
  - Environment variables
  - CORS configuration
- Database design
  - Model definition best practices
  - Field type guidelines
  - Relationship design
  - Indexing strategy
- Common patterns
  - Request handler pattern
  - Cloud function pattern
  - Middleware pattern
  - Query builder pattern
- Code quality standards

### INTERNAL_EXTENSION_GUIDE.md

**Covers:**
- Extension point overview with architecture diagram
- Custom Middleware
  - Basic middleware
  - Conditional middleware
  - Middleware chains
  - Context-based communication
- Cloud Functions
  - Basic functions
  - Functions with context
  - Function calling patterns
- Custom Routes
  - Basic routes
  - POST/PUT/DELETE routes
  - Advanced routing (multiple parameters, files, uploads)
- Database Triggers
  - Create triggers (Before/After)
  - Update triggers
  - Delete triggers
  - Find triggers
- GraphQL Extensions
  - Custom GraphQL actions
  - Subscription handlers
- Plugin Development
  - Creating custom plugins
  - Plugin usage
- Data Type Extensions
  - Custom data types
  - Extending DataMap
- Real-World Examples
  - User registration system
  - Admin dashboard with authorization
  - E-Commerce order processing
- Testing extensions
- Extension summary table

---

## Key Concepts Reference

### Core Components

**YekongaData** - Main server instance
- Holds routes, middlewares, functions, models
- Manages database connections
- Coordinates request handling

**Request/Response** - HTTP abstraction
- Abstracts Go's net/http
- Provides context management
- Handles body parsing and writing

**DataModel** - Schema representation
- Defines database tables/collections
- Specifies field types and constraints
- Manages relationships

**ModelQuery** - Database query builder
- Fluent interface for queries
- Supports chaining
- Database-agnostic

**Middleware** - Request processing
- Three types: Init, Preload, Global
- Can modify request/response
- Can store context for handlers

**Cloud Functions** - Custom business logic
- User-defined functions
- Can be triggered or called explicitly
- Have access to request context

---

## Common Tasks

### "I want to..."

#### Add a new API endpoint
1. Read: [INTERNAL_EXTENSION_GUIDE.md](INTERNAL_EXTENSION_GUIDE.md#custom-routes)
2. Follow: [INTERNAL_DEVELOPMENT_GUIDE.md](INTERNAL_DEVELOPMENT_GUIDE.md#common-patterns)

#### Understand the request lifecycle
1. Read: [INTERNAL_ARCHITECTURE.md](INTERNAL_ARCHITECTURE.md#requestresponse-lifecycle)
2. Reference: [INTERNAL_MODULES.md](INTERNAL_MODULES.md#requestresponse-processing)

#### Add middleware
1. Read: [INTERNAL_EXTENSION_GUIDE.md](INTERNAL_EXTENSION_GUIDE.md#custom-middleware)
2. See examples: [INTERNAL_EXTENSION_GUIDE.md](INTERNAL_EXTENSION_GUIDE.md#example-2-admin-dashboard-with-authorization)

#### Create a cloud function
1. Read: [INTERNAL_EXTENSION_GUIDE.md](INTERNAL_EXTENSION_GUIDE.md#cloud-functions)
2. See patterns: [INTERNAL_API_REFERENCE.md](INTERNAL_API_REFERENCE.md#cloud-functions-apis)

#### Optimize database queries
1. Read: [INTERNAL_DEVELOPMENT_GUIDE.md](INTERNAL_DEVELOPMENT_GUIDE.md#database-query-optimization)
2. Reference: [INTERNAL_API_REFERENCE.md](INTERNAL_API_REFERENCE.md#query-builder-apis)

#### Handle errors properly
1. Read: [INTERNAL_DEVELOPMENT_GUIDE.md](INTERNAL_DEVELOPMENT_GUIDE.md#error-handling)
2. See standards: [INTERNAL_API_REFERENCE.md](INTERNAL_API_REFERENCE.md#error-responses)

#### Extend the framework
1. Start: [INTERNAL_EXTENSION_GUIDE.md](INTERNAL_EXTENSION_GUIDE.md#extension-points)
2. Follow: [INTERNAL_DEVELOPMENT_GUIDE.md](INTERNAL_DEVELOPMENT_GUIDE.md)

#### Add database triggers
1. Read: [INTERNAL_EXTENSION_GUIDE.md](INTERNAL_EXTENSION_GUIDE.md#database-triggers)
2. See full example: [INTERNAL_EXTENSION_GUIDE.md](INTERNAL_EXTENSION_GUIDE.md#example-3-e-commerce-order-processing)

---

## Architecture Overview

```
YekongaServer Architecture (from INTERNAL_ARCHITECTURE.md)

HTTP Server (net/http)
    ↓
Router (REST/GraphQL/WebSocket)
    ↓
Request Wrapper (Request object)
    ↓
Middleware Pipeline
├─ Init Middlewares
├─ Preload Middlewares  
└─ Global Middlewares
    ↓
Route Handler / GraphQL Executor
    ↓
Cloud Functions / Database Access
    ↓
Database Layer (Multi-DB Support)
├─ MongoDB
├─ MySQL
├─ SQL
└─ Local JSON DB
    ↓
Response (HTTP response)
```

---

## Module Organization

```
Main Package: yekonga/
├── main.go (581 lines)
│   ├── YekongaData (server instance)
│   ├── ServerConfig() (initialization)
│   └── Route handling
├── request.go (326 lines)
│   ├── Request wrapper
│   ├── RequestContext
│   └── Parameter extraction
├── response.go
│   ├── Response writer
│   └── JSON/HTML/File responses
├── middleware.go (123 lines)
│   ├── TokenMiddleware
│   ├── UserInfoMiddleware
│   └── ApplicationKeyMiddleware
├── model.go (403 lines)
│   ├── DataModel definition
│   ├── DataModelField types
│   └── Field categorization
├── model_query.go (500+ lines)
│   ├── ModelQuery builder
│   ├── Where/OrderBy/Take/Skip
│   └── CRUD operations
├── graphql.go (1883 lines)
│   ├── GraphQL schema building
│   ├── Query/Mutation execution
│   └── Subscription handling
├── cloud_functions.go
│   ├── Function registry
│   ├── Trigger execution
│   └── Function calling
├── cronjob.go
│   ├── Scheduled job management
│   └── Cron execution
├── database_structure.go
│   ├── Schema parsing
│   └── Model initialization
├── dbconnect.go (184 lines)
│   ├── DatabaseConnections
│   └── Multi-DB coordination
├── dbconnect_mongodb.go
│   ├── MongoDB driver
│   └── Connection management
├── dbconnect_mysql.go
│   ├── MySQL driver
│   └── Connection management
├── dbconnect_sql.go
│   ├── SQL driver
│   └── Connection management
└── dbconnect_local.go
    ├── Local JSON database
    └── Development database
```

---

## Code Style & Standards

### Naming Conventions
- **Types**: PascalCase (YekongaData, DataModel)
- **Functions**: camelCase (serverConfig, parseQuery)
- **Constants**: SCREAMING_SNAKE_CASE or CamelCase (DataModelID)
- **Variables**: camelCase (user, query, data)

### Error Handling
- Return error as second value: `(result, error)`
- Middleware returns error to stop pipeline
- Use error wrapping: `fmt.Errorf("message: %w", err)`
- Log errors with context

### Performance
- Cache expensive operations
- Paginate large queries
- Order middlewares by cost
- Use goroutines for I/O-bound work

### Security
- Validate all input
- Use parameterized queries (automatic)
- Hash passwords (bcrypt)
- Protect sensitive config (environment variables)

---

## File Cross-Reference

### By Topic

**Architecture & Design**
- [INTERNAL_ARCHITECTURE.md](INTERNAL_ARCHITECTURE.md)
- [INTERNAL_MODULES.md](INTERNAL_MODULES.md)

**Implementation Details**
- [INTERNAL_API_REFERENCE.md](INTERNAL_API_REFERENCE.md)
- [INTERNAL_MODULES.md](INTERNAL_MODULES.md)

**Development Standards**
- [INTERNAL_DEVELOPMENT_GUIDE.md](INTERNAL_DEVELOPMENT_GUIDE.md)

**Extension & Customization**
- [INTERNAL_EXTENSION_GUIDE.md](INTERNAL_EXTENSION_GUIDE.md)
- [INTERNAL_DEVELOPMENT_GUIDE.md](INTERNAL_DEVELOPMENT_GUIDE.md#common-patterns)

**Code Examples**
- [INTERNAL_EXTENSION_GUIDE.md](INTERNAL_EXTENSION_GUIDE.md#real-world-examples)
- [INTERNAL_DEVELOPMENT_GUIDE.md](INTERNAL_DEVELOPMENT_GUIDE.md#common-patterns)

---

## Contributing

When contributing to YekongaServer:

1. **Read** the relevant documentation sections
2. **Follow** the [Development Guide](INTERNAL_DEVELOPMENT_GUIDE.md)
3. **Use** the [Extension Guide](INTERNAL_EXTENSION_GUIDE.md) if adding features
4. **Reference** the [API Reference](INTERNAL_API_REFERENCE.md) for correct usage
5. **Understand** the [Architecture](INTERNAL_ARCHITECTURE.md) implications
6. **Study** the [Modules](INTERNAL_MODULES.md) you're modifying

---

## Document Maintenance

**Last Updated**: January 14, 2026

**Version**: 1.0.0

These documents should be updated when:
- New features are added
- API changes are made
- Architecture is modified
- Performance characteristics change
- Security practices are updated

---

## FAQ

**Q: Where do I find API documentation?**
A: [INTERNAL_API_REFERENCE.md](INTERNAL_API_REFERENCE.md)

**Q: How do I add a new endpoint?**
A: [INTERNAL_EXTENSION_GUIDE.md#custom-routes](INTERNAL_EXTENSION_GUIDE.md#custom-routes)

**Q: What's the coding standard?**
A: [INTERNAL_DEVELOPMENT_GUIDE.md#naming-conventions](INTERNAL_DEVELOPMENT_GUIDE.md#naming-conventions)

**Q: How does authentication work?**
A: [INTERNAL_ARCHITECTURE.md#middleware-pipeline](INTERNAL_ARCHITECTURE.md#middleware-pipeline)

**Q: How do I query the database?**
A: [INTERNAL_API_REFERENCE.md#query-builder-apis](INTERNAL_API_REFERENCE.md#query-builder-apis)

**Q: How do I add custom business logic?**
A: [INTERNAL_EXTENSION_GUIDE.md#cloud-functions](INTERNAL_EXTENSION_GUIDE.md#cloud-functions)

**Q: How do I test my code?**
A: [INTERNAL_DEVELOPMENT_GUIDE.md#testing-strategies](INTERNAL_DEVELOPMENT_GUIDE.md#testing-strategies)

**Q: Where can I find examples?**
A: [INTERNAL_EXTENSION_GUIDE.md#real-world-examples](INTERNAL_EXTENSION_GUIDE.md#real-world-examples)

---

## Additional Resources

### User-Facing Documentation
- [README.md](README.md) - Getting started guide
- [API_DOCUMENTATION.md](API_DOCUMENTATION.md) - Public API reference

### Configuration
- [config.json](config.json) - Configuration example
- [database.json](database.json) - Database schema example

### Project Files
- [go.mod](go.mod) - Module definition and dependencies
- [example.go](example.go) - Code examples

---

## Quick Start for Developers

```go
// 1. Initialize server
app := yekonga.ServerConfig("config.json", "database.json")

// 2. Add middleware (optional)
app.Use(MyAuthMiddleware)

// 3. Add routes
app.Get("/api/data", func(req *yekonga.Request, res *yekonga.Response) {
    data := app.ModelQuery("Model").Find(nil)
    res.Json(data)
})

// 4. Start server
app.Start(":8080")
```

---

**For questions or clarifications, refer to the specific documentation file or the codebase itself.**
