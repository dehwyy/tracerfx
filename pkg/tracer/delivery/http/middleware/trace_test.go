package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

const incomingTraceparent = "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01"

func setupRecorder(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()

	recorder := tracetest.NewSpanRecorder()
	previousProvider := otel.GetTracerProvider()
	previousPropagator := otel.GetTextMapPropagator()
	otel.SetTracerProvider(tracesdk.NewTracerProvider(tracesdk.WithSpanProcessor(recorder)))
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		otel.SetTracerProvider(previousProvider)
		otel.SetTextMapPropagator(previousPropagator)
	})

	return recorder
}

func attrMap(attrs []attribute.KeyValue) map[string]attribute.Value {
	out := make(map[string]attribute.Value, len(attrs))
	for _, kv := range attrs {
		out[string(kv.Key)] = kv.Value
	}
	return out
}

func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	mux.HandleFunc("GET /boom", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	mux.HandleFunc("GET /implicit", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	return mux
}

func TestNewTrace(t *testing.T) {
	tests := []struct {
		name        string
		opts        []TraceOption
		path        string
		traceparent string
		wantName    string
		wantRoute   string
		wantStatus  int64
		wantErr     bool
		wantRemote  bool
	}{
		{
			name:       "route template from mux pattern",
			opts:       nil,
			path:       "/orders/42",
			wantName:   "HTTP GET /orders/{id}",
			wantRoute:  "/orders/{id}",
			wantStatus: http.StatusCreated,
		},
		{
			name:       "5xx marks span as error",
			path:       "/boom",
			wantName:   "HTTP GET /boom",
			wantRoute:  "/boom",
			wantStatus: http.StatusInternalServerError,
			wantErr:    true,
		},
		{
			name:       "implicit 200",
			path:       "/implicit",
			wantName:   "HTTP GET /implicit",
			wantRoute:  "/implicit",
			wantStatus: http.StatusOK,
		},
		{
			name:       "unmatched path keeps legacy span name",
			path:       "/missing",
			wantName:   "http.request",
			wantStatus: http.StatusNotFound,
		},
		{
			name:        "propagation off ignores traceparent",
			opts:        []TraceOption{WithPropagation(false)},
			path:        "/implicit",
			traceparent: incomingTraceparent,
			wantName:    "HTTP GET /implicit",
			wantRoute:   "/implicit",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "propagation on continues traceparent",
			opts:        []TraceOption{WithPropagation(true)},
			path:        "/implicit",
			traceparent: incomingTraceparent,
			wantName:    "HTTP GET /implicit",
			wantRoute:   "/implicit",
			wantStatus:  http.StatusOK,
			wantRemote:  true,
		},
		{
			name: "custom route and name funcs",
			opts: []TraceOption{
				WithRouteFunc(func(*http.Request) string { return "/custom" }),
				WithSpanNameFunc(func(r *http.Request, route string) string { return r.Method + "|" + route }),
			},
			path:       "/implicit",
			wantName:   "GET|/custom",
			wantRoute:  "/custom",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := setupRecorder(t)
			handler := NewTrace(tt.opts...)(newMux())

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.traceparent != "" {
				req.Header.Set("traceparent", tt.traceparent)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			spans := recorder.Ended()
			require.Len(t, spans, 1)
			span := spans[0]

			require.Equal(t, tt.wantName, span.Name())
			attrs := attrMap(span.Attributes())
			require.Equal(t, tt.wantStatus, attrs["http.response.status_code"].AsInt64())
			require.Equal(t, "GET", attrs["http.request.method"].AsString())
			require.Equal(t, tt.path, attrs["url.path"].AsString())
			if tt.wantRoute != "" {
				require.Equal(t, tt.wantRoute, attrs["http.route"].AsString())
			} else {
				require.NotContains(t, attrs, "http.route")
			}

			if tt.wantErr {
				require.Equal(t, codes.Error, span.Status().Code)
			} else {
				require.NotEqual(t, codes.Error, span.Status().Code)
			}

			if tt.wantRemote {
				require.True(t, span.Parent().IsRemote())
				require.Equal(t, "0af7651916cd43dd8448eb211c80319c", span.SpanContext().TraceID().String())
			} else {
				require.False(t, span.Parent().IsValid())
				require.NotEqual(t, "0af7651916cd43dd8448eb211c80319c", span.SpanContext().TraceID().String())
			}

			require.Equal(t, span.SpanContext().TraceID().String(), rec.Header().Get(TraceIDHeaderName))
		})
	}
}

func TestTraceMiddlewareKeepsLegacyBehavior(t *testing.T) {
	recorder := setupRecorder(t)
	handler := TraceMiddleware(newMux())

	req := httptest.NewRequest(http.MethodGet, "/orders/1", nil)
	req.Header.Set("traceparent", incomingTraceparent)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	require.False(t, spans[0].Parent().IsValid())
	attrs := attrMap(spans[0].Attributes())
	require.Equal(t, "GET", attrs["method"].AsString())
}

func TestStatusRecorderKeepsFlusherAndUnwrap(t *testing.T) {
	inner := httptest.NewRecorder()
	recorder := &statusRecorder{ResponseWriter: inner}

	var w http.ResponseWriter = recorder
	flusher, ok := w.(http.Flusher)
	require.True(t, ok)
	flusher.Flush()

	require.True(t, inner.Flushed)
	require.Same(t, inner, recorder.Unwrap())
	require.Equal(t, http.StatusOK, recorder.statusCode())
}

func TestTraceMiddlewarePreservesHijacker(t *testing.T) {
	hijackable := make(chan bool, 1)
	server := httptest.NewServer(TraceMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hijacker, ok := w.(http.Hijacker)
		hijackable <- ok
		if !ok {
			return
		}
		conn, _, err := hijacker.Hijack()
		if err != nil {
			t.Errorf("hijack: %v", err)
			return
		}
		if err := conn.Close(); err != nil {
			t.Errorf("close: %v", err)
		}
	})))
	t.Cleanup(server.Close)

	resp, err := http.Get(server.URL)
	if err == nil {
		require.NoError(t, resp.Body.Close())
	}

	require.True(t, <-hijackable)
}
