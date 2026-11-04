// Package tool lets programs in Jun's terminals, agents above all, ask the
// Jun that runs them about its terminals: `jun tool …` talks to Jun over a
// Unix socket, one JSON request and one JSON response a connection.
package tool

import (
	"fmt"
	"slices"
	"strings"
)

const (
	EnvAPI      = "JUN_API"      // the socket of the Jun that runs the terminal
	EnvTerminal = "JUN_TERMINAL" // the terminal's id, for reference: Jun does not trust it
	EnvBin      = "JUN_BIN"      // Jun's executable, whatever PATH and aliases say
)

const (
	DefaultLines = 40
	MaxLines     = 500
)

type Op string

const (
	OpTerminals Op = "terminals"
	OpScreen    Op = "screen"
)

type Request struct {
	Op       Op     `json:"op"`
	Terminal uint64 `json:"terminal,omitempty"` // screen
	Lines    int    `json:"lines,omitempty"`    // screen, 1 to MaxLines
}

type Response struct {
	Error     string     `json:"error,omitempty"`
	Terminals []Terminal `json:"terminals,omitempty"`
	Screen    string     `json:"screen,omitempty"`
}

type Terminal struct {
	ID      uint64 `json:"id"`
	Project string `json:"project"`
	Dir     string `json:"dir"`
	Title   string `json:"title"`
	Agent   string `json:"agent,omitempty"`  // claude, codex or pi
	Status  string `json:"status,omitempty"` // waiting, working or idle
	Self    bool   `json:"self"`             // the caller runs in it
}

// Env is what a terminal's environment gets, for its programs to reach Jun.
func Env(socket, bin string, terminal uint64) []string {
	return []string{EnvAPI + "=" + socket, EnvBin + "=" + bin, fmt.Sprintf("%s=%d", EnvTerminal, terminal)}
}

// Tail returns the last lines of text, after the blank ones at its end,
// as the screen has below the cursor.
func Tail(text string, lines int) string {
	all := strings.Split(text, "\n")
	end := len(all)
	for end > 0 && strings.TrimSpace(all[end-1]) == "" {
		end--
	}
	if end == 0 {
		return ""
	}
	return strings.Join(slices.Clone(all[max(0, end-lines):end]), "\n") + "\n"
}
