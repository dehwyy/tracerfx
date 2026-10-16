package dspan

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/dehwyy/tracerfx/pkg/tracer/log"
	"go.opentelemetry.io/otel/trace"
)

const pendingInline = 4

type dspan struct {
	TraceSpan  trace.Span
	Logger     log.Logger
	Attributes map[string]any
	Mu         sync.Mutex
	StartTime  time.Time
	SpanName   string
	SpanCaller string

	pending    []pendingAttribute
	pendingBuf [pendingInline]pendingAttribute
	deferred   bool
	fast       bool
	light      bool
	scope      *batchScope
	links      atomic.Int32
}
