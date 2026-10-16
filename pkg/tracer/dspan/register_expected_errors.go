package dspan

func RegisterExpectedErrors(predicates ...func(error) bool) {
	added := make(
		[]func(error) bool,
		0,
		len(predicates),
	)
	for _, predicate := range predicates {
		if predicate != nil {
			added = append(
				added,
				predicate,
			)
		}
	}

	if len(added) == 0 {
		return
	}

	for {
		current := expectedPredicates.Load()

		var merged []func(error) bool
		if current != nil {
			merged = append(
				merged,
				*current...,
			)
		}
		merged = append(
			merged,
			added...,
		)

		if expectedPredicates.CompareAndSwap(
			current,
			&merged,
		) {
			return
		}
	}
}
