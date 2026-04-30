package benchmark_test

import (
	"testing"

	"pg_gobench/internal/benchmark"
)

func TestResolveScaleMapsBenchmarkScaleToConcreteDatasetSizes(t *testing.T) {
	testCases := []struct {
		name  string
		scale int
		want  benchmark.ScaleModel
	}{
		{
			name:  "single scale unit",
			scale: 1,
			want: benchmark.ScaleModel{
				Branches:    1,
				Tellers:     10,
				Accounts:    100000,
				HistoryRows: 0,
			},
		},
		{
			name:  "larger scale preserves ratios",
			scale: 7,
			want: benchmark.ScaleModel{
				Branches:    7,
				Tellers:     70,
				Accounts:    700000,
				HistoryRows: 0,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := benchmark.ResolveScale(tc.scale)
			if got != tc.want {
				t.Fatalf("ResolveScale(%d) = %#v, want %#v", tc.scale, got, tc.want)
			}
		})
	}
}
