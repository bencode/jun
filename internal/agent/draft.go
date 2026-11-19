package agent

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// HasDraft reports whether the agent's input holds text the user has typed
// and not submitted, as its screen shows it: vt is the terminal's snapshot,
// its text with their styles. Agents show the hints in an empty input, such
// as a placeholder or a suggested prompt, dim; the user's text is not.
func HasDraft(kind Kind, vt []byte) bool {
	lines := strings.Split(strings.TrimRightFunc(undimmed(vt), unicode.IsSpace), "\n")
	lines = lines[max(0, len(lines)-tailLines):]
	switch kind {
	case Codex:
		return codexDraft(lines)
	case Claude:
		return boxDraft(lines, "❯!") // its prompt, ! in shell mode
	default:
		return boxDraft(lines, "")
	}
}

// boxDraft reads the input Claude Code and Pi show between the last two
// rules on the screen, past the prompt the box starts with, if one of
// prompts.
func boxDraft(lines []string, prompts string) bool {
	bottom := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if !strings.HasPrefix(strings.TrimSpace(lines[i]), "──") {
			continue
		}
		if bottom < 0 {
			bottom = i
			continue
		}
		input := strings.TrimSpace(strings.Join(lines[i+1:bottom], "\n"))
		if r, n := utf8.DecodeRuneInString(input); n > 0 && strings.ContainsRune(prompts, r) {
			input = strings.TrimSpace(input[n:])
		}
		return input != ""
	}
	return false
}

// codexDraft reads Codex's input, the last line that starts with its ›: a
// draft starts on it.
func codexDraft(lines []string) bool {
	for i := len(lines) - 1; i >= 0; i-- {
		if rest, ok := strings.CutPrefix(strings.TrimLeftFunc(lines[i], unicode.IsSpace), "›"); ok {
			return strings.TrimSpace(rest) != ""
		}
	}
	return false
}

// undimmed turns a VT snapshot into its text, a line each row, with dim
// text as blanks.
func undimmed(vt []byte) string {
	var b strings.Builder
	dim := false
	for i := 0; i < len(vt); {
		c := vt[i]
		switch {
		case c == 0x1b && i+1 < len(vt):
			var params string
			i, params = escape(vt, i)
			if params != "" {
				dim = sgrDim(params, dim)
			}
		case c == '\n':
			b.WriteByte('\n')
			i++
		case c < 0x20 || c == 0x7f:
			i++
		default:
			r, n := utf8.DecodeRune(vt[i:])
			i += n
			if dim {
				r = ' '
			}
			b.WriteRune(r)
		}
	}
	return b.String()
}

// escape skips the escape sequence at vt[i], and returns where it ends and,
// for a CSI that sets the style (SGR), its parameters, "0" for none.
func escape(vt []byte, i int) (int, string) {
	switch vt[i+1] {
	case '[':
		j := i + 2
		for j < len(vt) && (vt[j] < 0x40 || vt[j] > 0x7e) {
			j++
		}
		if j >= len(vt) {
			return len(vt), ""
		}
		params := string(vt[i+2 : j])
		if vt[j] != 'm' || strings.ContainsAny(params, "<=>?") {
			return j + 1, ""
		}
		if params == "" {
			params = "0"
		}
		return j + 1, params
	case ']', 'P', '_', '^', 'X':
		// A string, up to BEL or ST.
		for j := i + 2; j < len(vt); j++ {
			if vt[j] == 0x07 {
				return j + 1, ""
			}
			if vt[j] == 0x1b && j+1 < len(vt) && vt[j+1] == '\\' {
				return j + 2, ""
			}
		}
		return len(vt), ""
	case '(', ')', '*', '+', '#', '%':
		return min(i+3, len(vt)), ""
	default:
		return i + 2, ""
	}
}

// sgrDim is whether text is dim after the SGR parameters, dim before them.
func sgrDim(params string, dim bool) bool {
	fields := strings.Split(params, ";")
	for k := 0; k < len(fields); k++ {
		code, err := strconv.Atoi(strings.SplitN(fields[k], ":", 2)[0])
		if err != nil {
			code = 0
		}
		switch code {
		case 0, 22:
			dim = false
		case 2:
			dim = true
		case 38, 48, 58: // a color, its parameters after it unless given with colons
			if strings.Contains(fields[k], ":") || k+1 >= len(fields) {
				continue
			}
			if fields[k+1] == "5" {
				k += 2
			} else if fields[k+1] == "2" {
				k += 4
			}
		}
	}
	return dim
}
