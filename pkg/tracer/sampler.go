package tracer

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	tracesdk "go.opentelemetry.io/otel/sdk/trace"
)

const SampleRatioEnv = "TRACER_SAMPLE_RATIO"

const defaultSampleRatio = 1.0

func validSampleRatio(ratio float64) bool {
	return !math.IsNaN(ratio) && ratio >= 0 && ratio <= 1
}

func parseSampleRatio(raw string) (float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultSampleRatio, nil
	}

	ratio, err := strconv.ParseFloat(
		raw,
		64,
	)
	if err != nil {
		return defaultSampleRatio, fmt.Errorf(
			"%s=%q is not a number: %w",
			SampleRatioEnv,
			raw,
			err,
		)
	}

	if !validSampleRatio(ratio) {
		return defaultSampleRatio, fmt.Errorf(
			"%s=%q is outside [0, 1]",
			SampleRatioEnv,
			raw,
		)
	}

	return ratio, nil
}

func newSampler(ratio float64) tracesdk.Sampler {
	if ratio >= 1 {
		return tracesdk.AlwaysSample()
	}

	unsampled := newRecordOnlySampler(tracesdk.NeverSample())

	return tracesdk.ParentBased(
		newRecordOnlySampler(tracesdk.TraceIDRatioBased(ratio)),
		tracesdk.WithRemoteParentNotSampled(unsampled),
		tracesdk.WithLocalParentNotSampled(unsampled),
	)
}
