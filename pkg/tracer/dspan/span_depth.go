package dspan

import (
	"os"
	"strings"
	"sync/atomic"

	"github.com/dehwyy/tracerfx/pkg/tracer/log"
)

const SpanDepthEnv = "TRACERFX_SPAN_DEPTH"

const lightInfrastructurePrefix = "infrastructure."

const (
	depthUnset int32 = iota
	depthAll
	depthApp
	depthBatch
)

var depthState atomic.Int32

func parseSpanDepth(raw string) (int32, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "all":
		return depthAll, true
	case "app":
		return depthApp, true
	case "batch":
		return depthBatch, true
	}

	return depthAll, false
}

func spanDepth() int32 {
	if depth := depthState.Load(); depth != depthUnset {
		return depth
	}

	raw := os.Getenv(SpanDepthEnv)
	depth, valid := parseSpanDepth(raw)

	if depthState.CompareAndSwap(
		depthUnset,
		depth,
	) && !valid {
		log.Default().Warn(
			"tracerfx span depth ignored, using all",
			"env",
			SpanDepthEnv,
			"value",
			raw,
		)
	}

	return depthState.Load()
}

func depthSuppresses(depth int32, spanName string) bool {
	switch depth {
	case depthBatch:
		return true
	case depthApp:
		return strings.HasPrefix(
			spanName,
			lightInfrastructurePrefix,
		)
	}

	return false
}
