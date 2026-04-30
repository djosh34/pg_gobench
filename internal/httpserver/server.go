package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"pg_gobench/internal/benchmark"
	"pg_gobench/internal/benchmarkrun"
)

type Benchmark interface {
	Start(context.Context, benchmark.StartOptions) (benchmarkrun.State, error)
	Alter(benchmark.AlterOptions) (benchmarkrun.State, error)
	Stop() (benchmarkrun.State, error)
	State() benchmarkrun.State
	Results() benchmarkrun.Results
	Metrics() benchmarkrun.MetricsSnapshot
}

type Dependencies struct {
	Benchmark Benchmark
	Ready     func(context.Context) error
}

type handler struct {
	benchmark Benchmark
	ready     func(context.Context) error
}

type statusResponse struct {
	Status string `json:"status"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type routeSpec struct {
	path    string
	method  string
	handler http.HandlerFunc
}

func New(addr string, deps Dependencies) *http.Server {
	h := handler{
		benchmark: deps.Benchmark,
		ready:     deps.Ready,
	}

	mux := http.NewServeMux()
	routes := []routeSpec{
		{path: "/healthz", method: http.MethodGet, handler: h.handleHealthz},
		{path: "/readyz", method: http.MethodGet, handler: h.handleReadyz},
		{path: "/benchmark", method: http.MethodGet, handler: h.handleBenchmark},
		{path: "/benchmark/results", method: http.MethodGet, handler: h.handleBenchmarkResults},
		{path: "/benchmark/start", method: http.MethodPost, handler: h.handleBenchmarkStart},
		{path: "/benchmark/alter", method: http.MethodPost, handler: h.handleBenchmarkAlter},
		{path: "/benchmark/stop", method: http.MethodPost, handler: h.handleBenchmarkStop},
		{path: "/metrics", method: http.MethodGet, handler: h.handleMetrics},
	}
	for _, route := range routes {
		mux.HandleFunc(route.path, route.handler)
	}

	return &http.Server{
		Addr:    addr,
		Handler: newBrowserCompatibleHandler(mux, routes),
	}
}

func newBrowserCompatibleHandler(next http.Handler, routes []routeSpec) http.Handler {
	allowedMethods := make(map[string]string, len(routes))
	for _, route := range routes {
		allowedMethods[route.path] = route.method
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, ok := allowedMethods[r.URL.Path]
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		setBrowserAccessHeaders(w, method)
		if r.Method == http.MethodOptions {
			w.Header().Set("Allow", allowedMethodsList(method))
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (h handler) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if err := writeJSON(w, http.StatusOK, statusResponse{Status: "ok"}); err != nil {
		log.Printf("write /healthz response: %v", err)
	}
}

func (h handler) handleReadyz(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if h.ready == nil {
		if err := writeError(w, http.StatusInternalServerError, errors.New("readiness probe unavailable")); err != nil {
			log.Printf("write /readyz error response: %v", err)
		}
		return
	}
	if err := h.ready(r.Context()); err != nil {
		if writeErr := writeError(w, http.StatusServiceUnavailable, err); writeErr != nil {
			log.Printf("write /readyz failure response: %v", writeErr)
		}
		return
	}
	if err := writeJSON(w, http.StatusOK, statusResponse{Status: "ok"}); err != nil {
		log.Printf("write /readyz response: %v", err)
	}
}

func (h handler) handleBenchmark(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if h.benchmark == nil {
		if err := writeError(w, http.StatusInternalServerError, errors.New("benchmark controller unavailable")); err != nil {
			log.Printf("write /benchmark error response: %v", err)
		}
		return
	}
	if err := writeJSON(w, http.StatusOK, h.benchmark.State()); err != nil {
		log.Printf("write /benchmark response: %v", err)
	}
}

func (h handler) handleBenchmarkResults(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if h.benchmark == nil {
		if err := writeError(w, http.StatusInternalServerError, errors.New("benchmark controller unavailable")); err != nil {
			log.Printf("write /benchmark/results error response: %v", err)
		}
		return
	}
	if err := writeJSON(w, http.StatusOK, h.benchmark.Results()); err != nil {
		log.Printf("write /benchmark/results response: %v", err)
	}
}

func (h handler) handleBenchmarkStart(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if h.benchmark == nil {
		if err := writeError(w, http.StatusInternalServerError, errors.New("benchmark controller unavailable")); err != nil {
			log.Printf("write /benchmark/start error response: %v", err)
		}
		return
	}

	options, err := benchmark.DecodeStartOptions(r.Body)
	if err != nil {
		if writeErr := writeError(w, http.StatusBadRequest, err); writeErr != nil {
			log.Printf("write /benchmark/start decode error response: %v", writeErr)
		}
		return
	}

	state, err := h.benchmark.Start(r.Context(), options)
	if err != nil {
		if writeErr := writeError(w, statusForBenchmarkError(err), err); writeErr != nil {
			log.Printf("write /benchmark/start failure response: %v", writeErr)
		}
		return
	}

	if err := writeJSON(w, http.StatusOK, state); err != nil {
		log.Printf("write /benchmark/start response: %v", err)
	}
}

func (h handler) handleBenchmarkAlter(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if h.benchmark == nil {
		if err := writeError(w, http.StatusInternalServerError, errors.New("benchmark controller unavailable")); err != nil {
			log.Printf("write /benchmark/alter error response: %v", err)
		}
		return
	}

	options, err := benchmark.DecodeAlterOptions(r.Body)
	if err != nil {
		if writeErr := writeError(w, http.StatusBadRequest, err); writeErr != nil {
			log.Printf("write /benchmark/alter decode error response: %v", writeErr)
		}
		return
	}

	state, err := h.benchmark.Alter(options)
	if err != nil {
		if writeErr := writeError(w, statusForBenchmarkError(err), err); writeErr != nil {
			log.Printf("write /benchmark/alter failure response: %v", writeErr)
		}
		return
	}

	if err := writeJSON(w, http.StatusOK, state); err != nil {
		log.Printf("write /benchmark/alter response: %v", err)
	}
}

func (h handler) handleBenchmarkStop(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if h.benchmark == nil {
		if err := writeError(w, http.StatusInternalServerError, errors.New("benchmark controller unavailable")); err != nil {
			log.Printf("write /benchmark/stop error response: %v", err)
		}
		return
	}

	state, err := h.benchmark.Stop()
	if err != nil {
		if writeErr := writeError(w, statusForBenchmarkError(err), err); writeErr != nil {
			log.Printf("write /benchmark/stop failure response: %v", writeErr)
		}
		return
	}

	if err := writeJSON(w, http.StatusOK, state); err != nil {
		log.Printf("write /benchmark/stop response: %v", err)
	}
}

func (h handler) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if h.benchmark == nil {
		if err := writeError(w, http.StatusInternalServerError, errors.New("benchmark controller unavailable")); err != nil {
			log.Printf("write /metrics error response: %v", err)
		}
		return
	}

	w.Header().Set("Content-Type", benchmarkrun.PrometheusContentType())
	w.WriteHeader(http.StatusOK)
	if err := h.benchmark.Metrics().WritePrometheus(w); err != nil {
		log.Printf("write /metrics response: %v", err)
	}
}

func allowMethod(w http.ResponseWriter, r *http.Request, want string) bool {
	if r.Method == want {
		return true
	}

	w.Header().Set("Allow", want)
	if err := writeError(w, http.StatusMethodNotAllowed, fmt.Errorf("method %s not allowed", r.Method)); err != nil {
		log.Printf("write method-not-allowed response: %v", err)
	}
	return false
}

func statusForBenchmarkError(err error) int {
	switch {
	case errors.Is(err, benchmarkrun.ErrRunActive), errors.Is(err, benchmarkrun.ErrRunNotRunning):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func writeError(w http.ResponseWriter, status int, err error) error {
	return writeJSON(w, status, errorResponse{Error: err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, payload any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		return fmt.Errorf("encode JSON response: %w", err)
	}
	return nil
}

func setBrowserAccessHeaders(w http.ResponseWriter, method string) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", allowedMethodsList(method))
}

func allowedMethodsList(method string) string {
	return strings.Join([]string{method, http.MethodOptions}, ", ")
}
