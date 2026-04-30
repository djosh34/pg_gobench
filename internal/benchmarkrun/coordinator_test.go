package benchmarkrun_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"pg_gobench/internal/benchmark"
	"pg_gobench/internal/benchmarkrun"
)

func TestCoordinatorStartMovesIdleToRunningAndExposesOptions(t *testing.T) {
	startedAt := time.Date(2026, 4, 30, 9, 0, 0, 0, time.UTC)
	run := &fakeRun{waitResult: make(chan error, 1)}
	runner := &fakeRunner{run: run}
	coordinator := benchmarkrun.New(runner, benchmarkrun.WithNow(func() time.Time {
		return startedAt
	}))
	options := benchmark.StartOptions{
		Scale:           12,
		Clients:         4,
		DurationSeconds: 90,
		WarmupSeconds:   15,
		Profile:         benchmark.ProfileMixed,
		ReadPercent:     intPtr(70),
		TargetTPS:       intPtr(250),
	}

	initial := coordinator.State()
	if initial.Status != benchmarkrun.StatusIdle {
		t.Fatalf("initial Status = %q, want %q", initial.Status, benchmarkrun.StatusIdle)
	}

	state, err := coordinator.Start(context.Background(), options)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	if runner.startCalls != 1 {
		t.Fatalf("runner startCalls = %d, want %d", runner.startCalls, 1)
	}
	if !reflect.DeepEqual(runner.startedOptions, options) {
		t.Fatalf("runner startedOptions = %#v, want %#v", runner.startedOptions, options)
	}
	if state.Status != benchmarkrun.StatusRunning {
		t.Fatalf("Status = %q, want %q", state.Status, benchmarkrun.StatusRunning)
	}
	if !reflect.DeepEqual(state.Options, options) {
		t.Fatalf("Options = %#v, want %#v", state.Options, options)
	}
	if state.StartedAt == nil {
		t.Fatal("StartedAt = nil, want timestamp")
	}
	if !state.StartedAt.Equal(startedAt) {
		t.Fatalf("StartedAt = %v, want %v", state.StartedAt, startedAt)
	}
	if state.StoppedAt != nil {
		t.Fatalf("StoppedAt = %v, want nil", state.StoppedAt)
	}
	if state.Error != "" {
		t.Fatalf("Error = %q, want empty", state.Error)
	}
}

func TestCoordinatorRejectsSecondStartWhileRunIsActive(t *testing.T) {
	run := &fakeRun{waitResult: make(chan error, 1)}
	runner := &fakeRunner{run: run}
	coordinator := benchmarkrun.New(runner)
	firstOptions := benchmark.StartOptions{
		Scale:           10,
		Clients:         2,
		DurationSeconds: 60,
		WarmupSeconds:   10,
		Profile:         benchmark.ProfileRead,
	}
	secondOptions := benchmark.StartOptions{
		Scale:           20,
		Clients:         8,
		DurationSeconds: 120,
		WarmupSeconds:   20,
		Profile:         benchmark.ProfileWrite,
	}

	if _, err := coordinator.Start(context.Background(), firstOptions); err != nil {
		t.Fatalf("first Start returned error: %v", err)
	}

	state, err := coordinator.Start(context.Background(), secondOptions)
	if err == nil {
		t.Fatal("second Start returned nil error while first run was active")
	}
	if !errors.Is(err, benchmarkrun.ErrRunActive) {
		t.Fatalf("second Start error = %v, want %v", err, benchmarkrun.ErrRunActive)
	}
	if runner.startCalls != 1 {
		t.Fatalf("runner startCalls = %d, want %d", runner.startCalls, 1)
	}
	if state.Status != benchmarkrun.StatusRunning {
		t.Fatalf("Status = %q, want %q", state.Status, benchmarkrun.StatusRunning)
	}
	if !reflect.DeepEqual(state.Options, firstOptions) {
		t.Fatalf("Options = %#v, want %#v", state.Options, firstOptions)
	}
}

func TestCoordinatorStopCancelsRunAndEventuallyMarksItStopped(t *testing.T) {
	startedAt := time.Date(2026, 4, 30, 9, 0, 0, 0, time.UTC)
	stoppedAt := startedAt.Add(2 * time.Minute)
	run := &fakeRun{}
	runner := &fakeRunner{run: run}
	coordinator := benchmarkrun.New(runner, benchmarkrun.WithNow(sequenceNow(startedAt, stoppedAt)))
	options := benchmark.StartOptions{
		Scale:           10,
		Clients:         3,
		DurationSeconds: 60,
		WarmupSeconds:   10,
		Profile:         benchmark.ProfileMixed,
		ReadPercent:     intPtr(80),
	}

	run.waitFunc = func() error {
		<-runner.startedCtx.Done()
		return context.Canceled
	}

	if _, err := coordinator.Start(context.Background(), options); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	state, err := coordinator.Stop()
	if err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	if state.Status != benchmarkrun.StatusStopping {
		t.Fatalf("Stop Status = %q, want %q", state.Status, benchmarkrun.StatusStopping)
	}

	stoppedState := waitForState(t, coordinator, benchmarkrun.StatusStopped)
	if stoppedState.StoppedAt == nil {
		t.Fatal("StoppedAt = nil, want timestamp")
	}
	if !stoppedState.StoppedAt.Equal(stoppedAt) {
		t.Fatalf("StoppedAt = %v, want %v", stoppedState.StoppedAt, stoppedAt)
	}
	if stoppedState.Error != "" {
		t.Fatalf("Error = %q, want empty", stoppedState.Error)
	}
}

func TestCoordinatorStopIsIdempotentWhileStoppingAndAfterStopped(t *testing.T) {
	releaseWait := make(chan struct{})
	run := &fakeRun{}
	runner := &fakeRunner{run: run}
	coordinator := benchmarkrun.New(runner)

	run.waitFunc = func() error {
		<-runner.startedCtx.Done()
		<-releaseWait
		return context.Canceled
	}

	if _, err := coordinator.Start(context.Background(), benchmark.StartOptions{
		Scale:           10,
		Clients:         2,
		DurationSeconds: 60,
		WarmupSeconds:   10,
		Profile:         benchmark.ProfileRead,
	}); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	firstStop, err := coordinator.Stop()
	if err != nil {
		t.Fatalf("first Stop returned error: %v", err)
	}
	if firstStop.Status != benchmarkrun.StatusStopping {
		t.Fatalf("first Stop Status = %q, want %q", firstStop.Status, benchmarkrun.StatusStopping)
	}

	secondStop, err := coordinator.Stop()
	if err != nil {
		t.Fatalf("second Stop returned error: %v", err)
	}
	if secondStop.Status != benchmarkrun.StatusStopping {
		t.Fatalf("second Stop Status = %q, want %q", secondStop.Status, benchmarkrun.StatusStopping)
	}

	close(releaseWait)

	stoppedState := waitForState(t, coordinator, benchmarkrun.StatusStopped)
	thirdStop, err := coordinator.Stop()
	if err != nil {
		t.Fatalf("third Stop returned error: %v", err)
	}
	if thirdStop.Status != benchmarkrun.StatusStopped {
		t.Fatalf("third Stop Status = %q, want %q", thirdStop.Status, benchmarkrun.StatusStopped)
	}
	if !reflect.DeepEqual(thirdStop, stoppedState) {
		t.Fatalf("third Stop state = %#v, want %#v", thirdStop, stoppedState)
	}
}

func TestCoordinatorAlterWhileRunningUpdatesStateAndForwardsToRun(t *testing.T) {
	run := &fakeRun{waitResult: make(chan error, 1)}
	runner := &fakeRunner{run: run}
	coordinator := benchmarkrun.New(runner)
	startOptions := benchmark.StartOptions{
		Scale:           10,
		Clients:         4,
		DurationSeconds: 90,
		WarmupSeconds:   15,
		Profile:         benchmark.ProfileMixed,
		ReadPercent:     intPtr(80),
		TargetTPS:       intPtr(200),
	}
	alterOptions := benchmark.AlterOptions{
		Clients:   intPtr(6),
		TargetTPS: intPtr(300),
	}

	if _, err := coordinator.Start(context.Background(), startOptions); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	state, err := coordinator.Alter(alterOptions)
	if err != nil {
		t.Fatalf("Alter returned error: %v", err)
	}

	if run.alterCalls != 1 {
		t.Fatalf("run alterCalls = %d, want %d", run.alterCalls, 1)
	}
	if !reflect.DeepEqual(run.alteredOptions, alterOptions) {
		t.Fatalf("run alteredOptions = %#v, want %#v", run.alteredOptions, alterOptions)
	}
	if state.Status != benchmarkrun.StatusRunning {
		t.Fatalf("Status = %q, want %q", state.Status, benchmarkrun.StatusRunning)
	}
	if state.Options.Clients != 6 {
		t.Fatalf("Clients = %d, want %d", state.Options.Clients, 6)
	}
	if state.Options.TargetTPS == nil {
		t.Fatal("TargetTPS = nil, want value")
	}
	if *state.Options.TargetTPS != 300 {
		t.Fatalf("TargetTPS = %d, want %d", *state.Options.TargetTPS, 300)
	}
}

func TestCoordinatorAlterRejectsUnsafeRuntimeChange(t *testing.T) {
	run := &fakeRun{waitResult: make(chan error, 1)}
	runner := &fakeRunner{run: run}
	coordinator := benchmarkrun.New(runner)

	if _, err := coordinator.Start(context.Background(), benchmark.StartOptions{
		Scale:           10,
		Clients:         2,
		DurationSeconds: 60,
		WarmupSeconds:   10,
		Profile:         benchmark.ProfileLock,
	}); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	state, err := coordinator.Alter(benchmark.AlterOptions{Clients: intPtr(1)})
	if err == nil {
		t.Fatal("Alter returned nil error for unsafe runtime change")
	}
	if run.alterCalls != 0 {
		t.Fatalf("run alterCalls = %d, want %d", run.alterCalls, 0)
	}
	if state.Options.Clients != 2 {
		t.Fatalf("Clients = %d, want %d", state.Options.Clients, 2)
	}
}

func TestCoordinatorAlterRejectsWhenRunIsNotActive(t *testing.T) {
	coordinator := benchmarkrun.New(&fakeRunner{})

	state, err := coordinator.Alter(benchmark.AlterOptions{Clients: intPtr(4)})
	if err == nil {
		t.Fatal("Alter returned nil error without active run")
	}
	if !errors.Is(err, benchmarkrun.ErrRunNotRunning) {
		t.Fatalf("Alter error = %v, want %v", err, benchmarkrun.ErrRunNotRunning)
	}
	if state.Status != benchmarkrun.StatusIdle {
		t.Fatalf("Status = %q, want %q", state.Status, benchmarkrun.StatusIdle)
	}
}

func TestCoordinatorMarksWorkerFailureAsFailedAndExposesErrorInJSONState(t *testing.T) {
	run := &fakeRun{waitResult: make(chan error, 1)}
	runner := &fakeRunner{run: run}
	coordinator := benchmarkrun.New(runner)
	workerErr := errors.New("worker exploded")

	if _, err := coordinator.Start(context.Background(), benchmark.StartOptions{
		Scale:           10,
		Clients:         4,
		DurationSeconds: 60,
		WarmupSeconds:   10,
		Profile:         benchmark.ProfileWrite,
	}); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	run.waitResult <- workerErr

	state := waitForState(t, coordinator, benchmarkrun.StatusFailed)
	if state.Error != workerErr.Error() {
		t.Fatalf("Error = %q, want %q", state.Error, workerErr.Error())
	}
	encoded, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal state: %v", err)
	}
	if !strings.Contains(string(encoded), `"error":"worker exploded"`) {
		t.Fatalf("state JSON = %s, want error field", string(encoded))
	}
}

func TestCoordinatorMarksStartFailureAsFailedAndExposesSetupError(t *testing.T) {
	coordinator := benchmarkrun.New(&fakeRunner{
		startErr: errors.New("setup benchmark schema: relation pg_gobench.accounts already exists"),
	})

	state, err := coordinator.Start(context.Background(), benchmark.StartOptions{
		Scale:           10,
		Clients:         2,
		DurationSeconds: 60,
		WarmupSeconds:   10,
		Profile:         benchmark.ProfileRead,
	})
	if err == nil {
		t.Fatal("Start returned nil error for setup failure")
	}
	if !strings.Contains(err.Error(), "setup benchmark schema") {
		t.Fatalf("Start error = %q, want setup context", err)
	}
	if state.Status != benchmarkrun.StatusFailed {
		t.Fatalf("Status = %q, want %q", state.Status, benchmarkrun.StatusFailed)
	}
	if state.Error != err.Error() {
		t.Fatalf("Error = %q, want %q", state.Error, err.Error())
	}
}

func TestCoordinatorResultsExposeLiveSnapshotAndRetainFinishedStats(t *testing.T) {
	run := &fakeRun{
		waitResult: make(chan error, 1),
		snapshot: benchmarkrun.Stats{
			TotalOperations:      7,
			SuccessfulOperations: 6,
			FailedOperations:     1,
			ConfiguredClients:    4,
			OperationRates: benchmarkrun.OperationRates{
				PointRead: 3.5,
			},
			LatestError: "worker failed compactly",
		},
	}
	coordinator := benchmarkrun.New(&fakeRunner{run: run})

	if _, err := coordinator.Start(context.Background(), benchmark.StartOptions{
		Scale:           10,
		Clients:         4,
		DurationSeconds: 60,
		WarmupSeconds:   10,
		Profile:         benchmark.ProfileMixed,
	}); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	live := coordinator.Results()
	if live.Stats.TotalOperations != 7 {
		t.Fatalf("live TotalOperations = %d, want %d", live.Stats.TotalOperations, 7)
	}
	if live.Stats.OperationRates.PointRead != 3.5 {
		t.Fatalf("live PointRead rate = %v, want %v", live.Stats.OperationRates.PointRead, 3.5)
	}

	run.waitResult <- nil
	waitForState(t, coordinator, benchmarkrun.StatusStopped)

	finished := coordinator.Results()
	if finished.Stats.TotalOperations != 7 {
		t.Fatalf("finished TotalOperations = %d, want %d", finished.Stats.TotalOperations, 7)
	}
	if finished.Stats.LatestError != "worker failed compactly" {
		t.Fatalf("finished LatestError = %q, want snapshot value", finished.Stats.LatestError)
	}
}

func TestResultsJSONShapeIsStableAcrossProfiles(t *testing.T) {
	profiles := []benchmark.Profile{
		benchmark.ProfileRead,
		benchmark.ProfileWrite,
		benchmark.ProfileMixed,
		benchmark.ProfileTransaction,
	}

	var wantStatsKeys []string
	var wantOperationRateKeys []string

	for _, profile := range profiles {
		payload := benchmarkrun.Results{
			Status: benchmarkrun.StatusRunning,
			Options: benchmark.StartOptions{
				Scale:           10,
				Clients:         2,
				DurationSeconds: 60,
				WarmupSeconds:   10,
				Profile:         profile,
			},
			Stats: benchmarkrun.Stats{},
		}

		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("Marshal results for profile %q: %v", profile, err)
		}

		var decoded map[string]any
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			t.Fatalf("Unmarshal results for profile %q: %v", profile, err)
		}

		statsKeys := sortedKeys(decoded["stats"].(map[string]any))
		operationRateKeys := sortedKeys(decoded["stats"].(map[string]any)["operation_rates"].(map[string]any))

		if wantStatsKeys == nil {
			wantStatsKeys = statsKeys
			wantOperationRateKeys = operationRateKeys
			continue
		}
		if !reflect.DeepEqual(statsKeys, wantStatsKeys) {
			t.Fatalf("stats keys for profile %q = %v, want %v", profile, statsKeys, wantStatsKeys)
		}
		if !reflect.DeepEqual(operationRateKeys, wantOperationRateKeys) {
			t.Fatalf("operation_rates keys for profile %q = %v, want %v", profile, operationRateKeys, wantOperationRateKeys)
		}
	}
}

type fakeRunner struct {
	run            benchmarkrun.Run
	startErr       error
	startCalls     int
	startedOptions benchmark.StartOptions
	startedCtx     context.Context
}

func (f *fakeRunner) Start(ctx context.Context, options benchmark.StartOptions) (benchmarkrun.Run, error) {
	f.startCalls++
	f.startedOptions = options
	f.startedCtx = ctx
	return f.run, f.startErr
}

type fakeRun struct {
	waitResult     chan error
	waitFunc       func() error
	alterErr       error
	alterCalls     int
	alteredOptions benchmark.AlterOptions
	snapshot       benchmarkrun.Stats
}

func (f *fakeRun) Alter(options benchmark.AlterOptions) error {
	f.alterCalls++
	f.alteredOptions = options
	return f.alterErr
}

func (f *fakeRun) Wait() error {
	if f.waitFunc != nil {
		return f.waitFunc()
	}
	return <-f.waitResult
}

func (f *fakeRun) Snapshot() benchmarkrun.Stats {
	return f.snapshot
}

func intPtr(value int) *int {
	return &value
}

func sequenceNow(values ...time.Time) func() time.Time {
	index := 0

	return func() time.Time {
		if index >= len(values) {
			return values[len(values)-1]
		}
		value := values[index]
		index++
		return value
	}
}

func waitForState(t *testing.T, coordinator *benchmarkrun.Coordinator, want benchmarkrun.Status) benchmarkrun.State {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		state := coordinator.State()
		if state.Status == want {
			return state
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("state never reached %q; last state = %#v", want, coordinator.State())
	return benchmarkrun.State{}
}

func sortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
