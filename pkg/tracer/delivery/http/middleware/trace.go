package middleware

import (
	"net/http"

	"github.com/dehwyy/tracerfx/pkg/tracer/dspan"
)

const TraceIDHeaderName = "X-Trace-Id"

func TraceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctxTrace, span := dspan.Start(ctx, "http.request", dspan.Attr("method", r.Method), dspan.Attr("url", r.URL.String()))
			defer span.End()

			w.Header().Set(TraceIDHeaderName, span.TraceID())
			next.ServeHTTP(w, r.WithContext(ctxTrace))
		},
	)
}
