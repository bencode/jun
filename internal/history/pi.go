package history

import (
	"cmp"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// Pi keeps a session in --<its directory, encoded>--/<time>_<ID>.jsonl
// under its sessions directory. The file starts with a session line; its
// entries form a tree by id and parentId, a branch for each time the
// session went back to an earlier point, the latest entry's the current.
var piKind = kind{
	name:   "pi",
	home:   piHome,
	list:   piList,
	read:   piRead,
	resume: func(s Session) string { return "pi --session " + shellQuote(s.Path) },
}

// piHome is the sessions directory Pi uses by default:
// PI_CODING_AGENT_SESSION_DIR, else sessions in PI_CODING_AGENT_DIR's,
// else in ~/.pi/agent.
func piHome() (string, error) {
	if dir := os.Getenv("PI_CODING_AGENT_SESSION_DIR"); dir != "" {
		return dir, nil
	}
	agent, err := home("PI_CODING_AGENT_DIR", filepath.Join(".pi", "agent"))
	if err != nil {
		return "", err
	}
	return filepath.Join(agent, "sessions"), nil
}

// piRecord is what Jun reads of a line of a Pi session.
type piRecord struct {
	Type      string          `json:"type"`
	ID        string          `json:"id"`
	ParentID  string          `json:"parentId"`
	CWD       string          `json:"cwd"`
	Timestamp time.Time       `json:"timestamp"`
	Name      string          `json:"name"`
	Message   json.RawMessage `json:"message"`
}

// piEncode is the start of the name Pi gives a directory's folder: two
// dashes, then the directory without its leading separator, each
// separator and colon a dash. Two more dashes end the name.
func piEncode(dir string) string {
	return "--" + strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ':' {
			return '-'
		}
		return r
	}, strings.TrimPrefix(dir, "/"))
}

func piList(root string, dirs []string, c *cache) ([]Session, error) {
	return folderSessions(root, dirs, c, piEncode, piSummary)
}

func piSummary(path string) (Session, bool, error) {
	var s Session
	var name string
	first := true
	err := head(path, func(line []byte) bool {
		var r piRecord
		ok := json.Unmarshal(line, &r) == nil
		if first {
			first = false
			if !ok || r.Type != "session" || r.ID == "" {
				return false
			}
			s = Session{Agent: "pi", ID: r.ID, CWD: r.CWD, Start: r.Timestamp}
			return true
		}
		if ok {
			if r.Type == "session_info" {
				name = cmp.Or(r.Name, name)
			}
			if prompt, isPrompt := piPrompt(r); isPrompt {
				s.FirstPrompt = prompt
				return false
			}
		}
		return true
	})
	if err != nil || s.ID == "" {
		return Session{}, false, err
	}
	lines, err := tail(path)
	if err != nil {
		return Session{}, false, err
	}
	for _, line := range lines {
		var r piRecord
		if json.Unmarshal(line, &r) != nil {
			continue
		}
		if r.Type == "session_info" {
			name = cmp.Or(r.Name, name)
		}
		if prompt, ok := piPrompt(r); ok {
			s.LastPrompt = prompt
		}
	}
	if s.FirstPrompt == "" && s.LastPrompt == "" {
		return s, false, nil // nothing was asked
	}
	s.Title = cmp.Or(name, firstLine(s.FirstPrompt), firstLine(s.LastPrompt))
	return s, true, nil
}

// piRole is the role of the record's message, "" when it is none.
func piRole(r piRecord) string {
	if r.Type != "message" {
		return ""
	}
	var m struct {
		Role string `json:"role"`
	}
	if json.Unmarshal(r.Message, &m) != nil {
		return ""
	}
	return m.Role
}

// piPrompt is the user's text in a record, if it is a prompt.
func piPrompt(r piRecord) (string, bool) {
	if piRole(r) != "user" {
		return "", false
	}
	text := contentText(r.Message)
	return text, strings.TrimSpace(text) != ""
}

// piBranch is the current branch's entries, from the root: the latest
// entry and its ancestors.
func piBranch(entries []piRecord) []piRecord {
	byID := make(map[string]int, len(entries))
	for i, e := range entries {
		byID[e.ID] = i
	}
	var branch []piRecord
	i, ok := len(entries)-1, len(entries) > 0
	for ok && len(branch) < len(entries) { // the bound stops at a cycle
		branch = append(branch, entries[i])
		i, ok = byID[entries[i].ParentID]
	}
	slices.Reverse(branch)
	return branch
}

func piRead(path string) ([]Message, error) {
	var entries []piRecord
	err := eachLine(path, func(line []byte) {
		var r piRecord
		if json.Unmarshal(line, &r) == nil && r.ID != "" && r.Type != "session" {
			entries = append(entries, r)
		}
	})
	if err != nil {
		return nil, err
	}
	var messages []Message
	steps := 0
	for _, r := range piBranch(entries) {
		if prompt, ok := piPrompt(r); ok {
			messages = append(messages, Message{User: true, Text: prompt, Steps: steps})
			steps = 0
			continue
		}
		if piRole(r) != "assistant" {
			continue
		}
		var m struct {
			Content []textBlock `json:"content"`
		}
		if json.Unmarshal(r.Message, &m) != nil {
			continue
		}
		for _, b := range m.Content {
			if b.Type == "toolCall" {
				steps++
			}
			if b.Type == "text" && strings.TrimSpace(b.Text) != "" {
				messages = append(messages, Message{Text: b.Text, Steps: steps})
				steps = 0
			}
		}
	}
	return messages, nil
}
