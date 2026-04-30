package benchmarkrun_test

import (
	"bytes"
	"strings"
	"testing"

	"pg_gobench/internal/benchmarkrun"
)

func TestMetricsSnapshotWritePrometheusPrefixesEveryMetricName(t *testing.T) {
	snapshot := benchmarkrun.MetricsSnapshot{
		RunActive:            true,
		RunDurationSeconds:   12.5,
		ConfiguredClients:    4,
		ActiveClients:        3,
		OperationsTotal:      99,
		OperationErrorsTotal: 2,
		TPS:                  7.92,
		OperationLatency: benchmarkrun.LatencyHistogramSnapshot{
			Buckets: []benchmarkrun.LatencyHistogramBucket{
				{UpperBoundSeconds: 0.001, CumulativeCount: 1},
				{UpperBoundSeconds: 0.005, CumulativeCount: 3},
			},
			Count:      3,
			SumSeconds: 0.008,
		},
	}

	var output bytes.Buffer
	if err := snapshot.WritePrometheus(&output); err != nil {
		t.Fatalf("WritePrometheus returned error: %v", err)
	}

	for _, line := range strings.Split(strings.TrimSpace(output.String()), "\n") {
		name := strings.Fields(line)[0]
		if !strings.HasPrefix(name, "pg_gobench_") {
			t.Fatalf("metric line %q does not start with pg_gobench_", line)
		}
	}
}
