# TracerFX

TracerFX is a powerful tracing and logging library for Go, built on top of OpenTelemetry and Uber FX. It simplifies span management, automatic logging, and attribute extraction using reflection.

## Project Structure

- `pkg/tracerfx`: Core provider and options.
  - `caller`: Logic for automatic function name retrieval from the call stack.
  - `delivery/http/middleware`: HTTP middleware for automatic trace ID injection and span management.
  - `dspan`: The core span implementation.
  - `log`: Logger interface and adapters for `zap` and `zerolog`.

## Installation

```bash
go get github.com/dehwyy/tracerfx
```

## Initialization

### Using Uber FX (Recommended)

```go
import (
    "github.com/dehwyy/tracerfx/pkg/tracerfx"
    "github.com/dehwyy/tracerfx/pkg/tracerfx/log"
)

// ...

app := fx.New(
    tracerfx.FxModule(
        tracerfx.WithServiceName("my-awesome-service"),
        tracerfx.WithLogger(log.NewZapLogger(nil)), // Optional: defaults to zerolog
    ),
    // other modules...
)
app.Run()
```

### Manual Initialization

```go
import (
    "context"
    "github.com/dehwyy/tracerfx/pkg/tracerfx"
)

func main() {
    ctx := context.Background()
    provider := tracerfx.NewProvider(
        tracerfx.WithServiceName("my-awesome-service"),
        tracerfx.WithHost("localhost:4317"),
    )

    if err := provider.Start(ctx); err != nil {
        panic(err)
    }
    defer provider.Stop(ctx)
}
```

## Usage Examples (dspan)

### Basic Span Management

```go
import (
    "context"
    "github.com/dehwyy/tracerfx/pkg/tracerfx/dspan"
)

func MyFunction(ctx context.Context) {
    // Start automatically captures the function name and logs "Span started"
    ctx, span := dspan.Start(ctx)
    defer span.End() // Automatically logs accumulated attributes and duration

    // Your logic here
}
```

### Adding Attributes

`WithAttribute` uses reflection to extract fields if a struct is provided. It returns the span instance for chaining.

```go
span.WithAttribute("user_id", 123).
     WithAttribute("request", inputStruct) // Automatically extracts exported fields
```

### Error Handling

Use `Err(err)` to record an error in the span and log it. This does **not** close the span.

```go
if err := DoSomething(); err != nil {
    span.Err(err)
    return err
}
```

### Retrieving Trace ID

```go
traceID := span.TraceID()
fmt.Printf("Current Trace ID: %s\n", traceID)
```

## HTTP Middleware

The `TraceIDHeader` middleware automatically starts a span for each request, injects the trace ID into the response header (`X-Trace-Id`), and cleans up when the request is finished.

```go
import "github.com/dehwyy/tracerfx/pkg/tracerfx/delivery/http/middleware"

mux := http.NewServeMux()
handler := middleware.TraceIDHeader(mux)
http.ListenAndServe(":8080", handler)
```

## GitHub Actions Integration

The project includes a GitHub Action in `.github/workflows/release.yml` that:
1. Runs all tests on every push to `main`.
2. Automatically increments the version (patch) and creates a new GitHub Release if tests pass.
