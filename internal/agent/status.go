// Package agent recognizes coding agents running in terminals and reads
// their state from what they show. Status rules follow cc-mission-control.
package agent

import (
	"path/filepath"
	"regexp"
	"strings"
)

type Kind string

const (
	Claude Kind = "claude"
	Codex  Kind = "codex"
	Pi     Kind = "pi"
)

type Status string

const (
	Waiting Status = "waiting"
	Working Status = "working"
	Idle    Status = "idle"
)

// KindOf maps a foreground command, as ps reports it, to an agent.
func KindOf(command string) (Kind, bool) {
	kind := Kind(filepath.Base(strings.TrimSpace(command)))
	return kind, kind == Claude || kind == Codex || kind == Pi
}

const idleMarker = "✳"

var (
	claudeWaiting      = regexp.MustCompile(`❯\s+\d+\.`)
	codexActionTitle   = regexp.MustCompile(`^\[\s*[!.]\s*\]\s*Action Required(?:\s*\|.*)?\s*$`)
	codexWaitingCursor = regexp.MustCompile(`(?m)^\s*›\s+\d+\.`)
	codexWaitingFooter = regexp.MustCompile(`(?i)\benter to (?:submit answer|select|confirm)\b`)
	piWorking          = regexp.MustCompile(`(?m)^[\s─]*[⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏] Working(?:\.\.\.)?(?: \([^\n]* to interrupt\))?[\s─]*$`)
	titlePrefix        = regexp.MustCompile(`^(?:\[\s*.\s*\]\s*Action Required\s*\|\s*|[⠀-⣿✳◐◑]\s*)`)
)

// A live dialog is anchored to the bottom of the screen; a resolved one
// scrolls up, so only the tail counts.
const tailLines = 30

func tail(text string) string {
	lines := strings.Split(text, "\n")
	return strings.Join(lines[max(0, len(lines)-tailLines):], "\n")
}

func spinner(title string) bool {
	r := []rune(title)
	return len(r) > 0 && r[0] >= 0x2800 && r[0] <= 0x28ff
}

// DetectStatus reads an agent's status from its title and screen text.
func DetectStatus(kind Kind, title, screen string) Status {
	screen = tail(screen)
	if kind == Pi {
		if piWorking.MatchString(screen) {
			return Working
		}
		return Idle
	}
	if kind == Codex {
		if codexActionTitle.MatchString(title) || codexWaitingCursor.MatchString(screen) && codexWaitingFooter.MatchString(screen) {
			return Waiting
		}
		if spinner(title) {
			return Working
		}
		return Idle
	}
	if spinner(title) || strings.HasPrefix(title, "◐") || strings.HasPrefix(title, "◑") {
		return Working
	}
	if strings.HasPrefix(title, idleMarker) && claudeWaiting.MatchString(screen) {
		return Waiting
	}
	return Idle
}

// DisplayTitle drops the status prefixes agents put in their titles.
func DisplayTitle(title string) string {
	return strings.TrimSpace(titlePrefix.ReplaceAllString(strings.TrimSpace(title), ""))
}
