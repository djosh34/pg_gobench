package benchrunner

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"pg_gobench/internal/benchmark"
	"pg_gobench/internal/benchmarkrun"
)

func TestRunnerDurationStopsRunUsingDeterministicClock(t *testing.T) {
	durationCh := make(chan time.Time, 1)
	pace := newBlockingPaceGate()
	workload := &recordingWorkload{
		started: make(chan int, 4),
	}
	fakeClk := fakeClock{
		afterCh: durationCh,
		now:     time.Date(2026, 4, 30, 9, 0, 0, 0, time.UTC),
	}

	db, _ := openDriverDB(t, testDriverConfig{})
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Close db: %v", err)
		}
	})

	run, err := runner{
		db:    db,
		clock: fakeClk,
		newPlan: func(benchmark.StartOptions, benchmark.ScaleModel) (workloadPlan, error) {
			return workload, nil
		},
		newPaceGate: func(_ *int, _ clock) pacingGate {
			return pace
		},
	}.Start(context.Background(), benchmark.StartOptions{
		Scale:           1,
		Clients:         1,
		DurationSeconds: 60,
		WarmupSeconds:   10,
		Profile:         benchmark.ProfileRead,
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	pace.release()
	waitForWorkerStart(t, workload.started, 0)

	durationCh <- fakeClk.now.Add(60 * time.Second)

	if waitErr := run.Wait(); waitErr != nil {
		t.Fatalf("Wait error = %v, want nil", waitErr)
	}
}

func TestRunnerAlterAdjustsClientCountAndTargetTPS(t *testing.T) {
	workload := &recordingWorkload{
		started: make(chan int, 8),
		release: make(chan struct{}),
	}
	pace := newBlockingPaceGate()

	db, _ := openDriverDB(t, testDriverConfig{})
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Close db: %v", err)
		}
	})

	parent, cancel := context.WithCancel(context.Background())
	defer cancel()

	run, err := runner{
		db:    db,
		clock: fakeClock{afterCh: make(chan time.Time), now: time.Date(2026, 4, 30, 9, 0, 0, 0, time.UTC)},
		newPlan: func(benchmark.StartOptions, benchmark.ScaleModel) (workloadPlan, error) {
			return workload, nil
		},
		newPaceGate: func(_ *int, _ clock) pacingGate {
			return pace
		},
	}.Start(parent, benchmark.StartOptions{
		Scale:           1,
		Clients:         1,
		DurationSeconds: 600,
		WarmupSeconds:   10,
		Profile:         benchmark.ProfileRead,
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	pace.release()
	waitForWorkerStart(t, workload.started, 0)

	targetTPS := 200
	clientCount := 3
	if err := run.Alter(benchmark.AlterOptions{
		Clients:   &clientCount,
		TargetTPS: &targetTPS,
	}); err != nil {
		t.Fatalf("Alter returned error: %v", err)
	}

	pace.release()
	pace.release()
	seen := map[int]bool{0: true}
	for len(seen) < 3 {
		seen[waitForAnyWorkerStart(t, workload.started)] = true
	}

	if pace.updatedTargetTPS != targetTPS {
		t.Fatalf("pace updatedTargetTPS = %d, want %d", pace.updatedTargetTPS, targetTPS)
	}

	close(workload.release)
	cancel()

	if waitErr := run.Wait(); waitErr != context.Canceled {
		t.Fatalf("Wait error = %v, want %v", waitErr, context.Canceled)
	}
}

func TestRunnerPacingBlocksOperationIssuanceUntilTokensAreReleased(t *testing.T) {
	workload := &recordingWorkload{
		started: make(chan int, 4),
	}
	pace := newBlockingPaceGate()

	db, _ := openDriverDB(t, testDriverConfig{})
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Close db: %v", err)
		}
	})

	parent, cancel := context.WithCancel(context.Background())
	defer cancel()

	run, err := runner{
		db:    db,
		clock: fakeClock{afterCh: make(chan time.Time), now: time.Date(2026, 4, 30, 9, 0, 0, 0, time.UTC)},
		newPlan: func(benchmark.StartOptions, benchmark.ScaleModel) (workloadPlan, error) {
			return workload, nil
		},
		newPaceGate: func(_ *int, _ clock) pacingGate {
			return pace
		},
	}.Start(parent, benchmark.StartOptions{
		Scale:           1,
		Clients:         1,
		DurationSeconds: 600,
		WarmupSeconds:   10,
		Profile:         benchmark.ProfileRead,
		TargetTPS:       intPtr(50),
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	select {
	case workerID := <-workload.started:
		t.Fatalf("worker %d started before pace gate released a token", workerID)
	case <-time.After(100 * time.Millisecond):
	}

	pace.release()
	waitForWorkerStart(t, workload.started, 0)

	cancel()
	if waitErr := run.Wait(); waitErr != context.Canceled {
		t.Fatalf("Wait error = %v, want %v", waitErr, context.Canceled)
	}
}

func TestActiveRunSnapshotTracksMeasuredTPSAndElapsedAfterWarmup(t *testing.T) {
	clk := newMutableClock(time.Date(2026, 4, 30, 9, 0, 0, 0, time.UTC))
	workload := &timedWorkload{
		completed: make(chan int, 8),
		durations: []time.Duration{
			5 * time.Second,
			5 * time.Second,
			5 * time.Second,
		},
		kind: operationKindPointRead,
	}
	pace := newBlockingPaceGate()

	db, _ := openDriverDB(t, testDriverConfig{})
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Close db: %v", err)
		}
	})

	parent, cancel := context.WithCancel(context.Background())
	defer cancel()

	run, err := runner{
		db:    db,
		clock: clk,
		newPlan: func(benchmark.StartOptions, benchmark.ScaleModel) (workloadPlan, error) {
			return workload, nil
		},
		newPaceGate: func(_ *int, _ clock) pacingGate {
			return pace
		},
	}.Start(parent, benchmark.StartOptions{
		Scale:           1,
		Clients:         1,
		DurationSeconds: 600,
		WarmupSeconds:   10,
		Profile:         benchmark.ProfileRead,
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	pace.release()
	pace.release()
	pace.release()
	waitForOperationCompletions(t, workload.completed, 3)
	clk.Advance(5 * time.Second)

	snapshot := run.Snapshot()

	if snapshot.TotalOperations != 2 {
		t.Fatalf("TotalOperations = %d, want %d", snapshot.TotalOperations, 2)
	}
	if snapshot.ElapsedSeconds != 10 {
		t.Fatalf("ElapsedSeconds = %v, want %v", snapshot.ElapsedSeconds, 10.0)
	}
	if snapshot.TPS != 0.2 {
		t.Fatalf("TPS = %v, want %v", snapshot.TPS, 0.2)
	}
	if snapshot.OperationRates.PointRead != 0.2 {
		t.Fatalf("OperationRates.PointRead = %v, want %v", snapshot.OperationRates.PointRead, 0.2)
	}

	cancel()
	if waitErr := run.Wait(); waitErr != context.Canceled {
		t.Fatalf("Wait error = %v, want %v", waitErr, context.Canceled)
	}
}

func TestActiveRunSnapshotTracksSuccessFailureCountsClientCountsAndCompactLatestError(t *testing.T) {
	clk := newMutableClock(time.Date(2026, 4, 30, 9, 0, 0, 0, time.UTC))
	workload := &timedWorkload{
		completed: make(chan int, 8),
		durations: []time.Duration{
			2 * time.Second,
			2 * time.Second,
		},
		kind:       operationKindAccountUpdate,
		failAtCall: 2,
		err: errors.New(`update account failed:
			deadlock detected while
			writing row`),
	}
	pace := newBlockingPaceGate()

	db, _ := openDriverDB(t, testDriverConfig{})
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Close db: %v", err)
		}
	})

	parent, cancel := context.WithCancel(context.Background())
	defer cancel()

	run, err := runner{
		db:    db,
		clock: clk,
		newPlan: func(benchmark.StartOptions, benchmark.ScaleModel) (workloadPlan, error) {
			return workload, nil
		},
		newPaceGate: func(_ *int, _ clock) pacingGate {
			return pace
		},
	}.Start(parent, benchmark.StartOptions{
		Scale:           1,
		Clients:         1,
		DurationSeconds: 600,
		WarmupSeconds:   0,
		Profile:         benchmark.ProfileWrite,
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	pace.release()
	waitForOperationCompletions(t, workload.completed, 1)
	activeSnapshot := run.Snapshot()
	if activeSnapshot.ActiveClients != 1 {
		t.Fatalf("ActiveClients = %d, want %d", activeSnapshot.ActiveClients, 1)
	}
	if activeSnapshot.ConfiguredClients != 1 {
		t.Fatalf("ConfiguredClients = %d, want %d", activeSnapshot.ConfiguredClients, 1)
	}

	pace.release()
	waitForOperationCompletions(t, workload.completed, 1)
	waitErr := run.Wait()
	if waitErr == nil {
		t.Fatal("Wait returned nil error for failed operation")
	}

	snapshot := run.Snapshot()
	if snapshot.TotalOperations != 2 {
		t.Fatalf("TotalOperations = %d, want %d", snapshot.TotalOperations, 2)
	}
	if snapshot.SuccessfulOperations != 1 {
		t.Fatalf("SuccessfulOperations = %d, want %d", snapshot.SuccessfulOperations, 1)
	}
	if snapshot.FailedOperations != 1 {
		t.Fatalf("FailedOperations = %d, want %d", snapshot.FailedOperations, 1)
	}
	if snapshot.ActiveClients != 0 {
		t.Fatalf("ActiveClients = %d, want %d", snapshot.ActiveClients, 0)
	}
	if snapshot.ConfiguredClients != 1 {
		t.Fatalf("ConfiguredClients = %d, want %d", snapshot.ConfiguredClients, 1)
	}
	if snapshot.OperationRates.AccountUpdate != 0.5 {
		t.Fatalf("OperationRates.AccountUpdate = %v, want %v", snapshot.OperationRates.AccountUpdate, 0.5)
	}
	if snapshot.LatestError != "update account failed: deadlock detected while writing row" {
		t.Fatalf("LatestError = %q, want compact error text", snapshot.LatestError)
	}
}

func TestActiveRunSnapshotTracksConfiguredClientsAcrossAlterations(t *testing.T) {
	workload := &recordingWorkload{
		started: make(chan int, 16),
	}
	pace := newBlockingPaceGate()

	db, _ := openDriverDB(t, testDriverConfig{})
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Close db: %v", err)
		}
	})

	parent, cancel := context.WithCancel(context.Background())
	defer cancel()

	run, err := runner{
		db:    db,
		clock: newMutableClock(time.Date(2026, 4, 30, 9, 0, 0, 0, time.UTC)),
		newPlan: func(benchmark.StartOptions, benchmark.ScaleModel) (workloadPlan, error) {
			return workload, nil
		},
		newPaceGate: func(_ *int, _ clock) pacingGate {
			return pace
		},
	}.Start(parent, benchmark.StartOptions{
		Scale:           1,
		Clients:         1,
		DurationSeconds: 600,
		WarmupSeconds:   0,
		Profile:         benchmark.ProfileRead,
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	pace.release()
	waitForWorkerStart(t, workload.started, 0)

	targetClients := 3
	if err := run.Alter(benchmark.AlterOptions{Clients: &targetClients}); err != nil {
		t.Fatalf("Alter up returned error: %v", err)
	}

	pace.release()
	seen := map[int]bool{0: true}
	waitForUniqueWorkers(t, pace, workload.started, seen, 3)

	snapshot := run.Snapshot()
	if snapshot.ActiveClients != 3 {
		t.Fatalf("ActiveClients = %d, want %d", snapshot.ActiveClients, 3)
	}
	if snapshot.ConfiguredClients != 3 {
		t.Fatalf("ConfiguredClients = %d, want %d", snapshot.ConfiguredClients, 3)
	}

	targetClients = 1
	if err := run.Alter(benchmark.AlterOptions{Clients: &targetClients}); err != nil {
		t.Fatalf("Alter down returned error: %v", err)
	}

	waitForActiveClients(t, run, 1)
	reduced := run.Snapshot()
	if reduced.ConfiguredClients != 1 {
		t.Fatalf("ConfiguredClients = %d, want %d", reduced.ConfiguredClients, 1)
	}

	cancel()
	if waitErr := run.Wait(); waitErr != context.Canceled {
		t.Fatalf("Wait error = %v, want %v", waitErr, context.Canceled)
	}
}

type fakeClock struct {
	afterCh <-chan time.Time
	now     time.Time
}

func (c fakeClock) After(time.Duration) <-chan time.Time {
	return c.afterCh
}

func (c fakeClock) NewTicker(time.Duration) pacingTicker {
	return fakeTicker{ch: make(chan time.Time)}
}

func (c fakeClock) Now() time.Time {
	return c.now
}

type mutableClock struct {
	mu      sync.Mutex
	now     time.Time
	afterCh chan time.Time
}

func newMutableClock(now time.Time) *mutableClock {
	return &mutableClock{
		now:     now,
		afterCh: make(chan time.Time),
	}
}

func (c *mutableClock) After(time.Duration) <-chan time.Time {
	return c.afterCh
}

func (c *mutableClock) NewTicker(time.Duration) pacingTicker {
	return fakeTicker{ch: make(chan time.Time)}
}

func (c *mutableClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *mutableClock) Advance(duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(duration)
}

type fakeTicker struct {
	ch chan time.Time
}

func (t fakeTicker) C() <-chan time.Time {
	return t.ch
}

func (fakeTicker) Stop() {}

type recordingWorkload struct {
	started chan int
	release chan struct{}
	kind    operationKind
}

func (w *recordingWorkload) RunOnce(_ context.Context, session *workerSession) (operationKind, error) {
	w.started <- session.workerID
	if w.release != nil {
		<-w.release
	}
	return w.kind, nil
}

type timedWorkload struct {
	mu         sync.Mutex
	completed  chan int
	durations  []time.Duration
	kind       operationKind
	failAtCall int
	callCount  int
	err        error
}

func (w *timedWorkload) RunOnce(_ context.Context, session *workerSession) (operationKind, error) {
	w.mu.Lock()
	w.callCount++
	callNumber := w.callCount
	duration := w.durations[callNumber-1]
	err := error(nil)
	if w.failAtCall == callNumber {
		err = w.err
	}
	w.mu.Unlock()

	session.clock.(*mutableClock).Advance(duration)
	if w.completed != nil {
		w.completed <- session.workerID
	}
	return w.kind, err
}

type blockingPaceGate struct {
	tokens           chan struct{}
	updatedTargetTPS int
}

func newBlockingPaceGate() *blockingPaceGate {
	return &blockingPaceGate{
		tokens: make(chan struct{}, 16),
	}
}

func (g *blockingPaceGate) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-g.tokens:
		return nil
	}
}

func (g *blockingPaceGate) Update(targetTPS *int) {
	if targetTPS != nil {
		g.updatedTargetTPS = *targetTPS
	}
}

func (g *blockingPaceGate) Close() {}

func (g *blockingPaceGate) release() {
	g.tokens <- struct{}{}
}

func waitForWorkerStart(t *testing.T, started <-chan int, want int) {
	t.Helper()

	if got := waitForAnyWorkerStart(t, started); got != want {
		t.Fatalf("worker start = %d, want %d", got, want)
	}
}

func waitForAnyWorkerStart(t *testing.T, started <-chan int) int {
	t.Helper()

	select {
	case workerID := <-started:
		return workerID
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for worker start")
		return -1
	}
}

func waitForOperationCompletions(t *testing.T, completed <-chan int, want int) {
	t.Helper()

	for index := 0; index < want; index++ {
		select {
		case <-completed:
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for operation completion")
		}
	}
}

func waitForUniqueWorkers(t *testing.T, pace *blockingPaceGate, started <-chan int, seen map[int]bool, want int) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for len(seen) < want && time.Now().Before(deadline) {
		pace.release()
		select {
		case workerID := <-started:
			seen[workerID] = true
		case <-time.After(100 * time.Millisecond):
		}
	}

	if len(seen) < want {
		t.Fatalf("unique workers = %v, want %d workers", seen, want)
	}
}

func waitForActiveClients(t *testing.T, run interface{ Snapshot() benchmarkrun.Stats }, want int) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if got := run.Snapshot().ActiveClients; got == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("active clients never reached %d; last snapshot = %#v", want, run.Snapshot())
}
