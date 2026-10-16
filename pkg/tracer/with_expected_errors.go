package tracer

func WithExpectedErrors(predicates ...func(error) bool) Option {
	return optionFunc(
		func(o *options) {
			o.expectedErrors = append(
				o.expectedErrors,
				predicates...,
			)
		},
	)
}
