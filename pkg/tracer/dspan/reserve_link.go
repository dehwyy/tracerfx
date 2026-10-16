package dspan

func (s *dspan) reserveLink() bool {
	for {
		current := s.links.Load()
		if current >= maxLinks {
			return false
		}

		if s.links.CompareAndSwap(
			current,
			current+1,
		) {
			return true
		}
	}
}
