package benchrunner

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"pg_gobench/internal/benchmark"
	"pg_gobench/internal/benchmarkrun"
)

const benchmarkSchema = "pg_gobench"

type clock interface {
	After(time.Duration) <-chan time.Time
	NewTicker(time.Duration) pacingTicker
	Now() time.Time
}

type pacingTicker interface {
	C() <-chan time.Time
	Stop()
}

type workloadPlan interface {
	RunOnce(context.Context, *workerSession) error
}

type pacingGate interface {
	Wait(context.Context) error
	Update(*int)
	Close()
}

type planFactory func(benchmark.StartOptions, benchmark.ScaleModel) (workloadPlan, error)
type gateFactory func(*int, clock) pacingGate

type runner struct {
	db          *sql.DB
	clock       clock
	newPlan     planFactory
	newPaceGate gateFactory
}

func New(db *sql.DB) benchmarkrun.Runner {
	return runner{
		db:          db,
		clock:       realClock{},
		newPlan:     newSQLWorkloadPlan,
		newPaceGate: newRealPacingGate,
	}
}

func (r runner) Start(ctx context.Context, options benchmark.StartOptions) (benchmarkrun.Run, error) {
	r = r.withDefaults()

	if r.db == nil {
		return nil, fmt.Errorf("setup benchmark schema: database handle is nil")
	}
	if options.Scale < 1 {
		return nil, fmt.Errorf("scale must be at least 1")
	}
	if options.Clients < 1 {
		return nil, fmt.Errorf("clients must be at least 1")
	}
	if options.DurationSeconds < 1 {
		return nil, fmt.Errorf("duration_seconds must be at least 1")
	}
	if options.WarmupSeconds < 0 {
		return nil, fmt.Errorf("warmup_seconds must be at least 0")
	}
	if options.WarmupSeconds >= options.DurationSeconds {
		return nil, fmt.Errorf("warmup_seconds must be less than duration_seconds")
	}

	scale := benchmark.ResolveScale(options.Scale)
	plan, err := r.newPlan(options, scale)
	if err != nil {
		return nil, err
	}

	for _, statement := range setupStatements(options, scale) {
		if _, execErr := r.db.ExecContext(ctx, statement); execErr != nil {
			return nil, fmt.Errorf("setup benchmark schema: %w", execErr)
		}
	}

	return newActiveRun(ctx, r.db, scale, options, plan, r.newPaceGate(options.TargetTPS, r.clock), r.clock), nil
}

func (r runner) withDefaults() runner {
	if r.clock == nil {
		r.clock = realClock{}
	}
	if r.newPlan == nil {
		r.newPlan = newSQLWorkloadPlan
	}
	if r.newPaceGate == nil {
		r.newPaceGate = newRealPacingGate
	}
	return r
}

func setupStatements(options benchmark.StartOptions, scale benchmark.ScaleModel) []string {
	statements := make([]string, 0, 11)
	if options.Reset {
		statements = append(statements, "DROP SCHEMA IF EXISTS pg_gobench CASCADE")
	}

	statements = append(statements,
		"CREATE SCHEMA IF NOT EXISTS pg_gobench",
		`CREATE TABLE IF NOT EXISTS pg_gobench.branches (
			id integer PRIMARY KEY,
			balance bigint NOT NULL,
			name text NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS pg_gobench.tellers (
			id integer PRIMARY KEY,
			branch_id integer NOT NULL REFERENCES pg_gobench.branches(id),
			balance bigint NOT NULL,
			name text NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS pg_gobench.accounts (
			id bigint PRIMARY KEY,
			branch_id integer NOT NULL REFERENCES pg_gobench.branches(id),
			balance bigint NOT NULL,
			name text NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS pg_gobench.history (
			id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			account_id bigint NOT NULL REFERENCES pg_gobench.accounts(id),
			teller_id integer NOT NULL REFERENCES pg_gobench.tellers(id),
			branch_id integer NOT NULL REFERENCES pg_gobench.branches(id),
			amount bigint NOT NULL,
			note text NOT NULL,
			created_at timestamptz NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS pg_gobench_accounts_branch_id_id_idx
			ON pg_gobench.accounts (branch_id, id)`,
		`CREATE INDEX IF NOT EXISTS pg_gobench_tellers_branch_id_id_idx
			ON pg_gobench.tellers (branch_id, id)`,
		`CREATE INDEX IF NOT EXISTS pg_gobench_history_account_id_created_at_idx
			ON pg_gobench.history (account_id, created_at)`,
		fmt.Sprintf(`INSERT INTO pg_gobench.branches (id, balance, name)
SELECT id, 0, format('branch-%%s', id)
FROM generate_series(1, %d) AS id
WHERE NOT EXISTS (SELECT 1 FROM pg_gobench.branches LIMIT 1)`, scale.Branches),
		fmt.Sprintf(`INSERT INTO pg_gobench.tellers (id, branch_id, balance, name)
SELECT id, ((id - 1) / 10) + 1, 0, format('teller-%%s', id)
FROM generate_series(1, %d) AS id
WHERE NOT EXISTS (SELECT 1 FROM pg_gobench.tellers LIMIT 1)`, scale.Tellers),
		fmt.Sprintf(`INSERT INTO pg_gobench.accounts (id, branch_id, balance, name)
SELECT id, ((id - 1) %% %d) + 1, 0, format('account-%%s', id)
FROM generate_series(1, %d) AS id
WHERE NOT EXISTS (SELECT 1 FROM pg_gobench.accounts LIMIT 1)`, scale.Branches, scale.Accounts),
	)

	return statements
}

type realClock struct{}

func (realClock) After(duration time.Duration) <-chan time.Time {
	return time.After(duration)
}

func (realClock) NewTicker(duration time.Duration) pacingTicker {
	return realTicker{Ticker: time.NewTicker(duration)}
}

func (realClock) Now() time.Time {
	return time.Now()
}

type realTicker struct {
	*time.Ticker
}

func (t realTicker) C() <-chan time.Time {
	return t.Ticker.C
}
