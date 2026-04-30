package benchmarkrun

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"pg_gobench/internal/benchmark"
)

type Status string

const (
	StatusIdle     Status = "idle"
	StatusStarting Status = "starting"
	StatusRunning  Status = "running"
	StatusStopping Status = "stopping"
	StatusStopped  Status = "stopped"
	StatusFailed   Status = "failed"
)

type State struct {
	Status    Status                 `json:"status"`
	Options   benchmark.StartOptions `json:"options"`
	StartedAt *time.Time             `json:"started_at,omitempty"`
	StoppedAt *time.Time             `json:"stopped_at,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

type Runner interface {
	Start(ctx context.Context, options benchmark.StartOptions) (Run, error)
}

type Run interface {
	Alter(options benchmark.AlterOptions) error
	Sample() Sample
	Wait() error
}

type Option func(*Coordinator)

type Coordinator struct {
	mu     sync.Mutex
	runner Runner
	now    func() time.Time
	state  State
	sample Sample
	run    Run
	cancel context.CancelFunc
	runID  uint64
}

var ErrRunActive = errors.New("benchmark run already active")
var ErrRunNotRunning = errors.New("benchmark run is not running")

func New(runner Runner, opts ...Option) *Coordinator {
	coordinator := &Coordinator{
		runner: runner,
		now:    time.Now,
		state: State{
			Status: StatusIdle,
		},
		sample: zeroSample(),
	}

	for _, opt := range opts {
		opt(coordinator)
	}

	return coordinator
}

func WithNow(now func() time.Time) Option {
	return func(c *Coordinator) {
		if now != nil {
			c.now = now
		}
	}
}

func (c *Coordinator) Start(ctx context.Context, options benchmark.StartOptions) (State, error) {
	if err := ctx.Err(); err != nil {
		return c.State(), fmt.Errorf("start benchmark run: %w", err)
	}

	c.mu.Lock()
	if c.state.Status == StatusStarting || c.state.Status == StatusRunning || c.state.Status == StatusStopping {
		state := cloneState(c.state)
		c.mu.Unlock()
		return state, ErrRunActive
	}
	runCtx, cancel := context.WithCancel(context.Background())
	c.runID++
	runID := c.runID
	c.cancel = cancel
	c.run = nil
	c.state = State{
		Status:  StatusStarting,
		Options: cloneStartOptions(options),
	}
	c.sample = zeroSample()
	c.mu.Unlock()

	if c.runner == nil {
		return c.finishStartFailure(runID, options, errors.New("benchmark runner is nil"))
	}

	run, err := c.runner.Start(runCtx, cloneStartOptions(options))
	if err != nil {
		return c.finishStartFailure(runID, options, err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.runID != runID {
		return cloneState(c.state), nil
	}

	c.run = run
	if c.state.StartedAt == nil {
		startedAt := c.now()
		c.state.StartedAt = timePtr(startedAt)
	}
	if c.state.Status == StatusStarting {
		c.state.Status = StatusRunning
		c.state.Error = ""
	}

	go c.waitForRun(runID, run)

	return cloneState(c.state), nil
}

func (c *Coordinator) Alter(options benchmark.AlterOptions) (State, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state.Status != StatusRunning || c.run == nil {
		return cloneState(c.state), ErrRunNotRunning
	}

	updated, err := c.state.Options.ApplyAlter(options)
	if err != nil {
		return cloneState(c.state), err
	}
	if err := c.run.Alter(options); err != nil {
		return cloneState(c.state), err
	}

	c.state.Options = updated

	return cloneState(c.state), nil
}

func (c *Coordinator) Stop() (State, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch c.state.Status {
	case StatusIdle, StatusStopped, StatusFailed:
		return cloneState(c.state), nil
	case StatusStopping:
		return cloneState(c.state), nil
	case StatusStarting, StatusRunning:
		c.state.Status = StatusStopping
		if c.cancel != nil {
			c.cancel()
		}
		return cloneState(c.state), nil
	default:
		return cloneState(c.state), nil
	}
}

func (c *Coordinator) State() State {
	c.mu.Lock()
	defer c.mu.Unlock()

	return cloneState(c.state)
}

func (c *Coordinator) Results() Results {
	c.mu.Lock()
	defer c.mu.Unlock()

	sample := c.sample
	if c.run != nil {
		sample = c.run.Sample()
	}

	return stateToResults(c.state, sample)
}

func (c *Coordinator) Metrics() MetricsSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	sample := c.sample
	if c.run != nil {
		sample = c.run.Sample()
	}
	return sample.Metrics(c.state.Status == StatusStarting || c.state.Status == StatusRunning || c.state.Status == StatusStopping)
}

func cloneState(state State) State {
	cloned := State{
		Status:  state.Status,
		Options: cloneStartOptions(state.Options),
		Error:   state.Error,
	}
	if state.StartedAt != nil {
		cloned.StartedAt = timePtr(*state.StartedAt)
	}
	if state.StoppedAt != nil {
		cloned.StoppedAt = timePtr(*state.StoppedAt)
	}
	return cloned
}

func (c *Coordinator) finishStartFailure(runID uint64, options benchmark.StartOptions, err error) (State, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.runID != runID {
		return cloneState(c.state), err
	}

	c.run = nil
	c.cancel = nil
	stoppedAt := c.now()
	if c.state.Status == StatusStopping && errors.Is(err, context.Canceled) {
		c.state.Status = StatusStopped
		c.state.Error = ""
		c.state.Options = cloneStartOptions(options)
		c.state.StoppedAt = timePtr(stoppedAt)
		c.sample = zeroSample()
		return cloneState(c.state), err
	}

	c.state = State{
		Status:    StatusFailed,
		Options:   cloneStartOptions(options),
		StoppedAt: timePtr(stoppedAt),
		Error:     compactErrorText(err.Error()),
	}
	c.sample = zeroSample()

	return cloneState(c.state), err
}

func (c *Coordinator) waitForRun(runID uint64, run Run) {
	err := run.Wait()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.runID != runID {
		return
	}

	c.sample = run.Sample()
	c.run = nil
	c.cancel = nil
	stoppedAt := c.now()
	c.state.StoppedAt = timePtr(stoppedAt)

	switch {
	case c.state.Status == StatusStopping:
		c.state.Status = StatusStopped
		c.state.Error = ""
	case err == nil:
		c.state.Status = StatusStopped
		c.state.Error = ""
	case errors.Is(err, context.Canceled):
		c.state.Status = StatusFailed
		c.state.Error = compactErrorText(err.Error())
	default:
		c.state.Status = StatusFailed
		c.state.Error = compactErrorText(err.Error())
	}
}

func compactErrorText(message string) string {
	compact := strings.Join(strings.Fields(message), " ")
	if len(compact) <= 160 {
		return compact
	}
	return compact[:157] + "..."
}

func cloneStartOptions(options benchmark.StartOptions) benchmark.StartOptions {
	cloned := options
	if options.ReadPercent != nil {
		cloned.ReadPercent = intPtr(*options.ReadPercent)
	}
	if options.TargetTPS != nil {
		cloned.TargetTPS = intPtr(*options.TargetTPS)
	}
	return cloned
}

func timePtr(value time.Time) *time.Time {
	return &value
}

func intPtr(value int) *int {
	return &value
}
