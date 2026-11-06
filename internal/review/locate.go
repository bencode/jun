package review

import (
	"slices"
	"strings"
)

// Status is how a note's place was found.
type Status int

const (
	Exact Status = iota // its text is where it was
	Moved               // its text is elsewhere
	Lost                // its text is gone: the lines are where it was
)

// Place is where a note is in a text.
type Place struct {
	Lines  Lines
	Status Status
}

// Locate finds the note's text in text, nearest to where it was: lines
// move as an agent works on, and a note on the whole file has no place.
func Locate(text string, a Anchor) Place {
	if a.Lines == nil {
		return Place{}
	}
	hay, needle := textLines(text), textLines(a.Quote)
	best := -1
	for i := 0; i+len(needle) <= len(hay); i++ {
		if slices.Equal(hay[i:i+len(needle)], needle) && (best < 0 || distance(i, a.Lines.First) < distance(best, a.Lines.First)) {
			best = i
		}
	}
	if best < 0 {
		first := min(max(a.Lines.First, 1), len(hay))
		return Place{Lines{first, min(max(a.Lines.Last, first), len(hay))}, Lost}
	}
	found := Lines{best + 1, best + len(needle)}
	if found.First == a.Lines.First {
		return Place{found, Exact}
	}
	return Place{found, Moved}
}

// distance is how far index i, from 0, is from line, from 1.
func distance(i, line int) int { return max(i+1-line, line-i-1) }

// textLines splits a text into lines, without the space ending them.
func textLines(text string) []string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t\r")
	}
	return lines
}
