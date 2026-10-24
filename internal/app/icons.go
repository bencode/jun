package app

import (
	"strings"
	"unicode/utf8"
)

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
