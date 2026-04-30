package benchrunner

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"sync"
	"time"

	"pg_gobench/internal/benchmark"
)

type workerSession struct {
	db       *sql.DB
	scale    benchmark.ScaleModel
	clock    clock
	workerID int
}

type activeRun struct {
	db    *sql.DB
	scale benchmark.ScaleModel
	plan  workloadPlan
	pace  pacingGate
	clock clock

	stopCtx    context.Context
	stopCancel context.CancelFunc
	done       chan struct{}

	mu         sync.Mutex
	workers    map[int]context.CancelFunc
	nextID     int
	finalErr   error
	finishOnce sync.Once
	wg         sync.WaitGroup
}

func newActiveRun(
	parent context.Context,
	db *sql.DB,
	scale benchmark.ScaleModel,
	options benchmark.StartOptions,
	plan workloadPlan,
	pace pacingGate,
	clock clock,
) *activeRun {
	stopCtx, stopCancel := context.WithCancel(context.Background())
	run := &activeRun{
		db:         db,
		scale:      scale,
		plan:       plan,
		pace:       pace,
		clock:      clock,
		stopCtx:    stopCtx,
		stopCancel: stopCancel,
		done:       make(chan struct{}),
		workers:    map[int]context.CancelFunc{},
	}

	run.resizeWorkers(options.Clients)
	go run.watchParent(parent)
	go run.watchDuration(time.Duration(options.DurationSeconds) * time.Second)
	go run.awaitCompletion()

	return run
}

func (r *activeRun) Alter(options benchmark.AlterOptions) error {
	if options.Clients == nil && options.TargetTPS == nil {
		return fmt.Errorf("alter request must include at least one field")
	}
	if options.Clients != nil && *options.Clients < 1 {
		return fmt.Errorf("clients must be at least 1")
	}
	if options.TargetTPS != nil && *options.TargetTPS < 1 {
		return fmt.Errorf("target_tps must be at least 1")
	}

	if options.TargetTPS != nil {
		r.pace.Update(options.TargetTPS)
	}
	if options.Clients != nil {
		r.resizeWorkers(*options.Clients)
	}

	return nil
}

func (r *activeRun) Wait() error {
	<-r.done

	r.mu.Lock()
	defer r.mu.Unlock()
	return r.finalErr
}

func (r *activeRun) watchParent(parent context.Context) {
	select {
	case <-r.done:
		return
	case <-parent.Done():
		r.finish(parent.Err())
	}
}

func (r *activeRun) watchDuration(duration time.Duration) {
	select {
	case <-r.done:
		return
	case <-r.clock.After(duration):
		r.finish(nil)
	}
}

func (r *activeRun) awaitCompletion() {
	r.wg.Wait()
	r.pace.Close()
	close(r.done)
}

func (r *activeRun) finish(err error) {
	r.finishOnce.Do(func() {
		r.mu.Lock()
		r.finalErr = err
		r.mu.Unlock()
		r.stopCancel()
	})
}

func (r *activeRun) resizeWorkers(target int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for len(r.workers) < target {
		r.startWorkerLocked()
	}
	if len(r.workers) <= target {
		return
	}

	workerIDs := make([]int, 0, len(r.workers))
	for workerID := range r.workers {
		workerIDs = append(workerIDs, workerID)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(workerIDs)))

	for _, workerID := range workerIDs {
		if len(r.workers) <= target {
			return
		}
		cancel := r.workers[workerID]
		delete(r.workers, workerID)
		cancel()
	}
}

func (r *activeRun) startWorkerLocked() {
	workerID := r.nextID
	r.nextID++

	ctx, cancel := context.WithCancel(r.stopCtx)
	r.workers[workerID] = cancel
	r.wg.Add(1)

	go func() {
		defer r.wg.Done()
		defer r.workerStopped(workerID)

		session := &workerSession{
			db:       r.db,
			scale:    r.scale,
			clock:    r.clock,
			workerID: workerID,
		}

		for {
			if err := r.pace.Wait(ctx); err != nil {
				if ctx.Err() != nil {
					return
				}
				r.finish(fmt.Errorf("pace workload: %w", err))
				return
			}

			if err := r.plan.RunOnce(ctx, session); err != nil {
				if ctx.Err() != nil {
					return
				}
				r.finish(err)
				return
			}
		}
	}()
}

func (r *activeRun) workerStopped(workerID int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.workers, workerID)
}

type realPacingGate struct {
	clock  clock
	mu     sync.Mutex
	ticker pacingTicker
}

func newRealPacingGate(targetTPS *int, clock clock) pacingGate {
	gate := &realPacingGate{clock: clock}
	gate.Update(targetTPS)
	return gate
}

func (g *realPacingGate) Wait(ctx context.Context) error {
	g.mu.Lock()
	ticker := g.ticker
	g.mu.Unlock()

	if ticker == nil {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ticker.C():
		return nil
	}
}

func (g *realPacingGate) Update(targetTPS *int) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.ticker != nil {
		g.ticker.Stop()
		g.ticker = nil
	}
	if targetTPS == nil {
		return
	}

	interval := time.Second / time.Duration(*targetTPS)
	if interval <= 0 {
		interval = time.Nanosecond
	}
	g.ticker = g.clock.NewTicker(interval)
}

func (g *realPacingGate) Close() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.ticker != nil {
		g.ticker.Stop()
		g.ticker = nil
	}
}
