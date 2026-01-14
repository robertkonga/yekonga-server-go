# YekongaServer API Documentation

Complete API reference for the YekongaServer framework including helper functions, configuration, and server methods.

## Table of Contents

- [Helper Package](#helper-package)
  - [Type Conversion](#type-conversion)
  - [Type Checking](#type-checking)
  - [String Manipulation](#string-manipulation)
  - [Date & Time](#date--time)
  - [Collections & Data Structures](#collections--data-structures)
  - [File Operations](#file-operations)
  - [Network Operations](#network-operations)
  - [Data Validation](#data-validation)
- [Config Package](#config-package)
- [Server Package](#server-package)
  - [Server Initialization](#server-initialization)
  - [Routing](#routing)
  - [Middleware](#middleware)
  - [Database](#database)
  - [Cloud Functions](#cloud-functions)
  - [Static Files](#static-files)
  - [Cron Jobs](#cron-jobs)

---

## Helper Package

The helper package provides utility functions for common operations like type conversion, validation, and data manipulation.

### Type Conversion

#### `ToInt(value interface{}) int`
Converts a value to an integer.
```go
result := helper.ToInt("42")           // 42
result := helper.ToInt(3.14)           // 3 (truncated)
result := helper.ToInt("invalid")      // 0 (default)
```

#### `ToFloat(value interface{}) float64`
Converts a value to a float64.
```go
result := helper.ToFloat("3.14")       // 3.14
result := helper.ToFloat(42)           // 42.0
result := helper.ToFloat("invalid")    // 0.0 (default)
```

#### `ToJson(data interface{}) string`
Converts any data structure to a JSON string with indentation.
```go
data := map[string]interface{}{"name": "John", "age": 30}
json := helper.ToJson(data)
// Returns formatted JSON string
```

#### `ToByte(data interface{}) []byte`
Converts data to a byte slice (JSON marshaled).
```go
data := map[string]interface{}{"name": "John"}
bytes := helper.ToByte(data)
// Returns: []byte(`{"name":"John"}`)
```

#### `ToMap[T any](data interface{}) map[string]T`
Generic function to convert data to a typed map.
```go
result := helper.ToMap[string](jsonData)
result := helper.ToMap[int](jsonData)
result := helper.ToMap[interface{}](jsonData)
```

#### `ToMapList[T any](data interface{}) []map[string]T`
Converts a slice of objects to a slice of typed maps.
```go
records := helper.ToMapList[interface{}](jsonArray)
// Returns: []map[string]interface{}
```

#### `ToList[T any](data interface{}) []T`
Converts an array-like structure to a typed slice.
```go
users := helper.ToList[User](jsonArray)
numbers := helper.ToList[int](jsonArray)
```

#### `ToInterface(data interface{}) (interface{}, error)`
Converts data to interface{} with error handling.
```go
result, err := helper.ToInterface(data)
if err != nil {
    log.Println("Conversion failed:", err)
}
```

#### `ConvertTo[T any](data interface{}) (T, error)`
Generic type conversion with error handling. Useful for converting maps to structs.
```go
type User struct {
    Name string `json:"name"`
    Age  int    `json:"age"`
}

user, err := helper.ConvertTo[User](data)
if err != nil {
    log.Println("Conversion failed:", err)
}
```

#### `ConvertToDataMap(data map[string]interface{}) datatype.DataMap`
Converts a map to a DataMap type.
```go
dataMap := helper.ConvertToDataMap(myMap)
```

### Type Checking

#### `IsMap(data interface{}) bool`
Checks if data is a map type (supports multiple map types).
```go
if helper.IsMap(data) {
    // Handle map
}
```

#### `IsSlice(v interface{}) bool`
Checks if data is a slice.
```go
if helper.IsSlice(data) {
    // Handle slice
}
```

#### `IsList(v interface{}) bool`
Alias for `IsSlice()`. Checks if data is a list/slice.
```go
if helper.IsList(data) {
    // Handle list
}
```

#### `IsArray(v interface{}) bool`
Alias for `IsSlice()`. Checks if data is an array.
```go
if helper.IsArray(data) {
    // Handle array
}
```

#### `IsEmpty(value interface{}) bool`
Checks if a value is empty (nil, empty string, empty slice/map).
```go
if helper.IsEmpty(value) {
    // Value is empty
}
```

#### `IsNotEmpty(value interface{}) bool`
Checks if a value is not empty.
```go
if helper.IsNotEmpty(value) {
    // Value has content
}
```

#### `IsNumeric(value interface{}) bool`
Checks if a value is numeric (int, float, or numeric string).
```go
if helper.IsNumeric("123") {
    // It's numeric
}
```

#### `IsPointer(v interface{}) bool`
Checks if a value is a pointer.
```go
if helper.IsPointer(data) {
    // It's a pointer
}
```

#### `IsSliceOfMapStringInterface(v interface{}) bool`
Checks if data is a slice of map[string]interface{}.
```go
if helper.IsSliceOfMapStringInterface(data) {
    // It's a slice of maps
}
```

### Identifier Checking

#### `IsUsernameIdentifier() bool`
Checks if "username" is a configured user identifier.
```go
if helper.IsUsernameIdentifier() {
    // Username is enabled as identifier
}
```

#### `IsPhoneIdentifier() bool`
Checks if "phone" is a configured user identifier.
```go
if helper.IsPhoneIdentifier() {
    // Phone is enabled as identifier
}
```

#### `IsEmailIdentifier() bool`
Checks if "email" is a configured user identifier.
```go
if helper.IsEmailIdentifier() {
    // Email is enabled as identifier
}
```

#### `IsWhatsappIdentifier() bool`
Checks if "whatsapp" is a configured user identifier.
```go
if helper.IsWhatsappIdentifier() {
    // WhatsApp is enabled as identifier
}
```

### String Manipulation

#### `ToCamelCase(s string) string`
Converts a string to camelCase.
```go
result := helper.ToCamelCase("hello_world")      // helloWorld
result := helper.ToCamelCase("hello-world")      // helloWorld
```

#### `ToUnderscore(text string) string`
Converts a string to snake_case.
```go
result := helper.ToUnderscore("helloWorld")      // hello_world
result := helper.ToUnderscore("HelloWorld")      // hello_world
```

#### `CamelToSnake(s string) string`
Converts camelCase or PascalCase to snake_case.
```go
result := helper.CamelToSnake("helloWorld")      // hello_world
result := helper.CamelToSnake("HelloWorld")      // hello_world
```

#### `ToVariable(s string) string`
Converts a string to a valid variable name (camelCase, lowercase first letter).
```go
result := helper.ToVariable("HelloWorld")        // helloWorld
result := helper.ToVariable("hello_world")       // helloWorld
```

#### `ToTitle(s string) string`
Converts a string to title case with spaces.
```go
result := helper.ToTitle("hello_world")          // Hello World
result := helper.ToTitle("helloWorld")           // Hello World
```

#### `ToSlug(s string) string`
Converts a string to a URL-friendly slug.
```go
result := helper.ToSlug("Hello World")           // hello-world
result := helper.ToSlug("Hello-World")           // hello-world
```

#### `ToString(s interface{}) string`
Converts any value to its string representation.
```go
result := helper.ToString(123)                   // "123"
result := helper.ToString(true)                  // "true"
```

#### `Pluralize(word string) string`
Converts a singular noun to plural form.
```go
result := helper.Pluralize("cat")                // cats
result := helper.Pluralize("box")                // boxes
result := helper.Pluralize("city")               // cities
```

#### `Singularize(word string) string`
Converts a plural noun to singular form.
```go
result := helper.Singularize("cats")             // cat
result := helper.Singularize("boxes")            // box
result := helper.Singularize("cities")           // city
```

#### `FormatPhone(phone interface{}) string`
Formats and validates a phone number.
```go
result := helper.FormatPhone("0654321098")       // 255654321098
result := helper.FormatPhone("+255654321098")    // 255654321098
```

#### `PhoneFormat(phone interface{}) string`
Alias for `FormatPhone()`.
```go
result := helper.PhoneFormat("0654321098")
```

#### `ClearSpecialCharacters(val string) string`
Removes special characters and cleans HTML tags.
```go
result := helper.ClearSpecialCharacters("<p>Hello!</p>")  // Hello
```

#### `GetRandomString(length int, typ string) string`
Generates a random string. Types: "number", "letter", "hex".
```go
result := helper.GetRandomString(10, "number")   // Random 10-digit number
result := helper.GetRandomString(8, "letter")    // Random 8-letter string
result := helper.GetRandomString(16, "hex")      // Random hex string
```

#### `GetRandomInt(length int) string`
Generates a random integer string of specified length.
```go
result := helper.GetRandomInt(6)                 // Random 6-digit number
```

#### `GetHexString(length int) string`
Generates a random hexadecimal string.
```go
result := helper.GetHexString(32)                // Random 32-char hex string
```

#### `StringLength(value string) int`
Gets the character length of a string (counts Unicode characters correctly).
```go
length := helper.StringLength("Hello")           // 5
length := helper.StringLength("你好")             // 2
```

### Date & Time

#### `StringToDatetime(value interface{}) *time.Time`
Parses multiple date/time formats.
```go
t := helper.StringToDatetime("2024-01-14")
t := helper.StringToDatetime("2024-01-14 15:30:45")
t := helper.StringToDatetime("2024-01-14T15:30:45Z")
```

#### `StringToTimeOnly(value interface{}) *time.Time`
Parses time-only strings.
```go
t := helper.StringToTimeOnly("15:30:45")
t := helper.StringToTimeOnly("3:30 PM")
```

#### `GetTimestamp(value interface{}) time.Time`
Converts a value to a UTC timestamp.
```go
ts := helper.GetTimestamp("2024-01-14")
```

#### `ToTimestampString(value interface{}, layout string) time.Time`
Parses a string with a custom time layout.
```go
ts := helper.ToTimestampString("2024", "2006")
```

#### `Yesterday() time.Time`
Returns yesterday at 00:00:00 UTC.
```go
yesterday := helper.Yesterday()
```

#### `YesterdayEnd() time.Time`
Returns yesterday at 23:59:59 UTC.
```go
yesterdayEnd := helper.YesterdayEnd()
```

#### `Today() time.Time`
Returns today at 00:00:00 UTC.
```go
today := helper.Today()
```

#### `TodayEnd() time.Time`
Returns today at 23:59:59 UTC.
```go
todayEnd := helper.TodayEnd()
```

#### `Tomorrow() time.Time`
Returns tomorrow at 00:00:00 UTC.
```go
tomorrow := helper.Tomorrow()
```

#### `TomorrowEnd() time.Time`
Returns tomorrow at 23:59:59 UTC.
```go
tomorrowEnd := helper.TomorrowEnd()
```

#### `DateStart(value interface{}) time.Time`
Returns the start of day (00:00:00) for a given date.
```go
start := helper.DateStart("2024-01-14")
```

#### `DateEnd(value interface{}) time.Time`
Returns the end of day (23:59:59) for a given date.
```go
end := helper.DateEnd("2024-01-14")
```

#### `HourStart(value interface{}) time.Time`
Returns the start of hour (XX:00:00).
```go
start := helper.HourStart("2024-01-14 15:30:45")
```

#### `HourEnd(value interface{}) time.Time`
Returns the end of hour (XX:59:59).
```go
end := helper.HourEnd("2024-01-14 15:30:45")
```

#### `WeekStart(value interface{}) time.Time`
Returns the start of the week (Monday 00:00:00).
```go
start := helper.WeekStart("2024-01-14")
```

#### `WeekEnd(value interface{}) time.Time`
Returns the end of the week (Sunday 23:59:59).
```go
end := helper.WeekEnd("2024-01-14")
```

#### `MonthStart(value interface{}) time.Time`
Returns the start of the month (1st, 00:00:00).
```go
start := helper.MonthStart("2024-01-14")
```

#### `MonthEnd(value interface{}) time.Time`
Returns the end of the month (last day, 23:59:59).
```go
end := helper.MonthEnd("2024-01-14")
```

#### `TrackTime(start *time.Time, name string)`
Tracks elapsed time and prints it.
```go
start := time.Now()
// ... do work ...
helper.TrackTime(&start, "operation")  // Prints: operation took XXms
```

### Collections & Data Structures

#### `Contains(slice []string, item string) bool`
Checks if a slice contains an item.
```go
if helper.Contains([]string{"a", "b", "c"}, "b") {
    // Item found
}
```

#### `Reverse[T interface{}](slice []T)`
Reverses a slice in place.
```go
numbers := []int{1, 2, 3}
helper.Reverse(numbers)  // [3, 2, 1]
```

#### `SortMap[T interface{}](options map[string]T) map[string]T`
Sorts a map by values.
```go
sorted := helper.SortMap(myMap)
```

#### `SortedKeys[T interface{}](options map[string]T) []string`
Returns sorted keys from a map.
```go
keys := helper.SortedKeys(myMap)
```

#### `UUID() string`
Generates a UUID v1.
```go
id := helper.UUID()  // "f47ac10b-58cc-4372-a567-0e02b2c3d479"
```

#### `ObjectID(id interface{}) bson.ObjectID`
Converts a value to a MongoDB ObjectID.
```go
objID := helper.ObjectID("507f1f77bcf86cd799439011")
objID := helper.ObjectID(123)
```

#### `ObjectIDtoString(id bson.ObjectID) string`
Converts a MongoDB ObjectID to string.
```go
str := helper.ObjectIDtoString(objID)
```

#### `GetMapValue(data interface{}, key string) interface{}`
Gets a value from a map using dot notation.
```go
value := helper.GetMapValue(data, "user.profile.name")
```

#### `GetMapString(data interface{}, key string) string`
Gets a string value from a map.
```go
name := helper.GetMapString(data, "user.name")
```

#### `GetMapInt(data interface{}, key string) int`
Gets an int value from a map.
```go
age := helper.GetMapInt(data, "user.age")
```

#### `GetMapFloat(data interface{}, key string) float64`
Gets a float value from a map.
```go
price := helper.GetMapFloat(data, "product.price")
```

#### `GetMapBoolean(data interface{}, key string) bool`
Gets a boolean value from a map.
```go
active := helper.GetMapBoolean(data, "user.active")
```

#### `GetMapDate(data interface{}, key string) time.Time`
Gets a date value from a map.
```go
createdAt := helper.GetMapDate(data, "user.createdAt")
```

#### `GetMap(data interface{}, key string) map[string]interface{}`
Gets a map value from nested data.
```go
userMap := helper.GetMap(data, "user.profile")
```

#### `GetMapArray(data interface{}, source string, keys map[string]string) []interface{}`
Extracts specific keys from array elements.
```go
users := helper.GetMapArray(data, "users", map[string]string{
    "name": "name",
    "age": "age",
})
```

#### `GetList(data interface{}, key string) []interface{}`
Gets a list of values from nested array.
```go
names := helper.GetList(data, "user.names")
```

#### `GetFirst(data interface{}) interface{}`
Gets the first element from a map or slice.
```go
first := helper.GetFirst(data)
```

#### `GetType(data interface{}) string`
Gets the type of data as a string.
```go
t := helper.GetType(data)  // "map[string]interface{}"
```

### File Operations

#### `FileExists(filename string) bool`
Checks if a file exists.
```go
if helper.FileExists("config.json") {
    // File exists
}
```

#### `CreateFile(data interface{}, filename string) error`
Creates a file with data.
```go
err := helper.CreateFile(myData, "output.json")
```

#### `SaveToFile(data interface{}, filename string) error`
Saves data to a file.
```go
err := helper.SaveToFile(myData, "data.json")
```

#### `CreateDirectory(dir string) error`
Creates a directory (and parent directories if needed).
```go
err := helper.CreateDirectory("/path/to/directory")
```

#### `CreateFolder(dir string) error`
Alias for `CreateDirectory()`.
```go
err := helper.CreateFolder("/path/to/folder")
```

#### `LoadFile(filename string) string`
Reads a file and returns its content as a string.
```go
content := helper.LoadFile("config.json")
```

#### `LoadJSONFile(filename string) (map[string]interface{}, error)`
Reads and parses a JSON file.
```go
data, err := helper.LoadJSONFile("config.json")
```

#### `DownloadFile(url, destPath string, progress func(downloaded, total int64)) error`
Downloads a file from a URL with optional progress callback.
```go
err := helper.DownloadFile(
    "https://example.com/file.zip",
    "./downloads/file.zip",
    func(downloaded, total int64) {
        fmt.Printf("Downloaded: %d/%d bytes\n", downloaded, total)
    },
)
```

#### `HomeDirectory(name string) string`
Gets the application home directory.
```go
appDir := helper.HomeDirectory("myapp")  // ~/.yekonga-server/myapp
```

#### `GetDirectoryPath() string`
Gets the path of the current executable.
```go
exePath := helper.GetDirectoryPath()
```

#### `GetPublicPath() (string, error)`
Gets the public/static files directory path.
```go
publicPath, err := helper.GetPublicPath()
```

#### `GetPath(relativePath string) string`
Converts a relative path to an absolute path.
```go
absPath := helper.GetPath("./config.json")
```

### Network Operations

#### `GetLocalIP() (string, error)`
Gets the first non-loopback local IPv4 address.
```go
ip, err := helper.GetLocalIP()  // "192.168.1.100"
```

#### `GetLocalIPS() ([]string, error)`
Gets all local IPv4 addresses.
```go
ips, err := helper.GetLocalIPS()  // ["192.168.1.100", "10.0.0.5"]
```

#### `GetPublicIP() (string, error)`
Fetches the external/public IP address.
```go
ip, err := helper.GetPublicIP()
```

#### `GetNetworkIP() (string, error)`
Gets the network address from the local IP.
```go
networkIP, err := helper.GetNetworkIP()  // "192.168.1.0"
```

#### `GetRequest(url string, headers map[string]string) (int, string, error)`
Makes an HTTP GET request.
```go
status, body, err := helper.GetRequest(
    "https://api.example.com/users",
    map[string]string{"Authorization": "Bearer token"},
)
```

#### `PostRequest(url string, headers map[string]string, body interface{}) (int, string, error)`
Makes an HTTP POST request.
```go
status, body, err := helper.PostRequest(
    "https://api.example.com/users",
    map[string]string{"Content-Type": "application/json"},
    map[string]interface{}{"name": "John"},
)
```

### Data Validation

#### `ValidateEmail(email interface{}) bool`
Validates an email address format.
```go
if helper.ValidateEmail("user@example.com") {
    // Valid email
}
```

#### `IsEmail(email interface{}) bool`
Alias for `ValidateEmail()`.
```go
if helper.IsEmail("user@example.com") {
    // Valid email
}
```

#### `IsPhone(value interface{}) bool`
Validates a phone number format.
```go
if helper.IsPhone("0654321098") {
    // Valid phone
}
```

### Data Extraction

#### `GetSortedUniqueKeys(records []datatype.DataMap) []string`
Extracts and sorts unique keys from records.
```go
keys := helper.GetSortedUniqueKeys(records)
```

#### `ConvertJSONArrayToDataArray(jsonData interface{}, headingColumns []string) ([][]string, error)`
Converts JSON array to 2D string array.
```go
data, err := helper.ConvertJSONArrayToDataArray(jsonData, []string{"name", "email"})
```

#### `ConvertJSONArrayToCSV(jsonData interface{}, headingColumns []string, filename string) (string, error)`
Converts JSON array to CSV file.
```go
path, err := helper.ConvertJSONArrayToCSV(
    jsonData,
    []string{"name", "email"},
    "export.csv",
)
```

#### `ConvertJSONArrayToExcel(jsonData interface{}, headingColumns []string, filename string) (string, error)`
Converts JSON array to Excel file.
```go
path, err := helper.ConvertJSONArrayToExcel(
    jsonData,
    []string{"name", "email", "age"},
    "export.xlsx",
)
```

#### `TextTemplate(templateString string, data map[string]interface{}, regexPattern *regexp.Regexp) string`
Replaces placeholders in a template string.
```go
result := helper.TextTemplate(
    "Hello {{name}}, you are {{age}} years old",
    map[string]interface{}{"name": "John", "age": 30},
    nil,
)
// Result: "Hello John, you are 30 years old"
```

#### `GetTextContent(template string, data map[string]interface{}) string`
Reads and processes a template file.
```go
content := helper.GetTextContent("templates/email.txt", templateData)
```

#### `GetWhatsappContent(template string, data map[string]interface{}) interface{}`
Reads a template file and parses it as JSON for WhatsApp.
```go
content := helper.GetWhatsappContent("templates/whatsapp.json", data)
```

#### `GetEmailContent(layout, template string, data map[string]interface{}) string`
Reads and processes an email template with layout.
```go
html := helper.GetEmailContent(
    "layouts/email.html",
    "templates/welcome.html",
    emailData,
)
```

#### `GetCrossPlatformIdleTime() time.Duration`
Gets the system idle time (cross-platform).
```go
idleTime := helper.GetCrossPlatformIdleTime()
```

---

## Config Package

The config package handles application configuration.

### Functions

#### `NewYekongaConfig(file string) *YekongaConfig`
Loads and initializes configuration from a JSON file.
```go
config := config.NewYekongaConfig("./config.json")
config.AppName      // Application name
config.Environment  // development, staging, production
config.Debug        // Debug mode enabled/disabled
```

#### `LoadJSONFile(filename string) (map[string]interface{}, error)`
Reads and parses a JSON file.
```go
data, err := config.LoadJSONFile("config.json")
```

#### `FileExists(filename string) bool`
Checks if a file exists.
```go
if config.FileExists("config.json") {
    // File exists
}
```

#### `GetPath(relativePath string) string`
Converts a relative path to absolute path from executable.
```go
absPath := config.GetPath("./config.json")
```

#### `ToByte(data interface{}) []byte`
Marshals data to JSON bytes.
```go
bytes := config.ToByte(data)
```

### Configuration Structure

```go
type YekongaConfig struct {
    AppName                 string
    Version                 string
    Environment             string  // development, staging, production
    Debug                   bool
    Protocol                string  // http or https
    Address                 string
    Domain                  string
    DomainAlias             []string
    AppKey                  string
    MasterKey               string
    EnableAppKey            bool
    TokenKey                string
    TokenExpireTime         int
    SecureOnly              bool
    Cors                    bool
    
    Database struct {
        Kind             string  // mongodb, mysql, sql, local
        Host             string
        Port             string
        DatabaseName     string
        Username         interface{}
        Password         interface{}
    }
    
    Ports struct {
        Secure    bool
        Server    int
        SSLServer int
    }
    
    Authentication struct {
        SaltRound   int
        Algorithm   string
        TokenSecret string
    }
    
    // ... more fields
}
```

---

## Server Package

The server package provides the core framework functionality.

### Server Initialization

#### `YekongaServer(configFile string, databaseFile string) *YekongaData`
Initializes a YekongaServer instance.
```go
server := Yekonga.YekongaServer("./config.json", "./database.json")
server.Start(":8080")
```

#### `YekongaServerAuto(configFile string, databaseFile string)`
Auto-initializes the server (sets the global Server variable).
```go
Yekonga.YekongaServerAuto("./config.json", "./database.json")
// Now use: Yekonga.Server.Get(...)
```

### Routing

#### `Get(path string, handler Handler)`
Registers a GET route.
```go
server.Get("/users", func(req *Yekonga.Request, res *Yekonga.Response) {
    users := server.ModelQuery("User").Find(nil)
    res.Json(users)
})
```

#### `Post(path string, handler Handler)`
Registers a POST route.
```go
server.Post("/users", func(req *Yekonga.Request, res *Yekonga.Response) {
    data := req.Body()
    user, err := server.ModelQuery("User").Create(data)
    res.Json(user)
})
```

#### `Put(path string, handler Handler)`
Registers a PUT route.
```go
server.Put("/users/:id", func(req *Yekonga.Request, res *Yekonga.Response) {
    id := req.Param("id")
    data := req.Body()
    user, err := server.ModelQuery("User").Update(id, data)
    res.Json(user)
})
```

#### `Patch(path string, handler Handler)`
Registers a PATCH route.
```go
server.Patch("/users/:id", func(req *Yekonga.Request, res *Yekonga.Response) {
    // Handle PATCH
})
```

#### `Delete(path string, handler Handler)`
Registers a DELETE route.
```go
server.Delete("/users/:id", func(req *Yekonga.Request, res *Yekonga.Response) {
    id := req.Param("id")
    server.ModelQuery("User").Delete(id)
    res.Json(map[string]string{"status": "deleted"})
})
```

#### `Options(path string, handler Handler)`
Registers an OPTIONS route.
```go
server.Options("/users", func(req *Yekonga.Request, res *Yekonga.Response) {
    res.Header("Allow", "GET, POST, PUT, DELETE")
})
```

#### `All(path string, handler Handler)`
Registers a route for all HTTP methods.
```go
server.All("/status", func(req *Yekonga.Request, res *Yekonga.Response) {
    res.Json(map[string]string{"status": "ok"})
})
```

### Middleware

#### `Middleware(middleware Middleware, middlewareType MiddlewareType)`
Registers middleware.
```go
server.Middleware(func(req *Yekonga.Request, res *Yekonga.Response) error {
    // Your middleware logic
    return nil
}, Yekonga.MiddlewareTypeRequest)
```

#### Built-in Middleware

- `TokenMiddleware` - JWT token validation
- `UserInfoMiddleware` - User info extraction
- `ApplicationKeyMiddleware` - App key validation

### Database

#### `Model(name string) *DataModel`
Gets a data model by name.
```go
userModel := server.Model("User")
```

#### `ModelQuery(name string) *DataModelQuery`
Gets a query builder for a model.
```go
users := server.ModelQuery("User")
    .Where("age", ">", 18)
    .OrderBy("createdAt", "desc")
    .Take(10)
    .Find(nil)
```

### Query Builder Methods

```go
// Filter
.Where(field string, operator string, value interface{})
.WhereIn(field string, values []interface{})
.WhereNotIn(field string, values []interface{})

// Sorting
.OrderBy(field string, direction string)  // asc or desc

// Pagination
.Skip(count int)
.Take(count int)
.Page(number int)

// Selection
.Select(fields ...string)

// Execution
.Find(query interface{}) []interface{}
.FindOne(query interface{}) interface{}
.FindById(id interface{}) interface{}

// Aggregation
.Count() int64
.Sum(field string) float64
.Average(field string) float64
.Min(field string) interface{}
.Max(field string) interface{}
.GroupBy(field string)

// CRUD
.Create(data interface{}) interface{}
.Update(id interface{}, data interface{}) interface{}
.Delete(id interface{}) interface{}
```

### Static Files

#### `Static(config StaticConfig) error`
Configures static file serving.
```go
server.Static(Yekonga.StaticConfig{
    Directory:   "./public",
    PathPrefix:  "/public",
    IndexFile:   "index.html",
    Extensions:  []string{".html", ".css", ".js"},
    CacheMaxAge: 3600,
})
```

### Cron Jobs

#### `RegisterCronjob(name string, frequency time.Duration, callback func(app *YekongaData, time time.Time))`
Registers a cron job that runs at a specified interval.
```go
server.RegisterCronjob("cleanup", 24*time.Hour, func(app *Yekonga.YekongaData, t time.Time) {
    // Runs every 24 hours
    log.Println("Running cleanup job at", t)
})
```

#### `RegisterCronjobAt(name string, frequency JobFrequency, time time.Time, callback func(app *YekongaData, time time.Time))`
Registers a cron job at a specific time.
```go
server.RegisterCronjobAt(
    "daily-report",
    Yekonga.JobFrequencyDaily,
    time.Now().Add(24*time.Hour),
    func(app *Yekonga.YekongaData, t time.Time) {
        // Daily report logic
    },
)
```

### Cloud Functions

#### `Define(name string, function CloudFunction)`
Defines a cloud function.
```go
server.Define("sendEmail", func(data interface{}, rc *Yekonga.RequestContext) (interface{}, error) {
    // Email sending logic
    return map[string]string{"status": "sent"}, nil
})
```

#### `AfterFind(model string, query interface{}, data interface{}, callback TriggerCloudFunction)`
Trigger after fetching records.
```go
server.AfterFind("User", nil, nil, func(rc *Yekonga.RequestContext, qc *Yekonga.QueryContext) (interface{}, error) {
    // Process fetched users
    return nil, nil
})
```

#### `BeforeCreate(model string, callback TriggerCloudFunction)`
Trigger before creating a record.
```go
server.BeforeCreate("User", func(rc *Yekonga.RequestContext, qc *Yekonga.QueryContext) (interface{}, error) {
    // Validate or modify data before insert
    return nil, nil
})
```

### Request & Response

#### `Request` Object Methods

```go
req.Method()           // HTTP method (GET, POST, etc.)
req.URL()              // Request URL
req.Header(key string) // Get header value
req.Body()             // Get request body as interface{}
req.Param(name string) // Get URL parameter
req.Query(name string) // Get query parameter
req.Auth()             // Get authenticated user
req.Context()          // Get request context
```

#### `Response` Object Methods

```go
res.Status(code int)                    // Set HTTP status code
res.Header(key string, value string)    // Set response header
res.Json(data interface{})              // Send JSON response
res.Text(data string)                   // Send text response
res.Byte(data []byte)                   // Send bytes response
res.File(filepath string)               // Send file
res.Redirect(url string)                // Redirect to URL
res.Error(message string, code int)     // Send error response
```

### Server Lifecycle

#### `Start(address interface{})`
Starts the server.
```go
server.Start(":8080")
// or
server.Start("0.0.0.0:8080")
```

#### `Stop()`
Stops the server.
```go
server.Stop()
```

#### `Listen()`
Alias for `Start()` with default configuration.
```go
server.Listen()
```

### Helper Methods

#### `HomeDirectory() string`
Gets the application's home directory.
```go
dir := server.HomeDirectory()
```

#### `ServeHTTP(w http.ResponseWriter, r *http.Request)`
Standard Go HTTP handler interface.
```go
// Used internally, but available for manual routing
```

---

## Complete Example

```go
package main

import (
    "log"
    "time"
    "github.com/robertkonga/yekonga-server-go/yekonga"
    "github.com/robertkonga/yekonga-server-go/helper"
)

func main() {
    // Initialize server
    app := yekonga.ServerConfig("./config.json", "./database.json")
    
    // Register middleware
    app.Use(func(req *server.Request, res *server.Response) error {
        log.Println("Request:", req.Method(), req.URL())
        return nil
    })
    
    // Setup static files
    app.Static(server.StaticConfig{
        Directory:  "./public",
        PathPrefix: "/public",
        CacheMaxAge: 3600,
    })
    
    // Define routes
    app.Get("/", func(req *server.Request, res *server.Response) {
        res.File("index.html")
    })
    
    app.Get("/users", func(req *server.Request, res *server.Response) {
        users := app.ModelQuery("User").Take(10).Find(nil)
        res.Json(users)
    })
    
    app.Post("/users", func(req *server.Request, res *server.Response) {
        data := req.Body()
        user, err := app.ModelQuery("User").Create(data)
        if err != nil {
            res.Error("Failed to create user", 400)
            return
        }
        res.Json(user)
    })
    
    app.Get("/users/:id", func(req *server.Request, res *server.Response) {
        id := req.Param("id")
        user := app.ModelQuery("User").FindById(id)
        res.Json(user)
    })
    
    // Define cloud functions
    app.Define("sendWelcomeEmail", func(data interface{}, rc *server.RequestContext) (interface{}, error) {
        // Send email logic
        return map[string]string{"status": "sent"}, nil
    })
    
    // Register cron job
    app.RegisterCronjob("daily-cleanup", 24*time.Hour, func(app *server.YekongaData, t time.Time) {
        log.Println("Running daily cleanup at", t)
    })
    
    // Start server
    app.Start(":8080")
}
```

---

## Best Practices

1. **Error Handling**: Always check errors from database operations
2. **Type Safety**: Use generics like `ConvertTo[T]()` for type safety
3. **Middleware Order**: Register middleware before routes
4. **Connection Management**: Database connections are managed automatically
5. **Configuration**: Load config at startup and use it throughout the app
6. **Logging**: Use helper functions for consistent logging
7. **Date Handling**: Use helper functions for timezone-safe date operations
8. **File Operations**: Always create directories before saving files

---

## Version History

- **v1.0.0** - Initial release
- Go 1.24.0+
- Compatible with: MongoDB, MySQL, SQL databases

