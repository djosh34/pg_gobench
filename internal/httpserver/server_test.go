package httpserver_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pg_gobench/internal/benchmark"
	"pg_gobench/internal/benchmarkrun"
	"pg_gobench/internal/httpserver"
)

func TestNewServesBenchmarkStateAsJSON(t *testing.T) {
	controller := &fakeControl{
		state: runningState(),
	}
	server := newTestServer(controller, func(context.Context) error { return nil })

	response := serveRequest(t, server, http.MethodGet, "/benchmark", "")

	if response.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if got := response.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want %q", got, "application/json")
	}

	var state benchmarkrun.State
	decodeResponseJSON(t, response, &state)

	if state.Status != benchmarkrun.StatusRunning {
		t.Fatalf("Status = %q, want %q", state.Status, benchmarkrun.StatusRunning)
	}
	if state.Options.Scale != 12 {
		t.Fatalf("Options.Scale = %d, want %d", state.Options.Scale, 12)
	}
	if state.Options.ReadPercent == nil {
		t.Fatal("Options.ReadPercent = nil, want value")
	}
	if *state.Options.ReadPercent != 70 {
		t.Fatalf("Options.ReadPercent = %d, want %d", *state.Options.ReadPercent, 70)
	}
}

func TestNewStartsBenchmarkFromJSONRequest(t *testing.T) {
	controller := &fakeControl{
		startFn: func(_ context.Context, options benchmark.StartOptions) (benchmarkrun.State, error) {
			if options.Scale != 16 {
				t.Fatalf("Scale = %d, want %d", options.Scale, 16)
			}
			if options.Profile != benchmark.ProfileWrite {
				t.Fatalf("Profile = %q, want %q", options.Profile, benchmark.ProfileWrite)
			}
			return benchmarkrun.State{
				Status:  benchmarkrun.StatusRunning,
				Options: options,
			}, nil
		},
	}
	server := newTestServer(controller, func(context.Context) error { return nil })

	response := serveRequest(t, server, http.MethodPost, "/benchmark/start", `{"scale":16,"profile":"write"}`)

	if response.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", response.StatusCode, http.StatusOK)
	}

	var state benchmarkrun.State
	decodeResponseJSON(t, response, &state)

	if !controller.startCalled {
		t.Fatal("Start was not called")
	}
	if state.Status != benchmarkrun.StatusRunning {
		t.Fatalf("Status = %q, want %q", state.Status, benchmarkrun.StatusRunning)
	}
	if state.Options.Scale != 16 {
		t.Fatalf("Options.Scale = %d, want %d", state.Options.Scale, 16)
	}
}

func TestNewRejectsSecondStartWithConflictAndCompactErrorJSON(t *testing.T) {
	controller := &fakeControl{
		startFn: func(_ context.Context, options benchmark.StartOptions) (benchmarkrun.State, error) {
			return benchmarkrun.State{
				Status:  benchmarkrun.StatusRunning,
				Options: options,
			}, benchmarkrun.ErrRunActive
		},
	}
	server := newTestServer(controller, func(context.Context) error { return nil })

	response := serveRequest(t, server, http.MethodPost, "/benchmark/start", `{"scale":10}`)

	if response.StatusCode != http.StatusConflict {
		t.Fatalf("StatusCode = %d, want %d", response.StatusCode, http.StatusConflict)
	}

	var payload map[string]string
	decodeResponseJSON(t, response, &payload)
	if payload["error"] != benchmarkrun.ErrRunActive.Error() {
		t.Fatalf("error = %q, want %q", payload["error"], benchmarkrun.ErrRunActive.Error())
	}
}

func TestNewAltersBenchmarkAndReturnsUpdatedState(t *testing.T) {
	controller := &fakeControl{
		alterFn: func(options benchmark.AlterOptions) (benchmarkrun.State, error) {
			if options.Clients == nil {
				t.Fatal("Clients = nil, want value")
			}
			if *options.Clients != 8 {
				t.Fatalf("Clients = %d, want %d", *options.Clients, 8)
			}
			return benchmarkrun.State{
				Status: benchmarkrun.StatusRunning,
				Options: benchmark.StartOptions{
					Scale:           12,
					Clients:         8,
					DurationSeconds: 90,
					WarmupSeconds:   15,
					Profile:         benchmark.ProfileMixed,
					ReadPercent:     intPtr(70),
				},
			}, nil
		},
	}
	server := newTestServer(controller, func(context.Context) error { return nil })

	response := serveRequest(t, server, http.MethodPost, "/benchmark/alter", `{"clients":8}`)

	if response.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", response.StatusCode, http.StatusOK)
	}

	var state benchmarkrun.State
	decodeResponseJSON(t, response, &state)

	if !controller.alterCalled {
		t.Fatal("Alter was not called")
	}
	if state.Options.Clients != 8 {
		t.Fatalf("Options.Clients = %d, want %d", state.Options.Clients, 8)
	}
}

func TestNewRejectsAlterValidationErrorsAndStateConflicts(t *testing.T) {
	server := newTestServer(&fakeControl{}, func(context.Context) error { return nil })

	validationResponse := serveRequest(t, server, http.MethodPost, "/benchmark/alter", `{}`)
	if validationResponse.StatusCode != http.StatusBadRequest {
		t.Fatalf("validation StatusCode = %d, want %d", validationResponse.StatusCode, http.StatusBadRequest)
	}
	assertJSONErrorContains(t, validationResponse, "at least one field")

	conflictServer := newTestServer(&fakeControl{
		alterFn: func(benchmark.AlterOptions) (benchmarkrun.State, error) {
			return benchmarkrun.State{Status: benchmarkrun.StatusIdle}, benchmarkrun.ErrRunNotRunning
		},
	}, func(context.Context) error { return nil })

	conflictResponse := serveRequest(t, conflictServer, http.MethodPost, "/benchmark/alter", `{"clients":4}`)
	if conflictResponse.StatusCode != http.StatusConflict {
		t.Fatalf("conflict StatusCode = %d, want %d", conflictResponse.StatusCode, http.StatusConflict)
	}
	assertJSONErrorContains(t, conflictResponse, benchmarkrun.ErrRunNotRunning.Error())
}

func TestNewStopsBenchmarkAndReturnsState(t *testing.T) {
	controller := &fakeControl{
		stopFn: func() (benchmarkrun.State, error) {
			stoppedAt := time.Unix(1700000300, 0).UTC()
			return benchmarkrun.State{
				Status:    benchmarkrun.StatusStopped,
				Options:   runningState().Options,
				StoppedAt: &stoppedAt,
			}, nil
		},
	}
	server := newTestServer(controller, func(context.Context) error { return nil })

	response := serveRequest(t, server, http.MethodPost, "/benchmark/stop", "")

	if response.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", response.StatusCode, http.StatusOK)
	}

	var state benchmarkrun.State
	decodeResponseJSON(t, response, &state)

	if !controller.stopCalled {
		t.Fatal("Stop was not called")
	}
	if state.Status != benchmarkrun.StatusStopped {
		t.Fatalf("Status = %q, want %q", state.Status, benchmarkrun.StatusStopped)
	}
	if state.StoppedAt == nil {
		t.Fatal("StoppedAt = nil, want value")
	}
}

func TestNewServesBenchmarkResultsAsJSONSnapshot(t *testing.T) {
	controller := &fakeControl{
		results: runningResults(),
	}
	server := newTestServer(controller, func(context.Context) error { return nil })

	response := serveRequest(t, server, http.MethodGet, "/benchmark/results", "")

	if response.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", response.StatusCode, http.StatusOK)
	}

	var results benchmarkrun.Results
	decodeResponseJSON(t, response, &results)

	if results.Status != benchmarkrun.StatusRunning {
		t.Fatalf("Status = %q, want %q", results.Status, benchmarkrun.StatusRunning)
	}
	if results.Stats.ConfiguredClients != 4 {
		t.Fatalf("Stats.ConfiguredClients = %d, want %d", results.Stats.ConfiguredClients, 4)
	}
	if results.Stats.OperationRates.PointRead != 12.5 {
		t.Fatalf("Stats.OperationRates.PointRead = %v, want %v", results.Stats.OperationRates.PointRead, 12.5)
	}
}

func TestNewServesMetricsAsPrometheusTextWithoutAuthentication(t *testing.T) {
	controller := &fakeControl{
		metrics: benchmarkrun.MetricsSnapshot{
			RunActive:            true,
			RunDurationSeconds:   12.5,
			ConfiguredClients:    4,
			ActiveClients:        3,
			OperationsTotal:      99,
			OperationErrorsTotal: 2,
			TPS:                  7.92,
		},
	}
	server := newTestServer(controller, func(context.Context) error { return nil })

	response := serveRequest(t, server, http.MethodGet, "/metrics", "")

	if response.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if got := response.Header.Get("Content-Type"); got != "text/plain; version=0.0.4; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want %q", got, "text/plain; version=0.0.4; charset=utf-8")
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("Read body: %v", err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatalf("Close response body: %v", err)
	}

	bodyText := string(body)
	if !strings.Contains(bodyText, "pg_gobench_run_active 1\n") {
		t.Fatalf("body = %q, want pg_gobench_run_active metric", bodyText)
	}
	if !strings.Contains(bodyText, "pg_gobench_tps 7.92\n") {
		t.Fatalf("body = %q, want pg_gobench_tps metric", bodyText)
	}
}

func TestNewServesRequiredMetricsAndOnlyLowCardinalityHistogramLabels(t *testing.T) {
	controller := &fakeControl{
		metrics: benchmarkrun.MetricsSnapshot{
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
					{UpperBoundSeconds: 0.010, CumulativeCount: 3},
				},
				Count:      3,
				SumSeconds: 0.012,
			},
		},
	}
	server := newTestServer(controller, func(context.Context) error { return nil })

	response := serveRequest(t, server, http.MethodGet, "/metrics", "")

	if response.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", response.StatusCode, http.StatusOK)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("Read body: %v", err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatalf("Close response body: %v", err)
	}

	bodyText := string(body)
	requiredLines := []string{
		"pg_gobench_run_active 1",
		"pg_gobench_run_duration_seconds 12.5",
		"pg_gobench_configured_clients 4",
		"pg_gobench_active_clients 3",
		"pg_gobench_operations_total 99",
		"pg_gobench_operation_errors_total 2",
		"pg_gobench_tps 7.92",
		`pg_gobench_operation_latency_seconds_bucket{le="0.001"} 1`,
		`pg_gobench_operation_latency_seconds_bucket{le="0.01"} 3`,
		`pg_gobench_operation_latency_seconds_bucket{le="+Inf"} 3`,
		"pg_gobench_operation_latency_seconds_count 3",
		"pg_gobench_operation_latency_seconds_sum 0.012",
	}
	for _, want := range requiredLines {
		if !strings.Contains(bodyText, want+"\n") {
			t.Fatalf("body = %q, want line %q", bodyText, want)
		}
	}

	for _, line := range strings.Split(strings.TrimSpace(bodyText), "\n") {
		if !strings.Contains(line, "{") {
			continue
		}
		if !strings.Contains(line, `{le="`) {
			t.Fatalf("metric line %q has non-histogram labels", line)
		}
	}
	forbiddenSubstrings := []string{
		"select * from accounts",
		"db01.internal",
		"postgres://",
		"benchmark_id",
		"permission denied",
	}
	for _, forbidden := range forbiddenSubstrings {
		if strings.Contains(bodyText, forbidden) {
			t.Fatalf("body = %q, should not contain %q", bodyText, forbidden)
		}
	}
	if strings.Contains(bodyText, "latest_error") {
		t.Fatalf("body = %q, should not contain JSON-only latest_error field", bodyText)
	}
}

func TestNewServesHealthzAndReadyzAsJSON(t *testing.T) {
	server := newTestServer(&fakeControl{}, func(context.Context) error { return nil })

	healthz := serveRequest(t, server, http.MethodGet, "/healthz", "")
	if healthz.StatusCode != http.StatusOK {
		t.Fatalf("/healthz StatusCode = %d, want %d", healthz.StatusCode, http.StatusOK)
	}
	assertJSONStatusOK(t, healthz)

	readyz := serveRequest(t, server, http.MethodGet, "/readyz", "")
	if readyz.StatusCode != http.StatusOK {
		t.Fatalf("/readyz StatusCode = %d, want %d", readyz.StatusCode, http.StatusOK)
	}
	assertJSONStatusOK(t, readyz)
}

func TestNewReturnsReadyzFailureAsServiceUnavailableWithGoErrorText(t *testing.T) {
	server := newTestServer(&fakeControl{}, func(context.Context) error {
		return errors.New("dial tcp 127.0.0.1:5432: connect: connection refused")
	})

	response := serveRequest(t, server, http.MethodGet, "/readyz", "")

	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("StatusCode = %d, want %d", response.StatusCode, http.StatusServiceUnavailable)
	}
	assertJSONErrorContains(t, response, "connect: connection refused")
}

func TestNewRejectsMalformedUnknownAndTrailingJSONAndInvalidMethods(t *testing.T) {
	server := newTestServer(&fakeControl{}, func(context.Context) error { return nil })

	testCases := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
		wantError  string
	}{
		{
			name:       "start rejects malformed json",
			method:     http.MethodPost,
			path:       "/benchmark/start",
			body:       `{"scale":`,
			wantStatus: http.StatusBadRequest,
			wantError:  "decode JSON",
		},
		{
			name:       "start rejects unknown fields",
			method:     http.MethodPost,
			path:       "/benchmark/start",
			body:       `{"scale":12,"bogus":true}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "unknown field",
		},
		{
			name:       "start rejects trailing json",
			method:     http.MethodPost,
			path:       "/benchmark/start",
			body:       `{"scale":12}{"extra":true}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "unexpected trailing data",
		},
		{
			name:       "benchmark rejects invalid methods",
			method:     http.MethodDelete,
			path:       "/benchmark",
			wantStatus: http.StatusMethodNotAllowed,
			wantError:  "method DELETE not allowed",
		},
		{
			name:       "start rejects invalid methods",
			method:     http.MethodGet,
			path:       "/benchmark/start",
			wantStatus: http.StatusMethodNotAllowed,
			wantError:  "method GET not allowed",
		},
		{
			name:       "readyz rejects invalid methods",
			method:     http.MethodPost,
			path:       "/readyz",
			wantStatus: http.StatusMethodNotAllowed,
			wantError:  "method POST not allowed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			response := serveRequest(t, server, tc.method, tc.path, tc.body)
			if response.StatusCode != tc.wantStatus {
				t.Fatalf("StatusCode = %d, want %d", response.StatusCode, tc.wantStatus)
			}
			assertJSONErrorContains(t, response, tc.wantError)
		})
	}
}

func newTestServer(controller *fakeControl, ready func(context.Context) error) *http.Server {
	return httpserver.New("127.0.0.1:8080", httpserver.Dependencies{
		Benchmark: controller,
		Ready:     ready,
	})
}

func serveRequest(t *testing.T, server *http.Server, method, path, body string) *http.Response {
	t.Helper()

	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	server.Handler.ServeHTTP(recorder, request)

	return recorder.Result()
}

func decodeResponseJSON(t *testing.T, response *http.Response, target any) {
	t.Helper()
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Fatalf("Close response body: %v", err)
		}
	}()

	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		t.Fatalf("Decode response JSON: %v", err)
	}
}

func assertJSONStatusOK(t *testing.T, response *http.Response) {
	t.Helper()

	var payload map[string]string
	decodeResponseJSON(t, response, &payload)
	if payload["status"] != "ok" {
		t.Fatalf("status = %q, want %q", payload["status"], "ok")
	}
}

func assertJSONErrorContains(t *testing.T, response *http.Response, want string) {
	t.Helper()

	var payload map[string]string
	decodeResponseJSON(t, response, &payload)
	if !strings.Contains(payload["error"], want) {
		t.Fatalf("error = %q, want substring %q", payload["error"], want)
	}
}

func runningState() benchmarkrun.State {
	startedAt := time.Unix(1700000000, 0).UTC()
	return benchmarkrun.State{
		Status: benchmarkrun.StatusRunning,
		Options: benchmark.StartOptions{
			Scale:           12,
			Clients:         4,
			DurationSeconds: 90,
			WarmupSeconds:   15,
			Profile:         benchmark.ProfileMixed,
			ReadPercent:     intPtr(70),
			TargetTPS:       intPtr(220),
		},
		StartedAt: &startedAt,
	}
}

func runningResults() benchmarkrun.Results {
	state := runningState()
	return benchmarkrun.Results{
		Status:    state.Status,
		Options:   state.Options,
		StartedAt: state.StartedAt,
		Stats: benchmarkrun.Stats{
			ConfiguredClients: 4,
			OperationRates: benchmarkrun.OperationRates{
				PointRead: 12.5,
			},
		},
	}
}

type fakeControl struct {
	state       benchmarkrun.State
	results     benchmarkrun.Results
	metrics     benchmarkrun.MetricsSnapshot
	startCalled bool
	alterCalled bool
	stopCalled  bool
	startFn     func(context.Context, benchmark.StartOptions) (benchmarkrun.State, error)
	alterFn     func(benchmark.AlterOptions) (benchmarkrun.State, error)
	stopFn      func() (benchmarkrun.State, error)
}

func (f *fakeControl) Start(ctx context.Context, options benchmark.StartOptions) (benchmarkrun.State, error) {
	f.startCalled = true
	if f.startFn == nil {
		return benchmarkrun.State{}, nil
	}
	return f.startFn(ctx, options)
}

func (f *fakeControl) Alter(options benchmark.AlterOptions) (benchmarkrun.State, error) {
	f.alterCalled = true
	if f.alterFn == nil {
		return benchmarkrun.State{}, nil
	}
	return f.alterFn(options)
}

func (f *fakeControl) Results() benchmarkrun.Results {
	return f.results
}

func (f *fakeControl) Metrics() benchmarkrun.MetricsSnapshot {
	return f.metrics
}

func (f *fakeControl) Stop() (benchmarkrun.State, error) {
	f.stopCalled = true
	if f.stopFn == nil {
		return benchmarkrun.State{}, nil
	}
	return f.stopFn()
}

func (f *fakeControl) State() benchmarkrun.State {
	return f.state
}

func intPtr(value int) *int {
	return &value
}
