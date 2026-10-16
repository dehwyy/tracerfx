package dspan

import (
	"os"
	"strings"
	"sync/atomic"
)

const ExpectedErrorsEnv = "TRACERFX_EXPECTED_ERRORS"

const expectedErrorAttribute = "error.expected"

const (
	expectedStateUnset int32 = iota
	expectedStateOn
	expectedStateOff
)

var (
	expectedPredicates atomic.Pointer[[]func(error) bool]
	expectedState      atomic.Int32
)

func expectedErrorsEnabled() bool {
	switch expectedState.Load() {
	case expectedStateOn:
		return true
	case expectedStateOff:
		return false
	}

	state := expectedStateOn
	if strings.EqualFold(
		strings.TrimSpace(os.Getenv(ExpectedErrorsEnv)),
		"off",
	) {
		state = expectedStateOff
	}

	expectedState.CompareAndSwap(
		expectedStateUnset,
		state,
	)

	return expectedState.Load() == expectedStateOn
}

func isExpectedError(err error) bool {
	if !expectedErrorsEnabled() {
		return false
	}

	predicates := expectedPredicates.Load()
	if predicates == nil {
		return false
	}

	for _, predicate := range *predicates {
		if predicate(err) {
			return true
		}
	}

	return false
}
