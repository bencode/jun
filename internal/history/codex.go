package history

import (
	"bufio"
	"cmp"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// Codex keeps sessions by date, sessions/<year>/<month>/<day>/rollout-…-<ID>.jsonl
// under its home, each starting with a session_meta line; session_index.jsonl
// names them.
var codexKind = kind{name: "codex", list: codexList, read: codexRead, resume: "codex resume"}

type codexRecord struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type codexMeta struct {
	ID        string          `json:"id"`
	CWD       string          `json:"cwd"`
	Timestamp time.Time       `json:"timestamp"`
	Parent    string          `json:"parent_thread_id"`
	Source    json.RawMessage `json:"source"`
	Git       *struct {
		Branch string `json:"branch"`
	} `json:"git"`
}

type codexItem struct {
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

// codexSteps are the items that are a tool's work.
var codexSteps = []string{"function_call", "custom_tool_call", "local_shell_call", "web_search_call"}

func codexList(dirs []string, c *cache) ([]Session, error) {
	root, err := home("CODEX_HOME", ".codex")
	if err != nil {
		return nil, err
	}
	titles, err := codexTitles(filepath.Join(root, "session_index.jsonl"))
	if err != nil {
		return nil, err
	}
	var sessions []Session
	err = filepath.WalkDir(filepath.Join(root, "sessions"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".jsonl" {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		s, ok, err := c.summary(path, info, func() (Session, bool, error) { return codexSummary(path) })
		if err != nil {
			return err
		}
		if ok && within(s.CWD, dirs) {
			if title := titles[s.ID]; title != "" {
				s.Title = title
			}
			sessions = append(sessions, s)
		}
		return nil
	})
	if notInstalled(err) {
		return nil, nil
	}
	return sessions, err
}

// codexTitles names sessions by ID, as the index last did.
func codexTitles(path string) (map[string]string, error) {
	titles := map[string]string{}
	err := eachLine(path, func(line []byte) {
		var e struct {
			ID   string `json:"id"`
			Name string `json:"thread_name"`
		}
		if json.Unmarshal(line, &e) == nil && e.ID != "" && e.Name != "" {
			titles[e.ID] = e.Name
		}
	})
	if notInstalled(err) {
		return titles, nil
	}
	return titles, err
}

func codexSummary(path string) (Session, bool, error) {
	meta, err := codexReadMeta(path)
	if err != nil || meta == nil {
		return Session{}, false, err
	}
	if meta.Parent != "" || strings.Contains(string(meta.Source), `"subagent"`) {
		return Session{}, false, nil // a subagent's
	}
	s := Session{Agent: "codex", ID: meta.ID, CWD: meta.CWD, Start: meta.Timestamp}
	if meta.Git != nil {
		s.Branch = meta.Git.Branch
	}
	err = head(path, func(line []byte) bool {
		prompt, ok := codexPrompt(line)
		s.FirstPrompt = prompt
		return !ok
	})
	if err != nil {
		return Session{}, false, err
	}
	lines, err := tail(path)
	if err != nil {
		return Session{}, false, err
	}
	for _, line := range lines {
		if prompt, ok := codexPrompt(line); ok {
			s.LastPrompt = prompt
		}
	}
	if s.FirstPrompt == "" && s.LastPrompt == "" {
		return s, false, nil
	}
	s.Title = firstLine(cmp.Or(s.FirstPrompt, s.LastPrompt))
	return s, true, nil
}

// codexReadMeta reads the session_meta the file starts with, however
// long its line; nil when it starts otherwise.
func codexReadMeta(path string) (*codexMeta, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	line, err := bufio.NewReader(f).ReadBytes('\n')
	if err != nil && err != io.EOF {
		return nil, err
	}
	var r codexRecord
	if json.Unmarshal(line, &r) != nil || r.Type != "session_meta" {
		return nil, nil
	}
	var m codexMeta
	if json.Unmarshal(r.Payload, &m) != nil || m.ID == "" {
		return nil, nil
	}
	return &m, nil
}

// codexItemOf is the line's response item, if it is one.
func codexItemOf(line []byte) (codexItem, bool) {
	var r codexRecord
	if json.Unmarshal(line, &r) != nil || r.Type != "response_item" {
		return codexItem{}, false
	}
	var item codexItem
	return item, json.Unmarshal(r.Payload, &item) == nil
}

func (item codexItem) text() string {
	var parts []string
	for _, c := range item.Content {
		if strings.TrimSpace(c.Text) != "" {
			parts = append(parts, c.Text)
		}
	}
	return strings.Join(parts, "\n\n")
}

// codexPrompt is the user's text on the line, unless it is what Codex
// adds: instructions, the environment, earlier history.
func codexPrompt(line []byte) (string, bool) {
	item, ok := codexItemOf(line)
	if !ok || item.Type != "message" || item.Role != "user" {
		return "", false
	}
	text := item.text()
	trimmed := strings.TrimSpace(text)
	if trimmed == "" || strings.HasPrefix(trimmed, "<") || strings.HasPrefix(trimmed, "# AGENTS.md") {
		return "", false
	}
	return text, true
}

func codexRead(path string) ([]Message, error) {
	var messages []Message
	steps := 0
	err := eachLine(path, func(line []byte) {
		if prompt, ok := codexPrompt(line); ok {
			messages = append(messages, Message{User: true, Text: prompt, Steps: steps})
			steps = 0
			return
		}
		item, ok := codexItemOf(line)
		if !ok {
			return
		}
		if item.Type == "message" && item.Role == "assistant" {
			if text := item.text(); text != "" {
				messages = append(messages, Message{Text: text, Steps: steps})
				steps = 0
			}
			return
		}
		if slices.Contains(codexSteps, item.Type) {
			steps++
		}
	})
	return messages, err
}
