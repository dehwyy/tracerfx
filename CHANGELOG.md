# Changelog

## v0.4.0 (proposed)

Added:
- `TRACERFX_DSPAN_FAST` (read once, default off). Cheap `dspan.Start` for spans that are not sampled: cached SDK tracer, lazy span logger, raw attribute values expanded only on an unexpected `Err`, `Span started` / `Span ended` and `span_caller` only when Debug is enabled. The exported Error span of an unsampled trace carries the same attributes as before.
- `log.DebugEnabled(l)`, `log.DebugEnabler`, `log.LazyFromContext(ctx, sc)`, `log.NewLazySpanLogger(base, sc)`.
- `dspan.StartBatch(ctx, name, attrs...)` and `span.Link(ctx)`. `Link` adds a link from the span (a light span links on its batch) to the span in ctx, at most 128 per batch, invalid, nil and self links are skipped.
- `TRACERFX_SPAN_DEPTH=all|app|batch` (read once, default `all`). Inside a batch started by `StartBatch`, `app` makes `infrastructure.*` spans light and `batch` makes every span light. A light span has no OTel span, shares the batch context and trace_id, and an unexpected `Err` on it logs the Error line, records the exception on the batch span and sets the batch status to Error. The batch span gets `batch.failed` and `batch.suppressed_spans` on `End`. An unknown value is logged once and means `all`.

Fixed:
- none.

No breaking changes. Without the two envs nothing changes. Rollback: unset them or set `TRACERFX_DSPAN_FAST=off` and `TRACERFX_SPAN_DEPTH=all`.

## v0.3.0 (proposed)

Added:
- Expected errors. `tracer.WithExpectedErrors(predicates...)`, `dspan.RegisterExpectedErrors(predicates...)`, `dspan.ResetExpectedErrors()`. For an error that matches a predicate `span.Err` sets no span status and records no exception event, adds the attribute `error.expected=true`, logs at Debug as `Span recorded expected error` and returns the same error. Registration at `tracer.FxModule` build and at `tracer.NewProvider`. Env `TRACERFX_EXPECTED_ERRORS=off` (read once) restores the old behaviour. The library registers no predicates, so without registration nothing changes.
- `log.SetGlobalLevelFromEnv()`.

Changed:
- Sampling at `TRACER_SAMPLE_RATIO` / `WithSampleRatio` below 1: spans that are not sampled are now `RecordOnly` (v0.2.0: `Drop`), the sampler is `ParentBased` with the ratio root and `RecordOnly` for unsampled local and remote parents. A span that ends with status Error is exported even when its trace is not sampled (alone, without its unsampled parents); sampled spans are exported once. `IsRecording()` is true for all spans at ratio below 1. Ratio 1, no env: `AlwaysSample` and a plain batcher, unchanged.
- The zerolog global level is set from `TRACERFX_LOG_LEVEL` by `tracer.FxModule` (at build) and `Provider.Start`. Without the env it was Debug (the zerolog default) and is now Info. Debug lines of global zerolog loggers (brokerfx, ratelimitfx, circuitbreakerfx) disappear, and the global level is a lower bound for every zerolog logger in the process, including `log.New(log.WithLevel(log.LevelDebug))`. Back to the old output: `TRACERFX_LOG_LEVEL=debug`.
- `tracer.FxModule` builds the `Provider` from the options it already built for the logger instead of parsing the options twice.

Fixed:
- none.

No breaking changes to the public API. Behaviour changes are the default global log level and, at a sample ratio below 1, the sampling decision (both under Changed).

## v0.2.0 (proposed)

Added:
- `TRACER_SAMPLE_RATIO` env and `tracer.WithSampleRatio(ratio)` option. Default `1.0` keeps `AlwaysSample`; a ratio below 1 uses `ParentBased(TraceIDRatioBased(ratio))`. Invalid values fall back to 1.0 with a warning at start.

Fixed:
- none.

No breaking changes: without the env/option nothing changes.

## v0.1.0 (proposed)

Breaking (v0 minor):
- `log.Logger` gains `Debug` and `Warn`. Custom implementations of `log.Logger` must add both methods. `log.DebugLogger` is now an alias of `Logger`, `log.Debug(l, ...)` calls `l.Debug`; both are deprecated.
- `nats.ExtractTraceContext` no longer starts and ends a `nats.consumer` span. Use `nats.ConsumerSpan` around the handler.
- dspan log lines no longer carry an explicit `trace_id` field of their own; it comes from the logger (`FromContext`) together with `span_id`. Field name and value are unchanged.
- `middleware.TraceMiddleware` now delegates to `NewTrace`: it wraps the `ResponseWriter` (`http.Flusher`, `http.Hijacker`, `http.Pusher`, `Unwrap` are preserved), adds `http.request.method`, `url.path`, `http.response.status_code` (and `http.route` when `r.Pattern` is set) and renames the span to `HTTP <METHOD> <route>` when a route is known; otherwise the span name stays `http.request`. 5xx sets span status to Error.
- Log format of fields changed: `time.Duration` is written as milliseconds (was nanoseconds), values implementing `fmt.Stringer` (proto messages, structs with `String()`) are written as the `String()` text (were JSON objects). Check Loki/Grafana queries and dashboards that read these fields.

Added:
- `log.FromContext` adds `trace_id` / `span_id` for a valid span; `log.Default()`, `log.New(WithWriter, WithLevel)`, `log.Level`, `log.ParseLevel`, `log.Err`, `log.Attr`.
- Typed log fields (error, time, duration, Stringer, bytes); odd or non-string keys become `!BADKEY`.
- `log.NewFxEventLogger`, `tracer.FxEventLogger()`.
- `middleware.NewTrace` with `WithRouteFunc`, `WithPropagation`, `WithSpanNameFunc`; `http.route`, status code, 5xx error status.
- `nats.ExtractCore`, `nats.InjectMap`, `nats.ExtractMap`, `nats.ConsumerSpan`.
- `dspan.SetName`.

Fixed:
- `error` field was written as `{}`.
- Odd trailing log args were silently dropped.
- NATS consumer span was closed before the handler ran.
- `log.SetDefaultLogger` was a data race.
- `NewZapLogger(nil)` ignored the `NewProduction` error.

## v0.0.10

- Span start/end logged at debug; built-in zerolog gated by `TRACERFX_LOG_LEVEL`.
