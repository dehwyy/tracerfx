package dspan

func (s *dspan) appendPending(key string, value any) {
	if s.pending == nil {
		s.pending = s.pendingBuf[:0]
	}

	s.pending = append(
		s.pending,
		pendingAttribute{
			key:   key,
			value: value,
		},
	)
}

func (s *dspan) expandPending() {
	if !s.deferred {
		return
	}

	if s.Attributes == nil {
		s.Attributes = make(map[string]any)
	}

	for _, attribute := range s.pending {
		for k, v := range extractFields(
			attribute.key,
			attribute.value,
		) {
			s.Attributes[k] = v
			setAttr(
				s.TraceSpan,
				k,
				v,
			)
		}
	}

	s.pending = nil
	s.pendingBuf = [pendingInline]pendingAttribute{}
	s.deferred = false
}

func (s *dspan) appendAttributeArgs(args []any) []any {
	if s.deferred {
		for _, attribute := range s.pending {
			for k, v := range extractFields(
				attribute.key,
				attribute.value,
			) {
				args = append(
					args,
					k,
					v,
				)
			}
		}

		return args
	}

	for k, v := range s.Attributes {
		args = append(
			args,
			k,
			v,
		)
	}

	return args
}
