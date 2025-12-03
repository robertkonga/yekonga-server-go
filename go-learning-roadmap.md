Below is a formatted Markdown file containing the Go Language Learning Roadmap with detailed explanations and examples for each topic. You can copy this content into a file named `go-learning-roadmap.md` and save it for download or local use. The Markdown is structured with proper headings, code blocks, and concise explanations to cover all scenarios as requested.

---

# Go Language Learning Roadmap

This document provides a comprehensive roadmap for learning the Go programming language, covering basic, intermediate, and advanced topics with detailed examples and scenarios.

---

## ✅ Basic Topics (Foundations)

### 1. Setup & Tooling
**Overview**: Set up the Go environment, including installation, workspace configuration, and essential commands.

- **Installing Go**: Download from [golang.org](https://golang.org/doc/install) and verify with `go version`.
- **GOROOT and GOPATH**: `GOROOT` is the Go installation path; `GOPATH` is your workspace (defaults to `~/go`).
- **Commands**:
  - `go run <file.go>`: Compiles and runs a Go file.
  - `go build`: Creates an executable.
  - `go mod init <module-name>`: Initializes a module.
  - `go get <package>`: Fetches third-party packages.

**Example**: Setting up a project.
```go
// main.go
package main

import "fmt"

func main() {
    fmt.Println("Hello, Go!")
}
```
**Steps**:
1. Install Go and verify: `go version`.
2. Create directory: `mkdir myproject && cd myproject`.
3. Initialize module: `go mod init myproject`.
4. Run: `go run main.go` (prints "Hello, Go!").
5. Build: `go build` (creates `myproject` executable).
6. Add package: `go get github.com/google/uuid`.

**Example with UUID**:
```go
package main

import (
    "fmt"
    "github.com/google/uuid"
)

func main() {
    id := uuid.New()
    fmt.Println("Generated UUID:", id)
}
```

### 2. Basic Syntax
**Overview**: Covers variables, constants, and basic data types (`int`, `float64`, `string`, `bool`).

**Example**: Variables and constants.
```go
package main

import "fmt"

func main() {
    // Variables
    var name string = "Alice"
    age := 30 // Short declaration
    var height float64 = 5.9
    isStudent := false

    // Constants
    const pi = 3.14159

    fmt.Printf("Name: %s, Age: %d, Height: %.1f, IsStudent: %t, Pi: %.5f\n",
        name, age, height, isStudent, pi)
}
```
**Scenarios**:
- Explicit type declaration (`var name string`).
- Type inference (`age := 30`).
- Constants for immutable values.

### 3. Control Structures
**Overview**: Go supports `if`, `else`, `switch`, and `for` loops (no `while`).

**Example**: Conditionals and loops.
```go
package main

import "fmt"

func main() {
    // If-else
    score := 85
    if score >= 90 {
        fmt.Println("Grade: A")
    } else if score >= 80 {
        fmt.Println("Grade: B")
    } else {
        fmt.Println("Grade: C")
    }

    // Switch
    day := "Monday"
    switch day {
    case "Monday", "Tuesday":
        fmt.Println("Workday")
    case "Saturday", "Sunday":
        fmt.Println("Weekend")
    default:
        fmt.Println("Midweek")
    }

    // For loop
    for i := 1; i <= 5; i++ {
        fmt.Printf("Number: %d\n", i)
    }

    // Range loop
    fruits := []string{"apple", "banana", "orange"}
    for index, fruit := range fruits {
        fmt.Printf("Index: %d, Fruit: %s\n", index, fruit)
    }
}
```
**Scenarios**:
- Multi-condition `if` statements.
- `switch` with multiple cases.
- Classic and `range`-based loops.

### 4. Functions
**Overview**: Functions support multiple returns, variadic parameters, `defer`, `panic`, and `recover`.

**Example**: Function features.
```go
package main

import "fmt"

func add(a, b int) (sum int, err error) {
    if a < 0 || b < 0 {
        return 0, fmt.Errorf("negative numbers not allowed: %d, %d", a, b)
    }
    return a + b, nil
}

func printNames(names ...string) {
    for _, name := range names {
        fmt.Println("Name:", name)
    }
}

func divide(a, b float64) (float64, error) {
    if b == 0 {
        panic("division by zero")
    }
    return a / b, nil
}

func main() {
    // Multiple returns
    result, err := add(5, 3)
    if err != nil {
        fmt.Println("Error:", err)
    } else {
        fmt.Println("Sum:", result)
    }

    // Variadic function
    printNames("Alice", "Bob", "Charlie")

    // Defer
    defer fmt.Println("This runs at the end")

    // Panic and recover
    defer func() {
        if r := recover(); r != nil {
            fmt.Println("Recovered from:", r)
        }
    }()
    result, err = divide(10, 0)
    if err != nil {
        fmt.Println("Error:", err)
    } else {
        fmt.Println("Result:", result)
    }
}
```
**Scenarios**:
- Error handling with multiple returns.
- Variadic functions for flexible input.
- `defer` for cleanup, `panic`/`recover` for error recovery.

### 5. Arrays, Slices, and Maps
**Overview**: Arrays are fixed-size, slices are dynamic, and maps are key-value stores.

**Example**: Working with collections.
```go
package main

import "fmt"

func main() {
    // Array
    var numbers [3]int = [3]int{1, 2, 3}
    fmt.Println("Array:", numbers)

    // Slice
    slice := []int{4, 5, 6}
    slice = append(slice, 7)
    fmt.Println("Slice:", slice)
    fmt.Println("Slice[1:3]:", slice[1:3])

    // Map
    scores := map[string]int{
        "Alice": 90,
        "Bob":   85,
    }
    scores["Charlie"] = 95
    delete(scores, "Bob")
    fmt.Println("Map:", scores)
    if score, exists := scores["Alice"]; exists {
        fmt.Println("Alice's score:", score)
    }
}
```
**Scenarios**:
- Array vs. slice differences.
- Slice operations (append, slicing).
- Map operations (add, delete, check existence).

### 6. Structs and Interfaces
**Overview**: Structs define custom types, methods add behavior, and interfaces enable polymorphism.

**Example**: Structs and interfaces.
```go
package main

import "fmt"

type Person struct {
    Name string
    Age  int
}

func (p Person) Greet() string {
    return fmt.Sprintf("Hello, I'm %s!", p.Name)
}

type Greeter interface {
    Greet() string
}

func sayHello(g Greeter) {
    fmt.Println(g.Greet())
}

func main() {
    // Struct
    p := Person{Name: "Alice", Age: 30}
    fmt.Println("Person:", p)

    // Method
    fmt.Println(p.Greet())

    // Interface
    var g Greeter = p
    sayHello(g)

    // Type assertion
    if person, ok := g.(Person); ok {
        fmt.Println("Type assertion:", person.Name)
    }
}
```
**Scenarios**:
- Struct creation and method attachment.
- Interface-based polymorphism.
- Type assertions for concrete types.

### 7. Pointers
**Overview**: Pointers enable pass-by-reference for modifying data.

**Example**: Using pointers.
```go
package main

import "fmt"

func updateValue(x *int) {
    *x = *x + 10
}

func main() {
    x := 5
    fmt.Println("Before:", x)
    updateValue(&x)
    fmt.Println("After:", x)
    var p *int = &x
    fmt.Println("Pointer value:", *p)
}
```
**Scenarios**:
- Modifying values via pointers.
- Dereferencing pointers.

---

## 🔁 Intermediate Topics

### 8. Packages and Modules
**Overview**: Packages organize code; modules manage dependencies.

**Example**: Custom package.
**Directory Structure**:
```
myproject/
├── go.mod
├── main.go
└── utils/
    └── math.go
```
**`utils/math.go`**:
```go
package utils

func Add(a, b int) int {
    return a + b
}
```
**`main.go`**:
```go
package main

import (
    "fmt"
    "myproject/utils"
)

func main() {
    sum := utils.Add(3, 4)
    fmt.Println("Sum:", sum)
}
```
**Steps**:
1. Initialize: `go mod init myproject`.
2. Run: `go run .`.

**Scenarios**:
- Local package imports.
- Third-party packages via `go get`.

### 9. Error Handling
**Overview**: Go uses explicit error returns instead of exceptions.

**Example**: Custom errors.
```go
package main

import (
    "errors"
    "fmt"
)

type ValidationError struct {
    Field string
    Issue string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("%s: %s", e.Field, e.Issue)
}

func validateAge(age int) error {
    if age < 0 {
        return &ValidationError{Field: "age", Issue: "cannot be negative"}
    }
    return nil
}

func main() {
    err := validateAge(-5)
    if err != nil {
        fmt.Println("Error:", err)
        if vErr, ok := err.(*ValidationError); ok {
            fmt.Printf("Validation error - Field: %s, Issue: %s\n", vErr.Field, vErr.Issue)
        }
    } else {
        fmt.Println("Age is valid")
    }

    fmt.Println("Simple error:", errors.New("simple error"))
    fmt.Println("Formatted error:", fmt.Errorf("value %d is invalid", 42))
}
```
**Scenarios**:
- Custom error types.
- `errors.New` and `fmt.Errorf`.
- Type assertions for errors.

### 10. Concurrency
**Overview**: Goroutines and channels enable concurrent programming.

**Example**: Goroutines, channels, and mutex.
```go
package main

import (
    "fmt"
    "sync"
    "time"
)

func worker(id int, ch chan string) {
    ch <- fmt.Sprintf("Worker %d done", id)
}

func main() {
    // Goroutine
    go func() {
        fmt.Println("Goroutine running")
        time.Sleep(time.Second)
        fmt.Println("Goroutine done")
    }()

    // Channels
    ch := make(chan string, 2)
    for i := 1; i <= 2; i++ {
        go worker(i, ch)
    }
    for i := 0; i < 2; i++ {
        select {
        case msg := <-ch:
            fmt.Println(msg)
        }
    }

    // Mutex
    var counter int
    var mu sync.Mutex
    var wg sync.WaitGroup
    wg.Add(2)
    for i := 1; i <= 2; i++ {
        go func(id int) {
            defer wg.Done()
            mu.Lock()
            counter++
            fmt.Printf("Worker %d incremented counter to %d\n", id, counter)
            mu.Unlock()
        }(i)
    }
    wg.Wait()
}
```
**Scenarios**:
- Goroutines for parallel tasks.
- Buffered/unbuffered channels.
- `sync.Mutex` for thread safety.

### 11. File I/O
**Overview**: Use `os` and `io` packages for file operations.

**Example**: Reading and writing files.
```go
package main

import (
    "fmt"
    "io"
    "os"
)

func main() {
    // Write
    err := os.WriteFile("output.txt", []byte("Hello, Go!\n"), 0644)
    if err != nil {
        fmt.Println("Error writing:", err)
        return
    }

    // Read
    data, err := os.ReadFile("output.txt")
    if err != nil {
        fmt.Println("Error reading:", err)
        return
    }
    fmt.Println("File contents:", string(data))

    // Append
    f, err := os.OpenFile("output.txt", os.O_APPEND|os.O_WRONLY, 0644)
    if err != nil {
        fmt.Println("Error opening:", err)
        return
    }
    defer f.Close()
    if _, err := io.WriteString(f, "Appended text\n"); err != nil {
        fmt.Println("Error appending:", err)
    }
}
```
**Scenarios**:
- Reading/writing entire files.
- Appending to files.

### 12. Testing
**Overview**: Go supports unit tests and benchmarks with `go test`.

**Example**: Tests and benchmarks.
**`math.go`**:
```go
package math

func Add(a, b int) int {
    return a + b
}
```
**`math_test.go`**:
```go
package math

import "testing"

func TestAdd(t *testing.T) {
    tests := []struct {
        a, b, want int
    }{
        {1, 2, 3},
        {0, 0, 0},
        {-1, 1, 0},
    }
    for _, tt := range tests {
        if got := Add(tt.a, tt.b); got != tt.want {
            t.Errorf("Add(%d, %d) = %d; want %d", tt.a, tt.b, got, tt.want)
        }
    }
}

func BenchmarkAdd(b *testing.B) {
    for i := 0; i < b.N; i++ {
        Add(42, 58)
    }
}
```
**Steps**:
1. Run tests: `go test`.
2. Check coverage: `go test -cover`.
3. Run benchmark: `go test -bench=.`.

**Scenarios**:
- Table-driven tests.
- Performance benchmarking.

### 13. JSON and Encoding
**Overview**: Use `encoding/json` for JSON marshaling/unmarshaling.

**Example**: JSON handling.
```go
package main

import (
    "encoding/json"
    "fmt"
)

type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

func main() {
    // Marshal
    user := User{ID: 1, Name: "Alice"}
    data, err := json.MarshalIndent(user, "", "  ")
    if err != nil {
        fmt.Println("Error marshaling:", err)
        return
    }
    fmt.Println("JSON:", string(data))

    // Unmarshal
    jsonStr := `{"id": 2, "name": "Bob"}`
    var u User
    err = json.Unmarshal([]byte(jsonStr), &u)
    if err != nil {
        fmt.Println("Error unmarshaling:", err)
        return
    }
    fmt.Printf("Unmarshaled: %+v\n", u)
}
```
**Scenarios**:
- JSON serialization with struct tags.
- Deserialization to structs.

---

## 🚀 Advanced Topics

### 14. Context Package
**Overview**: Manages timeouts, cancellations, and request-scoped data.

**Example**: Context with timeout.
```go
package main

import (
    "context"
    "fmt"
    "time"
)

func longRunningTask(ctx context.Context) error {
    select {
    case <-time.After(2 * time.Second):
        fmt.Println("Task completed")
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
    defer cancel()
    err := longRunningTask(ctx)
    if err != nil {
        fmt.Println("Error:", err)
    }
}
```
**Scenarios**:
- Timeout management.
- Cancellation propagation.

### 15. Reflection
**Overview**: Inspect and manipulate types at runtime with `reflect`.

**Example**: Reflecting on a struct.
```go
package main

import (
    "fmt"
    "reflect"
)

func main() {
    val := struct {
        Name string
        Age  int
    }{Name: "Alice", Age: 30}

    v := reflect.ValueOf(val)
    t := reflect.TypeOf(val)

    for i := 0; i < t.NumField(); i++ {
        field := t.Field(i)
        value := v.Field(i).Interface()
        fmt.Printf("Field: %s, Value: %v\n", field.Name, value)
    }
}
```
**Scenarios**:
- Inspecting struct fields.
- Use sparingly due to complexity.

### 16. Build Tags and Conditional Compilation
**Overview**: Compile platform-specific code using build tags.

**Example**: Platform-specific code.
**`main_linux.go`**:
```go
// +build linux

package main

import "fmt"

func main() {
    fmt.Println("Running on Linux")
}
```
**`main_windows.go`**:
```go
// +build windows

package main

import "fmt"

func main() {
    fmt.Println("Running on Windows")
}
```
**Steps**:
- Build: `go build` (compiles for current OS).

**Scenarios**:
- Platform-specific implementations.
- Conditional compilation for environments.

### 17. Generics (Go 1.18+)
**Overview**: Type parameters enable reusable, type-safe code.

**Example**: Generic function and struct.
```go
package main

import "fmt"

type Number interface {
    int | float64
}

func Sum[T Number](a, b T) T {
    return a + b
}

type Pair[T any] struct {
    First, Second T
}

func main() {
    fmt.Println("Int sum:", Sum(3, 4))
    fmt.Println("Float sum:", Sum(3.14, 2.86))
    p := Pair[string]{First: "Hello", Second: "World"}
    fmt.Printf("Pair: %+v\n", p)
}
```
**Scenarios**:
- Generic functions with constraints.
- Generic structs for flexibility.

### 18. Embedding
**Overview**: Struct and interface embedding for composition.

**Example**: Embedding example.
```go
package main

import "fmt"

type Person struct {
    Name string
}

func (p Person) Greet() string {
    return "Hello, " + p.Name
}

type Employee struct {
    Person
    ID int
}

type Greeter interface {
    Greet() string
}

type SuperGreeter interface {
    Greeter
    Salute()
}

type Manager struct {
    Employee
}

func (m Manager) Salute() {
    fmt.Println("Saluting", m.Name)
}

func main() {
    emp := Employee{Person: Person{Name: "Alice"}, ID: 1}
    fmt.Println(emp.Greet())
    var sg SuperGreeter = Manager{Employee: emp}
    sg.Salute()
    fmt.Println(sg.Greet())
}
```
**Scenarios**:
- Struct embedding for composition.
- Interface embedding for extensibility.

### 19. Memory Management
**Overview**: Go uses garbage collection and escape analysis.

**Example**: Escape analysis (run `go build -gcflags="-m"`).
```go
package main

import "fmt"

func createLargeSlice() []int {
    s := make([]int, 1000)
    return s // Escapes to heap
}

func main() {
    s := createLargeSlice()
    fmt.Println("Slice length:", len(s))
}
```
**Scenarios**:
- Heap vs. stack allocation.
- Garbage collection basics.

### 20. Web Development with Go
**Overview**: Build HTTP servers with `net/http` and routers like `gorilla/mux`.

**Example**: HTTP server with middleware.
```go
package main

import (
    "fmt"
    "log"
    "net/http"
    "github.com/gorilla/mux"
)

func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Println("Request:", r.URL.Path)
        next.ServeHTTP(w, r)
    })
}

func main() {
    r := mux.NewRouter()
    r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "Welcome to Go server!")
    })
    r.HandleFunc("/user/{id}", func(w http.ResponseWriter, r *http.Request) {
        vars := mux.Vars(r)
        fmt.Fprintf(w, "User ID: %s", vars["id"])
    })
    r.Use(loggingMiddleware)
    log.Fatal(http.ListenAndServe(":8080", r))
}
```
**Scenarios**:
- RESTful endpoints.
- Middleware for logging.
- Dynamic routing.

### 21. Databases
**Overview**: Use `database/sql` for SQL databases (e.g., SQLite).

**Example**: SQLite operations.
```go
package main

import (
    "database/sql"
    "fmt"
    _ "github.com/mattn/go-sqlite3"
)

func main() {
    db, err := sql.Open("sqlite3", "./example.db")
    if err != nil {
        fmt.Println("Error opening DB:", err)
        return
    }
    defer db.Close()

    // Create table
    _, err = db.Exec(`CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT)`)
    if err != nil {
        fmt.Println("Error creating table:", err)
        return
    }

    // Insert
    _, err = db.Exec("INSERT INTO users (name) VALUES (?)", "Alice")
    if err != nil {
        fmt.Println("Error inserting:", err)
        return
    }

    // Query
    rows, err := db.Query("SELECT id, name FROM users")
    if err != nil {
        fmt.Println("Error querying:", err)
        return
    }
    defer rows.Close()
    for rows.Next() {
        var id int
        var name string
        rows.Scan(&id, &name)
        fmt.Printf("ID: %d, Name: %s\n", id, name)
    }
}
```
**Scenarios**:
- CRUD operations.
- Using drivers for PostgreSQL/MySQL.

### 22. Microservices and APIs
**Overview**: Build REST APIs or gRPC services.

**Example**: REST API.
```go
package main

import (
    "encoding/json"
    "net/http"
    "github.com/gorilla/mux"
)

type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

var users = []User{{ID: 1, Name: "Alice"}}

func getUsers(w http.ResponseWriter, r *http.Request) {
    json.NewEncoder(w).Encode(users)
}

func main() {
    r := mux.NewRouter()
    r.HandleFunc("/users", getUsers).Methods("GET")
    http.ListenAndServe(":8080", r)
}
```
**Scenarios**:
- RESTful API design.
- gRPC or message brokers (not shown).

### 23. Deployment and DevOps
**Overview**: Deploy Go apps with Docker and CI/CD.

**Example**: Dockerizing a Go app.
**`main.go`**:
```go
package main

import (
    "fmt"
    "net/http"
)

func main() {
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "Hello from Docker!")
    })
    http.ListenAndServe(":8080", nil)
}
```
**`Dockerfile`**:
```dockerfile
FROM golang:1.21 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o app

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/app .
EXPOSE 8080
CMD ["./app"]
```
**Steps**:
1. Build: `docker build -t go-app .`.
2. Run: `docker run -p 8080:8080 go-app`.

**Scenarios**:
- Static binaries for portability.
- Multi-stage Docker builds.

---

## Next Steps
- Practice each example in a Go environment.
- Explore the [Go Tour](https://tour.golang.org).
- Build projects like a CLI tool or REST API.
- Refer to [golang.org](https://golang.org/doc) for further documentation.
```

---

### How to Download
1. Copy the above Markdown content.
2. Paste it into a text editor (e.g., VS Code, Notepad).
3. Save the file as `go-learning-roadmap.md`.
4. Alternatively, use a terminal command to create the file:
   ```bash
   cat <<EOF > go-learning-roadmap.md
   [Paste the Markdown content here]
   EOF
   ```
5. The file can be opened in any Markdown viewer or converted to PDF/HTML using tools like Pandoc or a Markdown editor.

If you need help with specific topics, additional examples, or converting the Markdown to another format, let me know!
