package app

// Sizes of an Agents view's cards and list rows.
const (
	cardWidth, cardHeight       float32 = 360, 260 // the largest cards
	cardMinWidth, cardMinHeight float32 = 280, 180 // smaller cards would not read
	cardGap                     float32 = 8        // MyGo GridView's gap
	rowHeight, compactRowHeight float32 = 44, 24
	rowGap, groupHeight         float32 = 2, 20
)

// cardGrid finds the widest cards showing n agents in w×h without
// scrolling: from as many columns of cardWidth as fit, it adds columns
// until the rows fit, while the cards stay wide enough. GridView puts a
// gap around each row.
func cardGrid(w, h float32, n int) (width, height float32, ok bool) {
	first := max(1, int((w+cardGap)/(cardWidth+cardGap)))
	for cols := first; cols <= max(n, first); cols++ {
		width = (w - float32(cols-1)*cardGap) / float32(cols)
		if width < cardMinWidth {
			return 0, 0, false
		}
		rows := (n + cols - 1) / cols
		if height = h/float32(rows) - cardGap; height >= cardMinHeight {
			return width, min(height, cardHeight), true
		}
	}
	return 0, 0, false
}

// listFit says how a list of n agents under their groups' headings fits
// in height h: in rows, else in compact rows, else with the idle agents
// folded under their heading.
func listFit(h float32, groups, n int) (compact, foldIdle bool) {
	need := func(row float32) float32 {
		return float32(groups)*(groupHeight+rowGap) + float32(n)*(row+rowGap)
	}
	return need(rowHeight) > h, need(compactRowHeight) > h
}
