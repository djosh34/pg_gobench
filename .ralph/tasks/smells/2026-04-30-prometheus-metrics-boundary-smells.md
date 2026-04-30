## Smell Set: prometheus-metrics-boundary-smells <status>not_started</status> <passes>false</passes>

Please refer to skill 'improve-code-boundaries' to see what smells there are.

Inside dirs:
- `internal/httpserver`
- `internal/benchmarkrun`

Solve each smell:

---
- [ ] Smell 3, wrong place-ism
The current public benchmark snapshot exposed to transport is JSON-shaped `Results`/`Stats`, while the new task needs Prometheus histogram buckets and metric naming policy. If `/metrics` is added directly in `internal/httpserver`, the transport layer will become the courier for benchmark semantics, metric naming, histogram rendering rules, and label policy. Execution should move that knowledge into a typed metrics boundary owned by `internal/benchmarkrun`, with `internal/httpserver` staying thin.

code:
```go
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
```

```go
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
```
