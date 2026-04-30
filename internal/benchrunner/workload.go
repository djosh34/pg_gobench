package benchrunner

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync/atomic"

	"pg_gobench/internal/benchmark"
)

const defaultMixedReadPercent = 80

type operationKind int

const (
	operationKindPointRead operationKind = iota
	operationKindRangeRead
	operationKindHistoryInsert
	operationKindAccountUpdate
	operationKindTransaction
	operationKindCount
)

func newSQLWorkloadPlan(options benchmark.StartOptions, scale benchmark.ScaleModel) (workloadPlan, error) {
	switch options.Profile {
	case benchmark.ProfileRead:
		return &readWorkload{scale: scale}, nil
	case benchmark.ProfileWrite:
		return &writeWorkload{scale: scale}, nil
	case benchmark.ProfileMixed:
		return &mixedWorkload{
			readPercent: effectiveReadPercent(options),
			read:        &readWorkload{scale: scale},
			write:       &writeWorkload{scale: scale},
		}, nil
	case benchmark.ProfileTransaction:
		return &transactionWorkload{
			scale: scale,
			mix:   effectiveTransactionMix(options),
		}, nil
	case benchmark.ProfileJoin, benchmark.ProfileLock:
		return nil, fmt.Errorf("profile %q is not implemented yet", options.Profile)
	default:
		return nil, fmt.Errorf("profile %q is not implemented yet", options.Profile)
	}
}

type readWorkload struct {
	scale   benchmark.ScaleModel
	counter atomic.Uint64
}

func (w *readWorkload) RunOnce(ctx context.Context, session *workerSession) (operationKind, error) {
	iteration := w.counter.Add(1)
	if iteration%2 == 1 {
		return w.runPointRead(ctx, session, iteration)
	}
	return w.runRangeRead(ctx, session, iteration)
}

func (w *readWorkload) runPointRead(ctx context.Context, session *workerSession, iteration uint64) (operationKind, error) {
	var (
		balance int64
		name    string
	)
	if err := session.db.QueryRowContext(
		ctx,
		`SELECT balance, name
FROM pg_gobench.accounts
	WHERE id = $1`,
		accountID(iteration, w.scale),
	).Scan(&balance, &name); err != nil {
		return operationKindPointRead, fmt.Errorf("point read: %w", err)
	}
	return operationKindPointRead, nil
}

func (w *readWorkload) runRangeRead(ctx context.Context, session *workerSession, iteration uint64) (operationKind, error) {
	branchID := branchID(iteration, w.scale)
	startID := int64(branchID)
	endID := startID + int64(w.scale.Branches*9)
	if endID > int64(w.scale.Accounts) {
		endID = int64(w.scale.Accounts)
	}

	rows, err := session.db.QueryContext(
		ctx,
		`SELECT id, balance
FROM pg_gobench.accounts
WHERE branch_id = $1
  AND id BETWEEN $2 AND $3
ORDER BY id`,
		branchID,
		startID,
		endID,
	)
	if err != nil {
		return operationKindRangeRead, fmt.Errorf("range read: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id      int64
			balance int64
		)
		if scanErr := rows.Scan(&id, &balance); scanErr != nil {
			return operationKindRangeRead, fmt.Errorf("range read: %w", scanErr)
		}
	}
	if err := rows.Err(); err != nil {
		return operationKindRangeRead, fmt.Errorf("range read: %w", err)
	}
	return operationKindRangeRead, nil
}

type writeWorkload struct {
	scale   benchmark.ScaleModel
	counter atomic.Uint64
}

func (w *writeWorkload) RunOnce(ctx context.Context, session *workerSession) (operationKind, error) {
	iteration := w.counter.Add(1)
	if iteration%2 == 1 {
		return w.runInsert(ctx, session, iteration)
	}
	return w.runUpdate(ctx, session, iteration)
}

func (w *writeWorkload) runInsert(ctx context.Context, session *workerSession, iteration uint64) (operationKind, error) {
	account := accountID(iteration, w.scale)
	branch := branchID(iteration, w.scale)
	teller := tellerID(iteration, w.scale, branch)

	if _, err := session.db.ExecContext(
		ctx,
		`INSERT INTO pg_gobench.history (account_id, teller_id, branch_id, amount, note, created_at)
VALUES ($1, $2, $3, $4, $5, $6)`,
		account,
		teller,
		branch,
		amount(iteration),
		fmt.Sprintf("history-%d", iteration),
		session.clock.Now().UTC(),
	); err != nil {
		return operationKindHistoryInsert, fmt.Errorf("insert history: %w", err)
	}

	return operationKindHistoryInsert, nil
}

func (w *writeWorkload) runUpdate(ctx context.Context, session *workerSession, iteration uint64) (operationKind, error) {
	if _, err := session.db.ExecContext(
		ctx,
		`UPDATE pg_gobench.accounts
SET balance = balance + $1
WHERE id = $2`,
		amount(iteration),
		accountID(iteration, w.scale),
	); err != nil {
		return operationKindAccountUpdate, fmt.Errorf("update account: %w", err)
	}

	return operationKindAccountUpdate, nil
}

type mixedWorkload struct {
	readPercent int
	read        workloadPlan
	write       workloadPlan
	counter     atomic.Uint64
}

func (w *mixedWorkload) RunOnce(ctx context.Context, session *workerSession) (operationKind, error) {
	iteration := w.counter.Add(1)
	if int(iteration%100) < w.readPercent {
		return w.read.RunOnce(ctx, session)
	}
	return w.write.RunOnce(ctx, session)
}

type transactionWorkload struct {
	scale   benchmark.ScaleModel
	mix     benchmark.TransactionMix
	counter atomic.Uint64
}

func (w *transactionWorkload) RunOnce(ctx context.Context, session *workerSession) (operationKind, error) {
	iteration := w.counter.Add(1)
	tx, err := session.db.BeginTx(ctx, nil)
	if err != nil {
		return operationKindTransaction, fmt.Errorf("begin transaction: %w", err)
	}

	if err := w.runTransaction(ctx, tx, session, iteration); err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			return operationKindTransaction, errors.Join(fmt.Errorf("transaction workload: %w", err), fmt.Errorf("rollback transaction: %w", rollbackErr))
		}
		return operationKindTransaction, fmt.Errorf("transaction workload: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return operationKindTransaction, fmt.Errorf("commit transaction: %w", err)
	}
	return operationKindTransaction, nil
}

func (w *transactionWorkload) runTransaction(ctx context.Context, tx *sql.Tx, session *workerSession, iteration uint64) error {
	account := accountID(iteration, w.scale)
	branch := branchID(iteration, w.scale)
	teller := tellerID(iteration, w.scale, branch)
	value := amount(iteration)

	if err := tx.QueryRowContext(
		ctx,
		`SELECT balance, name
FROM pg_gobench.accounts
WHERE id = $1`,
		account,
	).Scan(new(int64), new(string)); err != nil {
		return fmt.Errorf("point read in transaction: %w", err)
	}

	switch w.mix {
	case benchmark.TransactionMixReadHeavy:
		rows, err := tx.QueryContext(
			ctx,
			`SELECT id, balance
FROM pg_gobench.accounts
WHERE branch_id = $1
  AND id BETWEEN $2 AND $3
ORDER BY id`,
			branch,
			int64(branch),
			int64(branch+w.scale.Branches*4),
		)
		if err != nil {
			return fmt.Errorf("range read in transaction: %w", err)
		}
		for rows.Next() {
			if err := rows.Scan(new(int64), new(int64)); err != nil {
				rows.Close()
				return fmt.Errorf("range read in transaction: %w", err)
			}
		}
		if err := rows.Close(); err != nil {
			return fmt.Errorf("range read in transaction: %w", err)
		}
	case benchmark.TransactionMixBalanced, benchmark.TransactionMixWriteHeavy:
	default:
		return fmt.Errorf("transaction mix %q is not implemented", w.mix)
	}

	if _, err := tx.ExecContext(
		ctx,
		`UPDATE pg_gobench.accounts
SET balance = balance + $1
WHERE id = $2`,
		value,
		account,
	); err != nil {
		return fmt.Errorf("update account in transaction: %w", err)
	}

	if w.mix == benchmark.TransactionMixWriteHeavy {
		if _, err := tx.ExecContext(
			ctx,
			`UPDATE pg_gobench.tellers
SET balance = balance + $1
WHERE id = $2`,
			value,
			teller,
		); err != nil {
			return fmt.Errorf("update teller in transaction: %w", err)
		}
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO pg_gobench.history (account_id, teller_id, branch_id, amount, note, created_at)
VALUES ($1, $2, $3, $4, $5, $6)`,
		account,
		teller,
		branch,
		value,
		fmt.Sprintf("tx-%d", iteration),
		session.clock.Now().UTC(),
	); err != nil {
		return fmt.Errorf("insert history in transaction: %w", err)
	}

	return nil
}

func effectiveReadPercent(options benchmark.StartOptions) int {
	if options.ReadPercent != nil {
		return *options.ReadPercent
	}
	return defaultMixedReadPercent
}

func effectiveTransactionMix(options benchmark.StartOptions) benchmark.TransactionMix {
	if options.TransactionMix != "" {
		return options.TransactionMix
	}
	return benchmark.TransactionMixBalanced
}

func accountID(iteration uint64, scale benchmark.ScaleModel) int64 {
	return int64((iteration-1)%uint64(scale.Accounts) + 1)
}

func branchID(iteration uint64, scale benchmark.ScaleModel) int {
	return int((iteration-1)%uint64(scale.Branches)) + 1
}

func tellerID(iteration uint64, scale benchmark.ScaleModel, branch int) int {
	offset := int((iteration - 1) % 10)
	base := (branch-1)*10 + 1
	if base+offset > scale.Tellers {
		return scale.Tellers
	}
	return base + offset
}

func amount(iteration uint64) int64 {
	return int64((iteration % 97) + 1)
}
