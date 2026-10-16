package dspan

func ResetExpectedErrors() {
	expectedPredicates.Store(nil)
}
