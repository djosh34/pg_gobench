package benchrunner

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"

	"pg_gobench/internal/benchmark"
)

const lockTimeout = "25ms"

type lockWorkload struct {
	scale   benchmark.ScaleModel
	counter atomic.Uint64
}

func (w *lockWorkload) RunOnce(ctx context.Context, session *workerSession) (operationKind, error) {
	iteration := w.counter.Add(1)
	if iteration%2 == 1 {
		return w.runLockContention(ctx, session, iteration)
	}
	return w.runHotUpdate(ctx, session, iteration)
}

func (w *lockWorkload) runLockContention(ctx context.Context, session *workerSession, iteration uint64) (operationKind, error) {
	return runTransactionalOperation(
		ctx,
		session,
		operationKindLockContention,
		transactionLabels{
			begin:    "begin lock contention transaction",
			workload: "lock contention",
			rollback: "rollback lock contention transaction",
			commit:   "commit lock contention transaction",
		},
		func(tx *sql.Tx) error {
			if err := setLocalLockTimeout(ctx, tx, lockTimeout); err != nil {
				return err
			}
			return lockAccountNowaitQuery(ctx, tx, hotAccountID(iteration, w.scale))
		},
	)
}

func (w *lockWorkload) runHotUpdate(ctx context.Context, session *workerSession, iteration uint64) (operationKind, error) {
	return runTransactionalOperation(
		ctx,
		session,
		operationKindHotUpdate,
		transactionLabels{
			begin:    "begin hot update transaction",
			workload: "hot update",
			rollback: "rollback hot update transaction",
			commit:   "commit hot update transaction",
		},
		func(tx *sql.Tx) error {
			if err := setLocalLockTimeout(ctx, tx, lockTimeout); err != nil {
				return err
			}
			if err := accountUpdateQuery(ctx, tx, hotAccountID(iteration, w.scale), amount(iteration)); err != nil {
				return fmt.Errorf("update account: %w", err)
			}
			return nil
		},
	)
}
