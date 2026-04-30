package benchmark

// ScaleModel resolves the public benchmark scale option into concrete dataset
// sizes. The ratios follow the classic pgbench shape so scale stays easy to
// reason about across later workload tasks:
//   - 1 branch per scale unit
//   - 10 tellers per scale unit
//   - 100000 accounts per scale unit
//   - 0 preseeded history rows because history is append-only workload output
type ScaleModel struct {
	Branches    int
	Tellers     int
	Accounts    int
	HistoryRows int
}

func ResolveScale(scale int) ScaleModel {
	return ScaleModel{
		Branches:    scale,
		Tellers:     scale * 10,
		Accounts:    scale * 100000,
		HistoryRows: 0,
	}
}
