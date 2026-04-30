package benchrunner

import (
	"context"
	"testing"
	"time"

	"pg_gobench/internal/benchmark"
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
}

func (w *recordingWorkload) RunOnce(_ context.Context, session *workerSession) error {
	w.started <- session.workerID
	if w.release != nil {
		<-w.release
	}
	return nil
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
