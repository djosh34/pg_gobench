package benchrunner

import (
	"testing"
	"time"
)

func TestRunStatsSnapshotComputesP95AndP99FromBoundedHistogram(t *testing.T) {
	startedAt := time.Date(2026, 4, 30, 9, 0, 0, 0, time.UTC)
	stats := newRunStats(startedAt, 0, 1)

	for milliseconds := 1; milliseconds <= 100; milliseconds++ {
		latency := time.Duration(milliseconds) * time.Millisecond
		stats.record(operationKindPointRead, latency, startedAt.Add(latency), nil)
	}

	snapshot := stats.snapshot(startedAt.Add(2 * time.Second))

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
