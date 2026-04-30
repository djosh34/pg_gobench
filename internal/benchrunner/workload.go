package benchrunner

import (
	"fmt"

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
	operationKindJoin
	operationKindAggregation
	operationKindLockContention
	operationKindHotUpdate
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
	case benchmark.ProfileJoin:
		return &joinWorkload{scale: scale}, nil
	case benchmark.ProfileLock:
		return &lockWorkload{scale: scale}, nil
	default:
		return nil, fmt.Errorf("profile %q is not implemented yet", options.Profile)
	}
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

func hotAccountID(iteration uint64, scale benchmark.ScaleModel) int64 {
	hotSetSize := scale.Accounts
	if hotSetSize > 8 {
		hotSetSize = 8
	}
	return int64((iteration-1)%uint64(hotSetSize) + 1)
}

func amount(iteration uint64) int64 {
	return int64((iteration % 97) + 1)
}
