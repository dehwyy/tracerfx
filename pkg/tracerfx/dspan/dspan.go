package dspan

import (
	"sync"
	"time"

	"github.com/dehwyy/tracerfx/pkg/tracerfx/log"
	"go.opentelemetry.io/otel/trace"
)

type dspan struct {
	TraceSpan  trace.Span
	Logger     log.Logger
	Attributes map[string]any
	Mu         sync.Mutex
	StartTime  time.Time
	SpanName   string
}
