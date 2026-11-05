// Package tool lets programs in Jun's terminals, agents above all, ask the
// Jun that runs them about its terminals and change what it shows:
// `jun tool …` talks to Jun over a Unix socket, one JSON request and one
// JSON response a connection, or a stream of them for watch.
package tool

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

const (
	EnvAPI      = "JUN_API"      // the socket of the Jun that runs the terminal
	EnvTerminal = "JUN_TERMINAL" // the terminal's id, for reference: Jun does not trust it
	EnvBin      = "JUN_BIN"      // Jun's executable, whatever PATH and aliases say
)

const (
	DefaultLines = 40
	MaxLines     = 500
	MaxText      = 1 << 20 // the bytes send takes
)

type Op string

const (
	OpTerminals  Op = "terminals"
	OpScreen     Op = "screen"
	OpOpen       Op = "open"
	OpClose      Op = "close"
	OpBack       Op = "back"
	OpForward    Op = "forward"
	OpSend       Op = "send"
	OpRun        Op = "run"
	OpLoop       Op = "loop"        // registers one
	OpLoops      Op = "loops"       // lists them
	OpCancelLoop Op = "cancel-loop" // the one of Request.Terminal
	OpWatch      Op = "watch"       // answers with events until the connection ends
)

// MinEvery is the shortest interval a loop takes.
const MinEvery = time.Minute

type TargetKind string

const (
	TargetTerminal    TargetKind = "terminal"    // by ID
	TargetDiff        TargetKind = "diff"        // the changes of the branch a terminal, by ID, is on
	TargetProject     TargetKind = "project"     // by Name
	TargetPerspective TargetKind = "perspective" // by Name, in the current project
	TargetPeek        TargetKind = "peek"        // closing only
)

// Target is what open and close act on.
type Target struct {
	Kind TargetKind `json:"kind"`
	ID   uint64     `json:"id,omitempty"`
	Name string     `json:"name,omitempty"`
}

type Request struct {
	Op       Op      `json:"op"`
	Terminal uint64  `json:"terminal,omitempty"` // screen
	Lines    int     `json:"lines,omitempty"`    // screen, 1 to MaxLines
	Target   *Target `json:"target,omitempty"`   // open, close
	// In is the perspective a Diff opens in, of the terminal's project:
	// selected, or made when there is none.
	In   string `json:"in,omitempty"`
	Peek bool   `json:"peek,omitempty"` // open terminal: in the peek drawer
	Text string `json:"text,omitempty"` // send: to submit, its line breaks kept
	// Command is the shell command line run and loop run, in Dir, an
	// absolute path.
	Command string        `json:"command,omitempty"`
	Dir     string        `json:"dir,omitempty"`
	Open    bool          `json:"open,omitempty"`  // run: in a tab, focused
	Every   time.Duration `json:"every,omitempty"` // loop: at least MinEvery, in whole seconds
}

type Response struct {
	Error     string     `json:"error,omitempty"`
	Terminals []Terminal `json:"terminals,omitempty"`
	Screen    string     `json:"screen,omitempty"`
	Message   string     `json:"message,omitempty"`  // what a change done leaves pending, such as the user's confirmation
	Terminal  uint64     `json:"terminal,omitempty"` // run, loop: the new terminal, which is the loop's id
	Loops     []Loop     `json:"loops,omitempty"`
}

// Loop is a command Jun runs again and again in a terminal of its own.
type Loop struct {
	ID      uint64 `json:"id"` // its terminal's
	Every   string `json:"every"`
	Command string `json:"command"`
	Project string `json:"project"`
	Dir     string `json:"dir"`
}

// Event is an agent's status changing, as watch tells it. A watch starts
// with one for each agent, From empty.
type Event struct {
	ID      uint64 `json:"id"` // its terminal's
	Project string `json:"project"`
	Agent   string `json:"agent"`
	Title   string `json:"title"`
	From    string `json:"from"` // empty: the agent is new
	To      string `json:"to"`   // empty: the agent is gone
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
