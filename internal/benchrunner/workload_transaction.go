package benchrunner

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"

	"pg_gobench/internal/benchmark"
)

type transactionWorkload struct {
	scale   benchmark.ScaleModel
	mix     benchmark.TransactionMix
	counter atomic.Uint64
}

func (w *transactionWorkload) RunOnce(ctx context.Context, session *workerSession) (operationKind, error) {
	iteration := w.counter.Add(1)
	return runTransactionalOperation(
		ctx,
		session,
		operationKindTransaction,
		transactionLabels{
			begin:    "begin transaction",
			workload: "transaction workload",
			rollback: "rollback transaction",
			commit:   "commit transaction",
		},
		func(tx *sql.Tx) error {
			return w.runTransaction(ctx, tx, session, iteration)
		},
	)
}

func (w *transactionWorkload) runTransaction(ctx context.Context, tx *sql.Tx, session *workerSession, iteration uint64) error {
	account := accountID(iteration, w.scale)
	branch := branchID(iteration, w.scale)
	teller := tellerID(iteration, w.scale, branch)
	value := amount(iteration)

	if err := pointReadQuery(ctx, tx, account); err != nil {
		return fmt.Errorf("point read in transaction: %w", err)
	}

	switch w.mix {
	case benchmark.TransactionMixReadHeavy:
		if err := rangeReadQuery(
			ctx,
			tx,
			branch,
			int64(branch),
			int64(branch+w.scale.Branches*4),
		); err != nil {
			return fmt.Errorf("range read in transaction: %w", err)
		}
	case benchmark.TransactionMixBalanced, benchmark.TransactionMixWriteHeavy:
	default:
		return fmt.Errorf("transaction mix %q is not implemented", w.mix)
	}

	if err := accountUpdateQuery(ctx, tx, account, value); err != nil {
		return fmt.Errorf("update account in transaction: %w", err)
	}

	if w.mix == benchmark.TransactionMixWriteHeavy {
		if _, err := tx.ExecContext(
			ctx,
			fmt.Sprintf(`UPDATE %s
SET balance = balance + $1
WHERE id = $2`, benchmarkTable("tellers")),
			value,
			teller,
		); err != nil {
			return fmt.Errorf("update teller in transaction: %w", err)
		}
	}

	if err := historyInsertQuery(
		ctx,
		tx,
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
