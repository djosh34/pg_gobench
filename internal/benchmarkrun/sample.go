package benchmarkrun

type Sample struct {
	Latency              LatencySample
	ElapsedSeconds       float64
	TotalOperations      uint64
	SuccessfulOperations uint64
	FailedOperations     uint64
	ActiveClients        int
	ConfiguredClients    int
	OperationCounts      OperationCounts
	LatestError          string
}

type LatencySample struct {
	MinMilliseconds     float64
	MaxMilliseconds     float64
	AverageMilliseconds float64
	P50Milliseconds     float64
	P90Milliseconds     float64
	P95Milliseconds     float64
	P99Milliseconds     float64
	Buckets             []LatencyHistogramBucket
	Count               uint64
	SumSeconds          float64
}

type OperationCounts struct {
	PointRead     uint64
	RangeRead     uint64
	HistoryInsert uint64
	AccountUpdate uint64
	Transaction   uint64
}

func (s Sample) Stats() Stats {
	return Stats{
		Latency: LatencyStats{
			MinMilliseconds:     s.Latency.MinMilliseconds,
			MaxMilliseconds:     s.Latency.MaxMilliseconds,
			AverageMilliseconds: s.Latency.AverageMilliseconds,
			P50Milliseconds:     s.Latency.P50Milliseconds,
			P90Milliseconds:     s.Latency.P90Milliseconds,
			P95Milliseconds:     s.Latency.P95Milliseconds,
			P99Milliseconds:     s.Latency.P99Milliseconds,
		},
		TPS:                  ratePerSecond(s.TotalOperations, s.ElapsedSeconds),
		TotalOperations:      s.TotalOperations,
		SuccessfulOperations: s.SuccessfulOperations,
		FailedOperations:     s.FailedOperations,
		ActiveClients:        s.ActiveClients,
		ConfiguredClients:    s.ConfiguredClients,
		ElapsedSeconds:       s.ElapsedSeconds,
		OperationRates: OperationRates{
			PointRead:     ratePerSecond(s.OperationCounts.PointRead, s.ElapsedSeconds),
			RangeRead:     ratePerSecond(s.OperationCounts.RangeRead, s.ElapsedSeconds),
			HistoryInsert: ratePerSecond(s.OperationCounts.HistoryInsert, s.ElapsedSeconds),
			AccountUpdate: ratePerSecond(s.OperationCounts.AccountUpdate, s.ElapsedSeconds),
			Transaction:   ratePerSecond(s.OperationCounts.Transaction, s.ElapsedSeconds),
		},
		LatestError: s.LatestError,
	}
}

func (s Sample) Metrics(runActive bool) MetricsSnapshot {
	return MetricsSnapshot{
		RunActive:            runActive,
		RunDurationSeconds:   s.ElapsedSeconds,
		ConfiguredClients:    s.ConfiguredClients,
		ActiveClients:        s.ActiveClients,
		OperationsTotal:      s.TotalOperations,
		OperationErrorsTotal: s.FailedOperations,
		TPS:                  ratePerSecond(s.TotalOperations, s.ElapsedSeconds),
		OperationLatency: LatencyHistogramSnapshot{
			Buckets:    append([]LatencyHistogramBucket(nil), s.Latency.Buckets...),
			Count:      s.Latency.Count,
			SumSeconds: s.Latency.SumSeconds,
		},
	}
}

func zeroSample() Sample {
	return Sample{}
}

func ratePerSecond(count uint64, elapsedSeconds float64) float64 {
	if count == 0 || elapsedSeconds <= 0 {
		return 0
	}
	return float64(count) / elapsedSeconds
}
