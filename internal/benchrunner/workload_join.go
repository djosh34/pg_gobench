package benchrunner

import (
	"context"
	"fmt"
	"sync/atomic"

	"pg_gobench/internal/benchmark"
)

type joinWorkload struct {
	scale   benchmark.ScaleModel
	counter atomic.Uint64
}

func (w *joinWorkload) RunOnce(ctx context.Context, session *workerSession) (operationKind, error) {
	iteration := w.counter.Add(1)
	if iteration%2 == 1 {
		return w.runJoin(ctx, session, iteration)
	}
	return w.runAggregation(ctx, session, iteration)
}

func (w *joinWorkload) runJoin(ctx context.Context, session *workerSession, iteration uint64) (operationKind, error) {
	if err := joinQuery(ctx, session.db, accountID(iteration, w.scale)); err != nil {
		return operationKindJoin, fmt.Errorf("join workload: %w", err)
	}
	return operationKindJoin, nil
}

func (w *joinWorkload) runAggregation(ctx context.Context, session *workerSession, iteration uint64) (operationKind, error) {
	if err := aggregationQuery(ctx, session.db, branchID(iteration, w.scale)); err != nil {
		return operationKindAggregation, fmt.Errorf("aggregation workload: %w", err)
	}
	return operationKindAggregation, nil
}
