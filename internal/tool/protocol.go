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
	EnvTerminal = "JUN_TERMINAL" // the terminal's id, for a caller outside Jun's terminals
	EnvBin      = "JUN_BIN"      // Jun's executable, whatever PATH and aliases say
)

const (
	DefaultLines = 40
	MaxLines     = 500
	MaxText      = 1 << 20 // the bytes send takes
	MaxMessage   = 4096    // the bytes message takes
	MaxTitle     = 200     // the bytes a message's title takes
	MaxReplies   = 50      // the replies a message keeps, the latest
	MaxName      = 64      // the bytes a terminal's name takes
	MaxTimerText = 500     // the bytes a terminal's timer text takes
)

type Op string

const (
	OpTerminals Op = "terminals"
	OpScreen    Op = "screen"
	OpOpen      Op = "open"
	OpClose     Op = "close"
	OpBack      Op = "back"
	OpForward   Op = "forward"
	OpSend      Op = "send"
	OpMessage   Op = "message"  // to the user, in Global
	OpMessages  Op = "messages" // lists them, the latest first
	// OpUpdateMessage adds a reply to the message of Request.Message, or
	// changes its kind, or both.
	OpUpdateMessage Op = "update-message"
	OpRun           Op = "run"
	OpLoop          Op = "loop"        // registers one
	OpLoops         Op = "loops"       // lists them
	OpCancelLoop    Op = "cancel-loop" // the one of Request.Terminal
	OpWatch         Op = "watch"       // answers with events until the connection ends
	OpJoin          Op = "join"        // makes the terminal of Request.Terminal the caller's lead
	OpLeave         Op = "leave"       // drops the caller's lead
	// OpUpdateTerminal names the terminal of Request.Terminal Request.Name,
	// or drops its name for none; sets its timer, from Request.Every,
	// Request.At and Request.Text, or drops it with Request.NoTimer.
	OpUpdateTerminal Op = "update-terminal"
	// OpRoles lists the roles, or with Request.Name, that role's sessions;
	// with Request.Mine, the role of the caller's session, if any.
	OpRoles Op = "roles"
)

// Sessions are the ids of the agent sessions a command runs in, from the
// variables agents give their commands: a name given from there records
// the session as the role's.
type Sessions struct {
	Claude string `json:"claude,omitempty"` // CLAUDE_CODE_SESSION_ID
	Codex  string `json:"codex,omitempty"`  // CODEX_THREAD_ID
	Pi     string `json:"pi,omitempty"`     // PI_SESSION_ID
}

// SessionsFrom reads the sessions' variables.
func SessionsFrom(getenv func(string) string) Sessions {
	return Sessions{Claude: getenv("CLAUDE_CODE_SESSION_ID"), Codex: getenv("CODEX_THREAD_ID"), Pi: getenv("PI_SESSION_ID")}
}

// Of is the agent's session id; none when the command runs in none of it.
func (s Sessions) Of(agent string) string {
	return map[string]string{"claude": s.Claude, "codex": s.Codex, "pi": s.Pi}[agent]
}

// DefaultRoleSessions is how many of a role's sessions roles <name> gives,
// the latest.
const DefaultRoleSessions = 20

// MinEvery is the shortest interval a loop or a timer takes.
const MinEvery = time.Minute

type TargetKind string

const (
	TargetTerminal    TargetKind = "terminal"    // by ID
	TargetDiff        TargetKind = "diff"        // the changes of the branch a terminal, by ID, is on
	TargetProject     TargetKind = "project"     // by Name
	TargetPerspective TargetKind = "perspective" // by Name, in the current project
	TargetFile        TargetKind = "file"        // by Name, an absolute path
	TargetPeek        TargetKind = "peek"        // closing only
)

// Target is what open and close act on.
type Target struct {
	Kind TargetKind `json:"kind"`
	ID   uint64     `json:"id,omitempty"`
	Name string     `json:"name,omitempty"`
}

// String is the target as the commands take it, as terminal:7.
func (t Target) String() string {
	if t.Kind == TargetPeek {
		return string(t.Kind)
	}
	if t.Kind == TargetTerminal || t.Kind == TargetDiff {
		return fmt.Sprintf("%s:%d", t.Kind, t.ID)
	}
	return fmt.Sprintf("%s:%s", t.Kind, t.Name)
}

// MessageKind tells at a glance what a message asks of the user.
type MessageKind string

const (
	MessageInfo MessageKind = "info"
	MessageAsk  MessageKind = "ask" // the user has something to do
	MessageWarn MessageKind = "warn"
	MessageDone MessageKind = "done"
)

var MessageKinds = []MessageKind{MessageAsk, MessageWarn, MessageDone, MessageInfo}

// Reply is what was added to a message later.
type Reply struct {
	Time time.Time `json:"time"`
	Text string    `json:"text"`
}

// Message is a message as `jun tool message` lists it.
type Message struct {
	ID      uint64      `json:"id"`
	Time    time.Time   `json:"time"`
	Kind    MessageKind `json:"kind"`
	Title   string      `json:"title,omitempty"`
	Text    string      `json:"text,omitempty"`
	Project string      `json:"project,omitempty"`
	Link    string      `json:"link,omitempty"` // as --link takes it
	Replies []Reply     `json:"replies,omitempty"`
}

type Request struct {
	Op Op `json:"op"`
	// Caller is the terminal the caller names as its own, from JUN_TERMINAL.
	// Jun knows a caller by its processes, and takes this only for one in
	// none of its terminals, such as a command Codex's shared server runs.
	Caller   uint64  `json:"caller,omitempty"`
	Terminal uint64  `json:"terminal,omitempty"` // screen
	Lines    int     `json:"lines,omitempty"`    // screen, 1 to MaxLines
	Target   *Target `json:"target,omitempty"`   // open, close, what a message links to
	// In is the perspective a Diff opens in, of the terminal's project:
	// selected, or made when there is none.
	In    string      `json:"in,omitempty"`
	Peek  bool        `json:"peek,omitempty"`  // open terminal: in the peek drawer
	Text  string      `json:"text,omitempty"`  // send: to submit, its line breaks kept; message: to show; update-terminal: the timer's
	Title string      `json:"title,omitempty"` // message
	Kind  MessageKind `json:"kind,omitempty"`  // message, none being MessageInfo; update-message, none leaving it
	// Message is the ID of the message update-message changes.
	Message uint64 `json:"message,omitempty"`
	// Command is the shell command line run and loop run, in Dir, an
	// absolute path.
	Command string        `json:"command,omitempty"`
	Dir     string        `json:"dir,omitempty"`
	Open    bool          `json:"open,omitempty"`    // run: in a tab, focused
	Project string        `json:"project,omitempty"` // run: the project, else the caller's; an empty Dir is its directory
	Every   time.Duration `json:"every,omitempty"`   // loop, a timer: at least MinEvery, in whole seconds
	Name    *string       `json:"name,omitempty"`    // update-terminal: the terminal's name, empty to drop it; roles: the role
	// Sessions are the caller's: update-terminal records the one of the
	// terminal's agent, when the terminal is the caller's.
	Sessions Sessions  `json:"sessions,omitzero"`
	Limit    int       `json:"limit,omitempty"`   // roles <name>: at most this many sessions
	Mine     bool      `json:"mine,omitempty"`    // roles: the role of the caller's session, from Sessions
	At       time.Time `json:"at,omitzero"`       // update-terminal: when a timer first fires
	NoTimer  bool      `json:"noTimer,omitempty"` // update-terminal: drops the terminal's timer
}

type Response struct {
	Error     string     `json:"error,omitempty"`
	Terminals []Terminal `json:"terminals,omitempty"`
	Screen    string     `json:"screen,omitempty"`
	Message   string     `json:"message,omitempty"`  // what a change done leaves pending, such as the user's confirmation
	Terminal  uint64     `json:"terminal,omitempty"` // run, loop: the new terminal, which is the loop's id
	Loops     []Loop     `json:"loops,omitempty"`
	Created   uint64     `json:"created,omitempty"` // message: the new message's ID
	Messages  []Message  `json:"messages,omitempty"`
	Roles     []Role     `json:"roles,omitempty"`
}

// Role is a name that lasts, as its sessions are its memory.
type Role struct {
	Name     string `json:"name"`
	Terminal uint64 `json:"terminal,omitempty"` // the running terminal of that name, if any
	Count    int    `json:"count"`              // its sessions
	// Last is its latest session; Sessions, for roles <name>, the latest
	// ones, first.
	Last     *RoleSession  `json:"last,omitempty"`
	Sessions []RoleSession `json:"sessions,omitempty"`
}

// RoleSession is one of a role's sessions.
type RoleSession struct {
	Agent string    `json:"agent"` // claude, codex or pi
	ID    string    `json:"id"`
	Dir   string    `json:"dir"`
	At    time.Time `json:"at"` // when it took the role
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
	ID      uint64 `json:"id"`             // its terminal's
	Name    string `json:"name,omitempty"` // as terminals has it
	Project string `json:"project"`
	Agent   string `json:"agent"`
	Title   string `json:"title"`
	From    string `json:"from"`           // empty: the agent is new
	To      string `json:"to"`             // empty: the agent is gone
	Lead    uint64 `json:"lead,omitempty"` // as terminals has it
}

type Terminal struct {
	ID      uint64 `json:"id"`
	Name    string `json:"name,omitempty"` // what agents call it, given with terminals update
	Project string `json:"project"`
	Dir     string `json:"dir"`
	Title   string `json:"title"`
	Agent   string `json:"agent,omitempty"`  // claude, codex or pi
	Status  string `json:"status,omitempty"` // waiting, working or idle
	Self    bool   `json:"self"`             // the caller runs in it
	// Lead is the terminal of the agent that leads it: the agent that ran
	// it, or the one it joined. None once that terminal ends.
	Lead  uint64 `json:"lead,omitempty"`
	Timer *Timer `json:"timer,omitempty"` // set with terminals update
}

// Timer is when Jun next signals the agent in a terminal, and how.
type Timer struct {
	Every string    `json:"every,omitempty"` // as loop's; none: it fires once
	Next  time.Time `json:"next"`
	Text  string    `json:"text,omitempty"`
}

// EnvCommand prints a terminal's variables, for the user to run in it when a
// program outside it needs them.
const EnvCommand = `printf 'JUN_API=%s\nJUN_BIN=%s\nJUN_TERMINAL=%s\n' "$JUN_API" "$JUN_BIN" "$JUN_TERMINAL"`

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
