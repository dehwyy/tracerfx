package dspan

func Response[T any](s *dspan, response T) T {
	s.WithAttribute("response", response)
	return response
}
