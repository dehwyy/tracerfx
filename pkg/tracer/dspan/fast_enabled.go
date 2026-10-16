package dspan

import (
	"os"
	"strings"
	"sync/atomic"
)

const FastEnv = "TRACERFX_DSPAN_FAST"

const (
	fastStateUnset int32 = iota
	fastStateOn
	fastStateOff
)

var fastState atomic.Int32

func fastEnabled() bool {
	switch fastState.Load() {
	case fastStateOn:
		return true
	case fastStateOff:
		return false
	}

	state := fastStateOff
	if strings.EqualFold(
		strings.TrimSpace(os.Getenv(FastEnv)),
		"on",
	) {
		state = fastStateOn
	}

	fastState.CompareAndSwap(
		fastStateUnset,
		state,
	)

	return fastState.Load() == fastStateOn
}
