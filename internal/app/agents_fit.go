package app

// Sizes of an Agents view's list rows.
const (
	rowHeight, compactRowHeight float32 = 44, 24
	rowGap, groupHeight         float32 = 2, 20
)

// listFit says how a list of n agents under their groups' headings fits
// in height h: in rows, else in compact rows, else with the idle agents
// folded under their heading.
func listFit(h float32, groups, n int) (compact, foldIdle bool) {
	need := func(row float32) float32 {
		return float32(groups)*(groupHeight+rowGap) + float32(n)*(row+rowGap)
	}
	return need(rowHeight) > h, need(compactRowHeight) > h
}
