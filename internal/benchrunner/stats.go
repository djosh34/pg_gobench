package benchrunner

import (
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"pg_gobench/internal/benchmarkrun"
)

const latestErrorLimit = 160

var latencyBucketBounds = buildLatencyBucketBounds()

type runStats struct {
	mu                   sync.Mutex
	runStartedAt         time.Time
	measurementStartedAt time.Time
	finishedAt           *time.Time
	activeClients        int
	configuredClients    int
	totalOperations      uint64
	successfulOperations uint64
	failedOperations     uint64
	perKind              [operationKindCount]uint64
	latency              latencyHistogram
	latestError          string
}

func newRunStats(startedAt time.Time, warmup time.Duration, configuredClients int) *runStats {
	return &runStats{
		runStartedAt:         startedAt,
		measurementStartedAt: startedAt.Add(warmup),
		configuredClients:    configuredClients,
		latency:              newLatencyHistogram(),
	}
}

func (s *runStats) setConfiguredClients(configuredClients int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.configuredClients = configuredClients
}

func (s *runStats) workerStarted() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.activeClients++
}

func (s *runStats) workerStopped() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.activeClients > 0 {
		s.activeClients--
	}
}

func (s *runStats) record(kind operationKind, latency time.Duration, completedAt time.Time, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err != nil {
		s.latestError = compactErrorText(err.Error())
	}
	if completedAt.Before(s.measurementStartedAt) {
		return
	}

	s.totalOperations++
	if err == nil {
		s.successfulOperations++
	} else {
		s.failedOperations++
	}
	if kind >= 0 && kind < operationKindCount {
		s.perKind[kind]++
	}
	s.latency.record(latency)
}

func (s *runStats) finish(finishedAt time.Time, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.finishedAt == nil {
		finishedCopy := finishedAt
		s.finishedAt = &finishedCopy
	}
	if err != nil {
		s.latestError = compactErrorText(err.Error())
	}
}

func (s *runStats) snapshot(now time.Time) benchmarkrun.Stats {
	s.mu.Lock()
	defer s.mu.Unlock()

	endAt := now
	if s.finishedAt != nil {
		endAt = *s.finishedAt
	}
	elapsedSeconds := 0.0
	if endAt.After(s.measurementStartedAt) {
		elapsedSeconds = endAt.Sub(s.measurementStartedAt).Seconds()
	}

	stats := benchmarkrun.Stats{
		Latency:              s.latency.snapshot(),
		TPS:                  ratePerSecond(s.totalOperations, elapsedSeconds),
		TotalOperations:      s.totalOperations,
		SuccessfulOperations: s.successfulOperations,
		FailedOperations:     s.failedOperations,
		ActiveClients:        s.activeClients,
		ConfiguredClients:    s.configuredClients,
		ElapsedSeconds:       elapsedSeconds,
		OperationRates:       operationRatesSnapshot(s.perKind, elapsedSeconds),
		LatestError:          s.latestError,
	}
	if s.finishedAt == nil && !s.runStartedAt.IsZero() && now.Before(s.measurementStartedAt) {
		stats.ElapsedSeconds = 0
		stats.TPS = 0
		stats.OperationRates = benchmarkrun.OperationRates{}
	}
	return stats
}

type latencyHistogram struct {
	bounds []time.Duration
	counts []uint64
	total  uint64
	sum    time.Duration
	min    time.Duration
	max    time.Duration
}

func newLatencyHistogram() latencyHistogram {
	return latencyHistogram{
		bounds: latencyBucketBounds,
		counts: make([]uint64, len(latencyBucketBounds)+1),
	}
}

func (h *latencyHistogram) record(latency time.Duration) {
	if latency < 0 {
		latency = 0
	}
	index := sort.Search(len(h.bounds), func(i int) bool {
		return latency <= h.bounds[i]
	})
	h.counts[index]++
	h.total++
	h.sum += latency
	if h.total == 1 || latency < h.min {
		h.min = latency
	}
	if h.total == 1 || latency > h.max {
		h.max = latency
	}
}

func (h latencyHistogram) snapshot() benchmarkrun.LatencyStats {
	if h.total == 0 {
		return benchmarkrun.LatencyStats{}
	}

	return benchmarkrun.LatencyStats{
		MinMilliseconds:     durationMilliseconds(h.min),
		MaxMilliseconds:     durationMilliseconds(h.max),
		AverageMilliseconds: float64(h.sum) / float64(h.total) / float64(time.Millisecond),
		P50Milliseconds:     durationMilliseconds(h.quantile(0.50)),
		P90Milliseconds:     durationMilliseconds(h.quantile(0.90)),
		P95Milliseconds:     durationMilliseconds(h.quantile(0.95)),
		P99Milliseconds:     durationMilliseconds(h.quantile(0.99)),
	}
}

func (h latencyHistogram) quantile(percentile float64) time.Duration {
	if h.total == 0 {
		return 0
	}

	rank := uint64(math.Ceil(percentile * float64(h.total)))
	if rank == 0 {
		rank = 1
	}

	var cumulative uint64
	for index, count := range h.counts {
		cumulative += count
		if cumulative < rank {
			continue
		}
		if index < len(h.bounds) {
			return h.bounds[index]
		}
		return h.max
	}

	return h.max
}

func buildLatencyBucketBounds() []time.Duration {
	bounds := make([]time.Duration, 0, 321)
	for milliseconds := 1; milliseconds <= 100; milliseconds++ {
		bounds = append(bounds, time.Duration(milliseconds)*time.Millisecond)
	}
	for milliseconds := 105; milliseconds <= 500; milliseconds += 5 {
		bounds = append(bounds, time.Duration(milliseconds)*time.Millisecond)
	}
	for milliseconds := 510; milliseconds <= 1000; milliseconds += 10 {
		bounds = append(bounds, time.Duration(milliseconds)*time.Millisecond)
	}
	for milliseconds := 1100; milliseconds <= 10000; milliseconds += 100 {
		bounds = append(bounds, time.Duration(milliseconds)*time.Millisecond)
	}
	return bounds
}

func operationRatesSnapshot(counts [operationKindCount]uint64, elapsedSeconds float64) benchmarkrun.OperationRates {
	return benchmarkrun.OperationRates{
		PointRead:     ratePerSecond(counts[operationKindPointRead], elapsedSeconds),
		RangeRead:     ratePerSecond(counts[operationKindRangeRead], elapsedSeconds),
		HistoryInsert: ratePerSecond(counts[operationKindHistoryInsert], elapsedSeconds),
		AccountUpdate: ratePerSecond(counts[operationKindAccountUpdate], elapsedSeconds),
		Transaction:   ratePerSecond(counts[operationKindTransaction], elapsedSeconds),
	}
}

func ratePerSecond(count uint64, elapsedSeconds float64) float64 {
	if count == 0 || elapsedSeconds <= 0 {
		return 0
	}
	return float64(count) / elapsedSeconds
}

func durationMilliseconds(duration time.Duration) float64 {
	return float64(duration) / float64(time.Millisecond)
}

func compactErrorText(message string) string {
	compact := strings.Join(strings.Fields(message), " ")
	if len(compact) <= latestErrorLimit {
		return compact
	}
	return compact[:latestErrorLimit-3] + "..."
}
