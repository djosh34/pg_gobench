package benchmarkrun

import (
	"fmt"
	"io"
	"strconv"
)

const prometheusContentType = "text/plain; version=0.0.4; charset=utf-8"

type MetricsSnapshot struct {
	RunActive            bool
	RunDurationSeconds   float64
	ConfiguredClients    int
	ActiveClients        int
	OperationsTotal      uint64
	OperationErrorsTotal uint64
	TPS                  float64
	OperationLatency     LatencyHistogramSnapshot
}

type LatencyHistogramSnapshot struct {
	Buckets    []LatencyHistogramBucket
	Count      uint64
	SumSeconds float64
}

type LatencyHistogramBucket struct {
	UpperBoundSeconds float64
	CumulativeCount   uint64
}

func (m MetricsSnapshot) WritePrometheus(w io.Writer) error {
	if err := writePrometheusMetric(w, "pg_gobench_run_active", boolGaugeValue(m.RunActive)); err != nil {
		return err
	}
	if err := writePrometheusMetric(w, "pg_gobench_run_duration_seconds", m.RunDurationSeconds); err != nil {
		return err
	}
	if err := writePrometheusMetric(w, "pg_gobench_configured_clients", m.ConfiguredClients); err != nil {
		return err
	}
	if err := writePrometheusMetric(w, "pg_gobench_active_clients", m.ActiveClients); err != nil {
		return err
	}
	if err := writePrometheusMetric(w, "pg_gobench_operations_total", m.OperationsTotal); err != nil {
		return err
	}
	if err := writePrometheusMetric(w, "pg_gobench_operation_errors_total", m.OperationErrorsTotal); err != nil {
		return err
	}
	if err := writePrometheusMetric(w, "pg_gobench_tps", m.TPS); err != nil {
		return err
	}
	if err := writePrometheusHistogram(w, "pg_gobench_operation_latency_seconds", m.OperationLatency); err != nil {
		return err
	}
	return nil
}

func metricsFromResults(results Results) MetricsSnapshot {
	return MetricsSnapshot{
		RunActive:            results.Status == StatusStarting || results.Status == StatusRunning || results.Status == StatusStopping,
		RunDurationSeconds:   results.Stats.ElapsedSeconds,
		ConfiguredClients:    results.Stats.ConfiguredClients,
		ActiveClients:        results.Stats.ActiveClients,
		OperationsTotal:      results.Stats.TotalOperations,
		OperationErrorsTotal: results.Stats.FailedOperations,
		TPS:                  results.Stats.TPS,
	}
}

func PrometheusContentType() string {
	return prometheusContentType
}

func boolGaugeValue(value bool) int {
	if value {
		return 1
	}
	return 0
}

func writePrometheusMetric(w io.Writer, name string, value any) error {
	if _, err := fmt.Fprintf(w, "%s %v\n", name, value); err != nil {
		return fmt.Errorf("write %s metric: %w", name, err)
	}
	return nil
}

func writePrometheusHistogram(w io.Writer, name string, histogram LatencyHistogramSnapshot) error {
	for _, bucket := range histogram.Buckets {
		if _, err := fmt.Fprintf(
			w,
			"%s_bucket{le=%q} %d\n",
			name,
			prometheusFloat(bucket.UpperBoundSeconds),
			bucket.CumulativeCount,
		); err != nil {
			return fmt.Errorf("write %s bucket metric: %w", name, err)
		}
	}
	if _, err := fmt.Fprintf(w, "%s_bucket{le=%q} %d\n", name, "+Inf", histogram.Count); err != nil {
		return fmt.Errorf("write %s +Inf bucket metric: %w", name, err)
	}
	if err := writePrometheusMetric(w, name+"_count", histogram.Count); err != nil {
		return err
	}
	if err := writePrometheusMetric(w, name+"_sum", histogram.SumSeconds); err != nil {
		return err
	}
	return nil
}

func prometheusFloat(value float64) string {
	return strconv.FormatFloat(value, 'g', -1, 64)
}
