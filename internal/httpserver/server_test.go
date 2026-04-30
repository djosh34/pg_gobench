package httpserver_test

import (
	"context"
	"encoding/json"
	"errors"
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
		state: runningState(),
	}
	server := newTestServer(controller, func(context.Context) error { return nil })

	response := serveRequest(t, server, http.MethodGet, "/benchmark/results", "")

	if response.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", response.StatusCode, http.StatusOK)
	}

	var state benchmarkrun.State
	decodeResponseJSON(t, response, &state)

	if state.Status != benchmarkrun.StatusRunning {
		t.Fatalf("Status = %q, want %q", state.Status, benchmarkrun.StatusRunning)
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

type fakeControl struct {
	state       benchmarkrun.State
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
