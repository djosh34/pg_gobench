package benchrunner

import (
	"context"
	"fmt"
	"sync/atomic"

	"pg_gobench/internal/benchmark"
)

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
	if err := pointReadQuery(ctx, session.db, accountID(iteration, w.scale)); err != nil {
		return operationKindPointRead, fmt.Errorf("point read: %w", err)
	}
	return operationKindPointRead, nil
}

func (w *readWorkload) runRangeRead(ctx context.Context, session *workerSession, iteration uint64) (operationKind, error) {
	branch := branchID(iteration, w.scale)
	startID := int64(branch)
	endID := startID + int64(w.scale.Branches*9)
	if endID > int64(w.scale.Accounts) {
		endID = int64(w.scale.Accounts)
	}

	if err := rangeReadQuery(ctx, session.db, branch, startID, endID); err != nil {
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
	if err := historyInsertQuery(
		ctx,
		session.db,
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
	if err := accountUpdateQuery(ctx, session.db, accountID(iteration, w.scale), amount(iteration)); err != nil {
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
