package benchrunner

import (
	"context"
	"database/sql"
	"fmt"

	"pg_gobench/internal/benchmark"
	"pg_gobench/internal/benchmarkrun"
)

const benchmarkSchema = "pg_gobench"

type runner struct {
	db *sql.DB
}

type run struct {
	ctx context.Context
}

func New(db *sql.DB) benchmarkrun.Runner {
	return runner{db: db}
}

func (r runner) Start(ctx context.Context, options benchmark.StartOptions) (benchmarkrun.Run, error) {
	if r.db == nil {
		return nil, fmt.Errorf("setup benchmark schema: database handle is nil")
	}

	for _, statement := range setupStatements(options, benchmark.ResolveScale(options.Scale)) {
		if _, err := r.db.ExecContext(ctx, statement); err != nil {
			return nil, fmt.Errorf("setup benchmark schema: %w", err)
		}
	}

	return run{ctx: ctx}, nil
}

func (r run) Alter(benchmark.AlterOptions) error {
	return nil
}

func (r run) Wait() error {
	<-r.ctx.Done()
	return r.ctx.Err()
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
