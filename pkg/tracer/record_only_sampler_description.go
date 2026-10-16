package tracer

func (s recordOnlySampler) Description() string {
	return "RecordOnlyUnsampled{" + s.Sampler.Description() + "}"
}
