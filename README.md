# TracerFX

TracerFX is a powerful tracing and logging library for Go, built on top of OpenTelemetry and Uber FX. It simplifies span management, automatic logging, and attribute extraction using reflection.

## Project Structure

- `pkg/tracer`: Core provider and options.
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
    "github.com/dehwyy/tracerfx/pkg/tracer"
    "github.com/dehwyy/tracerfx/pkg/tracer/log"
)

// ...

app := fx.New(
    tracer.FxModule(
        tracer.WithServiceName("my-awesome-service"),
        tracer.WithLogger(log.NewZapLogger(nil)), // Optional: defaults to zerolog
    ),
    // other modules...
)
app.Run()
```

### Manual Initialization

```go
import (
    "context"
    "github.com/dehwyy/tracerfx/pkg/tracer"
)

func main() {
    ctx := context.Background()
    provider := tracer.NewProvider(
        tracer.WithServiceName("my-awesome-service"),
        tracer.WithHost("localhost:4317"),
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
    "github.com/dehwyy/tracerfx/pkg/tracer/dspan"
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

## Logger

`log.Logger` wraps zerolog. Services do not import `rs/zerolog` or its global `log` package.

```go
type Logger interface {
    Debug(msg string, args ...any)
    Info(msg string, args ...any)
    Warn(msg string, args ...any)
    Error(msg string, args ...any)
    With(args ...any) Logger
}
```

```go
l := log.FromContext(ctx)
l.Warn("retry scheduled", "attempt", 3, "delay", time.Second, log.Err(err))
l.Error("failed", "error", err)
```

- `FromContext(ctx)` adds `trace_id` and `span_id` when ctx holds a valid span. Outside a span, and at startup, use `log.Default()` (`tracer.FxModule` sets it at build time).
- Fields are key/value pairs or `log.Attr`. Typed branches: `error` (text), `time.Time`, `time.Duration` (ms), `fmt.Stringer`, `[]byte`. A dangling or non-string key is written as `!BADKEY`.
- `log.New(log.WithWriter(w), log.WithLevel(log.LevelDebug))`. The default level comes from `TRACERFX_LOG_LEVEL` (`debug|info|warn|error`).
- There is no `Fatal`: return an error from the constructor and let fx stop the app.
- `log.DebugLogger` and `log.Debug(l, ...)` are deprecated and will be removed in the next release.

### fx events

```go
fx.New(
    tracer.FxModule(tracer.WithServiceName("svc")),
    tracer.FxEventLogger(),
)
```

`FxEventLogger` must be passed at the root `fx.New`, not inside `fx.Module`.

## HTTP Middleware

```go
handler := middleware.NewTrace(
    middleware.WithPropagation(false),
    middleware.WithRouteFunc(func(r *http.Request) string { return r.Pattern }),
)(mux)
```

- `TraceMiddleware(next)` is unchanged: span `http.request`, `X-Trace-Id` response header, no incoming `traceparent`.
- `NewTrace` options: `WithRouteFunc` (called after the handler, with the request the handler saw; default reads `r.Pattern` of `http.ServeMux`), `WithPropagation(true)` to continue an incoming W3C `traceparent` (off by default: do not trust it on public APIs), `WithSpanNameFunc`.
- Attributes: `http.request.method`, `http.route`, `http.response.status_code`, `url.path`. Status 5xx sets span status to Error. The span is named `HTTP <METHOD> <route>`; without a route it stays `http.request`.

## NATS

```go
nats.InjectTraceContext(ctx, msg)
nats.InjectMap(ctx, headers)

ctx, err := nats.ExtractTraceContext(ctx, jsMsg)
ctx = nats.ExtractCore(ctx, coreMsg)
ctx = nats.ExtractMap(ctx, headers)

ctx, done := nats.ConsumerSpan(ctx, subject)
defer done()
```

`ExtractTraceContext` only restores the remote span context. It no longer starts and ends a `nats.consumer` span before the handler runs; open it with `ConsumerSpan` and call `done` after the handler.

## GitHub Actions Integration

The project includes a GitHub Action in `.github/workflows/release.yml` that:
1. Runs all tests on every push to `main`.
2. Automatically increments the version (patch) and creates a new GitHub Release if tests pass.
