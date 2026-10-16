package tracer

import (
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
)

type recordOnlySampler struct {
	tracesdk.Sampler
}

func newRecordOnlySampler(inner tracesdk.Sampler) tracesdk.Sampler {
	return recordOnlySampler{
		Sampler: inner,
	}
}

func (s recordOnlySampler) ShouldSample(parameters tracesdk.SamplingParameters) tracesdk.SamplingResult {
	result := s.Sampler.ShouldSample(parameters)
	if result.Decision == tracesdk.Drop {
		result.Decision = tracesdk.RecordOnly
	}

	return result
}
