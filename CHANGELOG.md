# Changelog

## v0.1.0 (proposed)

Breaking (v0 minor):
- `log.Logger` gains `Debug` and `Warn`. Custom implementations of `log.Logger` must add both methods. `log.DebugLogger` is now an alias of `Logger`, `log.Debug(l, ...)` calls `l.Debug`; both are deprecated.
- `nats.ExtractTraceContext` no longer starts and ends a `nats.consumer` span. Use `nats.ConsumerSpan` around the handler.
- dspan log lines no longer carry an explicit `trace_id` field of their own; it comes from the logger (`FromContext`) together with `span_id`. Field name and value are unchanged.

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
