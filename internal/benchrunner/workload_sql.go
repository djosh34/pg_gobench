package benchrunner

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type sqlExecutor interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type transactionLabels struct {
	begin    string
	workload string
	rollback string
	commit   string
}

func runTransactionalOperation(
	ctx context.Context,
	session *workerSession,
	kind operationKind,
	labels transactionLabels,
	step func(*sql.Tx) error,
) (operationKind, error) {
	tx, err := session.db.BeginTx(ctx, nil)
	if err != nil {
		return kind, fmt.Errorf("%s: %w", labels.begin, err)
	}

	if err := step(tx); err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			return kind, errors.Join(
				fmt.Errorf("%s: %w", labels.workload, err),
				fmt.Errorf("%s: %w", labels.rollback, rollbackErr),
			)
		}
		return kind, fmt.Errorf("%s: %w", labels.workload, err)
	}

	if err := tx.Commit(); err != nil {
		return kind, fmt.Errorf("%s: %w", labels.commit, err)
	}
	return kind, nil
}

func pointReadQuery(ctx context.Context, executor sqlExecutor, account int64) error {
	return executor.QueryRowContext(
		ctx,
		`SELECT balance, name
FROM pg_gobench.accounts
WHERE id = $1`,
		account,
	).Scan(new(int64), new(string))
}

func rangeReadQuery(ctx context.Context, executor sqlExecutor, branch int, startID int64, endID int64) error {
	rows, err := executor.QueryContext(
		ctx,
		`SELECT id, balance
FROM pg_gobench.accounts
WHERE branch_id = $1
  AND id BETWEEN $2 AND $3
ORDER BY id`,
		branch,
		startID,
		endID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(new(int64), new(int64)); err != nil {
			return err
		}
	}
	return rows.Err()
}

func historyInsertQuery(
	ctx context.Context,
	executor sqlExecutor,
	account int64,
	teller int,
	branch int,
	value int64,
	note string,
	createdAt time.Time,
) error {
	_, err := executor.ExecContext(
		ctx,
		`INSERT INTO pg_gobench.history (account_id, teller_id, branch_id, amount, note, created_at)
VALUES ($1, $2, $3, $4, $5, $6)`,
		account,
		teller,
		branch,
		value,
		note,
		createdAt,
	)
	return err
}

func accountUpdateQuery(ctx context.Context, executor sqlExecutor, account int64, value int64) error {
	_, err := executor.ExecContext(
		ctx,
		`UPDATE pg_gobench.accounts
SET balance = balance + $1
WHERE id = $2`,
		value,
		account,
	)
	return err
}

func joinQuery(ctx context.Context, executor sqlExecutor, account int64) error {
	return executor.QueryRowContext(
		ctx,
		`SELECT a.id, a.name, b.name, t.name, a.balance
FROM pg_gobench.accounts AS a
JOIN pg_gobench.branches AS b ON b.id = a.branch_id
JOIN pg_gobench.tellers AS t ON t.branch_id = b.id
WHERE a.id = $1
ORDER BY t.id
LIMIT 1`,
		account,
	).Scan(new(int64), new(string), new(string), new(string), new(int64))
}

func aggregationQuery(ctx context.Context, executor sqlExecutor, branch int) error {
	rows, err := executor.QueryContext(
		ctx,
		`SELECT a.branch_id, COUNT(*), COALESCE(SUM(a.balance), 0)
FROM pg_gobench.accounts AS a
WHERE a.branch_id = $1
GROUP BY a.branch_id`,
		branch,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(new(int64), new(int64), new(int64)); err != nil {
			return err
		}
	}
	return rows.Err()
}

func setLocalLockTimeout(ctx context.Context, executor sqlExecutor, timeout string) error {
	_, err := executor.ExecContext(ctx, fmt.Sprintf("SET LOCAL lock_timeout = '%s'", timeout))
	return err
}

func lockAccountNowaitQuery(ctx context.Context, executor sqlExecutor, account int64) error {
	return executor.QueryRowContext(
		ctx,
		`SELECT id
FROM pg_gobench.accounts
WHERE id = $1
FOR UPDATE NOWAIT`,
		account,
	).Scan(new(int64))
}
