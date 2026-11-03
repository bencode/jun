// Package history reads the sessions coding agents (Claude Code, Codex,
// Pi) keep on disk: which there are for some directories, and what was
// said.
package history

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
)

// Session is one agent session, as its file tells at its start and end.
type Session struct {
	Agent       string // a kind's name: "claude", "codex" or "pi"
	ID          string
	Path        string // the session's file
	Title       string // the agent's, or the first prompt's first line
	CWD         string
	Branch      string // checked out when the session started; "" when unknown
	Start       time.Time
	Updated     time.Time // the file's last change
	FirstPrompt string
	LastPrompt  string
}

// Message is a prompt or an answer of a session; tools' work is counted,
// not kept.
type Message struct {
	User  bool
	Text  string
	Steps int // tool steps between the message before and this one
}

// kind is an agent whose sessions Jun reads.
type kind struct {
	name string
	// home is the agent's directory unless others are given.
	home func() (string, error)
	// list gives the sessions of the agent's directory root in or under
	// dirs, summarizing changed files with c.
	list func(root string, dirs []string, c *cache) ([]Session, error)
	read func(path string) ([]Message, error)
	// resume is the shell command that resumes the session.
	resume func(s Session) string
}

var kinds = []kind{claudeKind, codexKind, piKind}

func kindOf(agent string) (kind, bool) {
	i := slices.IndexFunc(kinds, func(k kind) bool { return k.name == agent })
	if i < 0 {
		return kind{}, false
	}
	return kinds[i], true
}

// Known reports whether agent names a kind of session Jun reads.
func Known(agent string) bool {
	_, ok := kindOf(agent)
	return ok
}

// Scanner lists sessions, reading again only the files that changed
// since it last did.
type Scanner struct {
	c cache
}

// List gives every kind's sessions in or under dirs, the latest first,
// from the agent's directories homes names by its kind's name, else from
// its own. An agent with no sessions on this machine has none; other
// failures are returned with the sessions that could be read.
func (s *Scanner) List(dirs []string, homes map[string][]string) ([]Session, error) {
	var all []Session
	var errs []error
	for _, k := range kinds {
		roots, err := k.roots(homes[k.name])
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", k.name, err))
			continue
		}
		for _, root := range roots {
			sessions, err := k.list(root, cleanDirs(dirs), &s.c)
			all = append(all, sessions...)
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", k.name, err))
			}
		}
	}
	slices.SortFunc(all, func(a, b Session) int { return b.Updated.Compare(a.Updated) })
	return all, errors.Join(errs...)
}

// roots are the agent's directories to read: homes, each once, else its
// own.
func (k kind) roots(homes []string) ([]string, error) {
	if roots := slices.Compact(slices.Sorted(slices.Values(cleanDirs(homes)))); len(roots) > 0 {
		return roots, nil
	}
	root, err := k.home()
	if err != nil {
		return nil, err
	}
	return []string{root}, nil
}

// Read gives the session's prompts and answers, in order.
func Read(s Session) ([]Message, error) {
	k, ok := kindOf(s.Agent)
	if !ok {
		return nil, fmt.Errorf("unknown agent %q", s.Agent)
	}
	return k.read(s.Path)
}

var validID = regexp.MustCompile(`^[0-9A-Za-z][0-9A-Za-z-]*$`)

// ValidID reports whether id can be a session's: what a shell takes as
// one plain word.
func ValidID(id string) bool { return validID.MatchString(id) }

// ResumeCommand is the shell command that resumes the session.
func ResumeCommand(s Session) (string, error) {
	k, ok := kindOf(s.Agent)
	if !ok {
		return "", fmt.Errorf("unknown agent %q", s.Agent)
	}
	if !ValidID(s.ID) {
		return "", fmt.Errorf("invalid session ID %q", s.ID)
	}
	return k.resume(s), nil
}

// shellQuote is s as one word of a POSIX shell's command.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// cache keeps sessions summarized from files, while the files stay the
// same; ok is false for files that are no session to list.
type cache struct {
	mu      sync.Mutex
	entries map[string]cached
}

type cached struct {
	mod     time.Time
	size    int64
	session Session
	ok      bool
}

// summary gives the file's session, summarizing the file again only
// when it changed.
func (c *cache) summary(path string, info fs.FileInfo, summarize func() (Session, bool, error)) (Session, bool, error) {
	c.mu.Lock()
	e, hit := c.entries[path]
	c.mu.Unlock()
	if hit && e.mod.Equal(info.ModTime()) && e.size == info.Size() {
		return e.session, e.ok, nil
	}
	s, ok, err := summarize()
	if err != nil {
		return Session{}, false, err
	}
	s.Path, s.Updated = path, info.ModTime()
	c.mu.Lock()
	if c.entries == nil {
		c.entries = map[string]cached{}
	}
	c.entries[path] = cached{mod: info.ModTime(), size: info.Size(), session: s, ok: ok}
	c.mu.Unlock()
	return s, ok, nil
}

// folderSessions lists the sessions of base's folders, one file each,
// that are in or under dirs. A directory's sessions are in the folder
// encode names it; a folder's name tells only roughly whose it is, so
// its sessions say.
func folderSessions(base string, dirs []string, c *cache, encode func(dir string) string, summarize func(path string) (Session, bool, error)) ([]Session, error) {
	folders, err := os.ReadDir(base)
	if notInstalled(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	mayHold := func(name string) bool {
		return slices.ContainsFunc(dirs, func(d string) bool {
			e := encode(d)
			return name == e || strings.HasPrefix(name, e+"-")
		})
	}
	var sessions []Session
	for _, folder := range folders {
		if !folder.IsDir() || !mayHold(folder.Name()) {
			continue
		}
		files, err := os.ReadDir(filepath.Join(base, folder.Name()))
		if err != nil {
			return sessions, err
		}
		for _, f := range files {
			if f.IsDir() || filepath.Ext(f.Name()) != ".jsonl" {
				continue
			}
			path := filepath.Join(base, folder.Name(), f.Name())
			info, err := f.Info()
			if err != nil {
				return sessions, err
			}
			s, ok, err := c.summary(path, info, func() (Session, bool, error) { return summarize(path) })
			if err != nil {
				return sessions, err
			}
			if ok && within(s.CWD, dirs) {
				sessions = append(sessions, s)
			}
		}
	}
	return sessions, nil
}

// textBlock is a block of a message's content; Jun reads its type, and
// its text when it is text.
type textBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// contentText is a message's text: its content when that is a string,
// else its text blocks'.
func contentText(raw json.RawMessage) string {
	var m struct {
		Content json.RawMessage `json:"content"`
	}
	if json.Unmarshal(raw, &m) != nil {
		return ""
	}
	var text string
	if json.Unmarshal(m.Content, &text) == nil {
		return text
	}
	var blocks []textBlock
	if json.Unmarshal(m.Content, &blocks) != nil {
		return ""
	}
	var parts []string
	for _, b := range blocks {
		if b.Type == "text" && b.Text != "" {
			parts = append(parts, b.Text)
		}
	}
	return strings.Join(parts, "\n\n")
}

func cleanDirs(dirs []string) []string {
	clean := make([]string, 0, len(dirs))
	for _, d := range dirs {
		if d != "" {
			clean = append(clean, filepath.Clean(d))
		}
	}
	return clean
}

// within reports whether dir is one of dirs or under one.
func within(dir string, dirs []string) bool {
	if dir == "" {
		return false
	}
	dir = filepath.Clean(dir)
	return slices.ContainsFunc(dirs, func(d string) bool {
		return dir == d || strings.HasPrefix(dir, strings.TrimSuffix(d, string(filepath.Separator))+string(filepath.Separator))
	})
}

// headLimit is how much of a file's start a summary reads at most, and
// tailSize how much of its end.
const headLimit, tailSize = 4 << 20, 256 << 10

// head calls more with the file's first lines, until it returns false or
// headLimit bytes are read.
func head(path string, more func(line []byte) bool) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	r := bufio.NewReaderSize(io.LimitReader(f, headLimit), 1<<20)
	for {
		line, err := r.ReadBytes('\n')
		if err == io.EOF {
			return nil // a line the limit cut is left out
		}
		if err != nil {
			return err
		}
		if !more(line) {
			return nil
		}
	}
}

// tail reads the whole lines in the file's last tailSize bytes.
func tail(path string) ([][]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	at := max(info.Size()-tailSize, 0)
	buf := make([]byte, info.Size()-at)
	if _, err := f.ReadAt(buf, at); err != nil && err != io.EOF {
		return nil, err
	}
	lines := bytes.Split(buf, []byte("\n"))
	if at > 0 {
		lines = lines[1:] // the line cut
	}
	return lines, nil
}

// eachLine calls fn with each line of the file, of any length.
func eachLine(path string, fn func(line []byte)) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	r := bufio.NewReaderSize(f, 1<<20)
	for {
		line, err := r.ReadBytes('\n')
		if len(line) > 0 {
			fn(line)
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// firstLine is a prompt's first non-empty line, for a title.
func firstLine(text string) string {
	for line := range strings.Lines(text) {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return ""
}

// home is the directory an environment variable names, else one in the
// user's home directory.
func home(env, name string) (string, error) {
	if dir := os.Getenv(env); dir != "" {
		return dir, nil
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, name), nil
}

// notInstalled reports err as a missing directory: the agent never ran.
func notInstalled(err error) bool { return errors.Is(err, fs.ErrNotExist) }
