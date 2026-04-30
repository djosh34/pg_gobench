package benchrunner

import (
	"testing"
	"time"

	"pg_gobench/internal/benchmarkrun"
)

func TestRunStatsSnapshotComputesP95AndP99FromBoundedHistogram(t *testing.T) {
	startedAt := time.Date(2026, 4, 30, 9, 0, 0, 0, time.UTC)
	stats := newRunStats(startedAt, 0, 1)

	for milliseconds := 1; milliseconds <= 100; milliseconds++ {
		latency := time.Duration(milliseconds) * time.Millisecond
		stats.record(operationKindPointRead, latency, startedAt.Add(latency), nil)
	}

	snapshot := stats.sample(startedAt.Add(2 * time.Second)).Stats()

	if snapshot.TotalOperations != 100 {
		t.Fatalf("TotalOperations = %d, want %d", snapshot.TotalOperations, 100)
	}
	if snapshot.Latency.MinMilliseconds != 1 {
		t.Fatalf("MinMilliseconds = %v, want %v", snapshot.Latency.MinMilliseconds, 1.0)
	}
	if snapshot.Latency.MaxMilliseconds != 100 {
		t.Fatalf("MaxMilliseconds = %v, want %v", snapshot.Latency.MaxMilliseconds, 100.0)
	}
	if snapshot.Latency.AverageMilliseconds != 50.5 {
		t.Fatalf("AverageMilliseconds = %v, want %v", snapshot.Latency.AverageMilliseconds, 50.5)
	}
	if snapshot.Latency.P95Milliseconds != 95 {
		t.Fatalf("P95Milliseconds = %v, want %v", snapshot.Latency.P95Milliseconds, 95.0)
	}
	if snapshot.Latency.P99Milliseconds != 99 {
		t.Fatalf("P99Milliseconds = %v, want %v", snapshot.Latency.P99Milliseconds, 99.0)
	}
}

func TestRunStatsMetricsSnapshotExportsPrometheusHistogramInSeconds(t *testing.T) {
	startedAt := time.Date(2026, 4, 30, 9, 0, 0, 0, time.UTC)
	stats := newRunStats(startedAt, 0, 2)

	stats.workerStarted()
	stats.record(operationKindPointRead, 4*time.Millisecond, startedAt.Add(4*time.Millisecond), nil)
	stats.record(operationKindPointRead, 12*time.Millisecond, startedAt.Add(12*time.Millisecond), nil)

	snapshot := stats.sample(startedAt.Add(2 * time.Second)).Metrics(false)

	if snapshot.OperationLatency.Count != 2 {
		t.Fatalf("OperationLatency.Count = %d, want %d", snapshot.OperationLatency.Count, 2)
	}
	if snapshot.OperationLatency.SumSeconds != 0.016 {
		t.Fatalf("OperationLatency.SumSeconds = %v, want %v", snapshot.OperationLatency.SumSeconds, 0.016)
	}
	if got := histogramBucketCount(snapshot.OperationLatency.Buckets, 0.003); got != 0 {
		t.Fatalf("bucket le=0.003 count = %d, want %d", got, 0)
	}
	if got := histogramBucketCount(snapshot.OperationLatency.Buckets, 0.004); got != 1 {
		t.Fatalf("bucket le=0.004 count = %d, want %d", got, 1)
	}
	if got := histogramBucketCount(snapshot.OperationLatency.Buckets, 0.012); got != 2 {
		t.Fatalf("bucket le=0.012 count = %d, want %d", got, 2)
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
