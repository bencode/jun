package app

import (
	"strings"
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

func projectIcon(name string) string {
	first, _ := utf8.DecodeRuneInString(name)
	return strings.ToUpper(string(first))
}
