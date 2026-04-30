package benchmarkrun_test

import (
	"testing"

	"pg_gobench/internal/benchmarkrun"
)

func TestSampleBuildsConsistentJSONAndPrometheusViews(t *testing.T) {
	sample := benchmarkrun.Sample{
		Latency: benchmarkrun.LatencySample{
			MinMilliseconds:     4,
			MaxMilliseconds:     12,
			AverageMilliseconds: 8,
			P50Milliseconds:     4,
			P90Milliseconds:     12,
			P95Milliseconds:     12,
			P99Milliseconds:     12,
			Buckets: []benchmarkrun.LatencyHistogramBucket{
				{UpperBoundSeconds: 0.004, CumulativeCount: 1},
				{UpperBoundSeconds: 0.012, CumulativeCount: 2},
			},
			Count:      2,
			SumSeconds: 0.016,
		},
		ElapsedSeconds:       10,
		TotalOperations:      2,
		SuccessfulOperations: 1,
		FailedOperations:     1,
		ActiveClients:        1,
		ConfiguredClients:    2,
		OperationCounts: benchmarkrun.OperationCounts{
			PointRead:      2,
			Join:           4,
			Aggregation:    6,
			LockContention: 8,
			HotUpdate:      10,
		},
		LatestError: "worker failed compactly",
	}

	stats := sample.Stats()
	if stats.TPS != 0.2 {
		t.Fatalf("stats TPS = %v, want %v", stats.TPS, 0.2)
	}
	if stats.OperationRates.PointRead != 0.2 {
		t.Fatalf("stats point-read rate = %v, want %v", stats.OperationRates.PointRead, 0.2)
	}
	if stats.OperationRates.Join != 0.4 {
		t.Fatalf("stats join rate = %v, want %v", stats.OperationRates.Join, 0.4)
	}
	if stats.OperationRates.Aggregation != 0.6 {
		t.Fatalf("stats aggregation rate = %v, want %v", stats.OperationRates.Aggregation, 0.6)
	}
	if stats.OperationRates.LockContention != 0.8 {
		t.Fatalf("stats lock-contention rate = %v, want %v", stats.OperationRates.LockContention, 0.8)
	}
	if stats.OperationRates.HotUpdate != 1 {
		t.Fatalf("stats hot-update rate = %v, want %v", stats.OperationRates.HotUpdate, 1.0)
	}
	if stats.LatestError != "worker failed compactly" {
		t.Fatalf("stats latest error = %q, want compact sample error", stats.LatestError)
	}

	metrics := sample.Metrics(true)
	if !metrics.RunActive {
		t.Fatal("metrics RunActive = false, want true")
	}
	if metrics.TPS != stats.TPS {
		t.Fatalf("metrics TPS = %v, want %v", metrics.TPS, stats.TPS)
	}
	if metrics.OperationErrorsTotal != stats.FailedOperations {
		t.Fatalf("metrics failed ops = %d, want %d", metrics.OperationErrorsTotal, stats.FailedOperations)
	}
	if metrics.OperationLatency.Count != 2 {
		t.Fatalf("metrics histogram count = %d, want %d", metrics.OperationLatency.Count, 2)
	}
	if got := histogramBucketCount(metrics.OperationLatency.Buckets, 0.012); got != 2 {
		t.Fatalf("metrics bucket le=0.012 count = %d, want %d", got, 2)
	}
}

func histogramBucketCount(buckets []benchmarkrun.LatencyHistogramBucket, upperBoundSeconds float64) uint64 {
	for _, bucket := range buckets {
		if bucket.UpperBoundSeconds == upperBoundSeconds {
			return bucket.CumulativeCount
		}
	}
	return 0
}
