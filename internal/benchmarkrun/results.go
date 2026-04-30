package benchmarkrun

import (
	"time"

	"pg_gobench/internal/benchmark"
)

type Results struct {
	Status    Status                 `json:"status"`
	Options   benchmark.StartOptions `json:"options"`
	StartedAt *time.Time             `json:"started_at,omitempty"`
	StoppedAt *time.Time             `json:"stopped_at,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Stats     Stats                  `json:"stats"`
}

type Stats struct {
	Latency              LatencyStats   `json:"latency"`
	TPS                  float64        `json:"tps"`
	TotalOperations      uint64         `json:"total_operations"`
	SuccessfulOperations uint64         `json:"successful_operations"`
	FailedOperations     uint64         `json:"failed_operations"`
	ActiveClients        int            `json:"active_clients"`
	ConfiguredClients    int            `json:"configured_clients"`
	ElapsedSeconds       float64        `json:"elapsed_seconds"`
	OperationRates       OperationRates `json:"operation_rates"`
	LatestError          string         `json:"latest_error,omitempty"`
}

type LatencyStats struct {
	MinMilliseconds     float64 `json:"min_ms"`
	MaxMilliseconds     float64 `json:"max_ms"`
	AverageMilliseconds float64 `json:"avg_ms"`
	P50Milliseconds     float64 `json:"p50_ms"`
	P90Milliseconds     float64 `json:"p90_ms"`
	P95Milliseconds     float64 `json:"p95_ms"`
	P99Milliseconds     float64 `json:"p99_ms"`
}

type OperationRates struct {
	PointRead     float64 `json:"point_read"`
	RangeRead     float64 `json:"range_read"`
	HistoryInsert float64 `json:"history_insert"`
	AccountUpdate float64 `json:"account_update"`
	Transaction   float64 `json:"transaction"`
}

func zeroStats() Stats {
	return Stats{
		Latency:        LatencyStats{},
		OperationRates: OperationRates{},
	}
}

func stateToResults(state State, stats Stats) Results {
	return Results{
		Status:    state.Status,
		Options:   cloneStartOptions(state.Options),
		StartedAt: cloneTimePtr(state.StartedAt),
		StoppedAt: cloneTimePtr(state.StoppedAt),
		Error:     state.Error,
		Stats:     stats,
	}
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	return timePtr(*value)
}
