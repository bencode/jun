package app

import (
	"hash/fnv"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

var perspectiveIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="m12 3 9 5-9 5-9-5 9-5Zm-9 9 9 5 9-5M3 16l9 5 9-5"/></svg>`))

var settingsIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="m9.5 3-.5 2.5-2 1.2-2.4-.8-2.5 4.3L4 12l-1.9 1.8 2.5 4.3 2.4-.8 2 1.2.5 2.5h5l.5-2.5 2-1.2 2.4.8 2.5-4.3L20 12l1.9-1.8L19.4 6l-2.4.8-2-1.2-.5-2.5h-5Z"/><circle cx="12" cy="12" r="3"/></svg>`))

var downIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="m4 6 4 4 4-4"/></svg>`))

const (
	logo       = "J"
	moreIcon   = "⋯"
	closeIcon  = "×"
	globalIcon = "◧"
)

// projectAbbrev tells projects apart in the collapsed rail with two
// characters: the initials of the name's first two words, words parted by
// separators, camelCase and digits, as in "cc-mission-control" (CM) and
// "jun2" (J2), or else the name's first two letters.
func projectAbbrev(name string) string {
	words := projectWords(name)
	upper := func(r rune) string { return string(unicode.ToUpper(r)) }
	if len(words) >= 2 {
		return upper(words[0][0]) + upper(words[1][0])
	}
	if len(words) == 0 {
		first, _ := utf8.DecodeRuneInString(name)
		return upper(first)
	}
	if w := words[0]; len(w) > 1 {
		return upper(w[0]) + string(w[1])
	}
	return upper(words[0][0])
}

func projectWords(name string) [][]rune {
	var words [][]rune
	var previous rune
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			previous = 0
			continue
		}
		if previous == 0 || unicode.IsLower(previous) && unicode.IsUpper(r) || unicode.IsDigit(previous) != unicode.IsDigit(r) {
			words = append(words, nil)
		}
		words[len(words)-1] = append(words[len(words)-1], r)
		previous = r
	}
	return words
}

// projectColors are tiles' colors, readable under white text.
var projectColors = [...]ui.Color{
	ui.Hex("#3D6FD9"), ui.Hex("#2E9963"), ui.Hex("#C7752B"), ui.Hex("#8E5BC7"),
	ui.Hex("#C44569"), ui.Hex("#2A93A8"), ui.Hex("#A88A2A"), ui.Hex("#5A63C9"),
}

// projectColor picks a project's tile color by its name, the same each launch.
func projectColor(name string) ui.Color {
	h := fnv.New32a()
	h.Write([]byte(name))
	return projectColors[h.Sum32()%uint32(len(projectColors))]
}
