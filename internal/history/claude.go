package history

import (
	"cmp"
	"encoding/json"
	"path/filepath"
	"strings"
	"time"
)

// Claude Code keeps a session in projects/<its directory, encoded>/<ID>.jsonl
// under its configuration directory; subagents' sessions are a level down.
var claudeKind = kind{name: "claude", home: claudeHome, list: claudeList, read: claudeRead, resume: claudeResume}

// claudeHome is the configuration directory Claude Code uses by default:
// CLAUDE_CONFIG_DIR's, else ~/.claude.
func claudeHome() (string, error) { return home("CLAUDE_CONFIG_DIR", ".claude") }

// claudeResume resumes the session from its configuration directory, which
// Claude Code is told unless it is the one it uses by default: told one,
// it reads its state there rather than from ~/.claude.json.
func claudeResume(s Session) string {
	command := "claude --resume " + s.ID
	root := filepath.Dir(filepath.Dir(filepath.Dir(s.Path)))
	if def, err := claudeHome(); s.Path == "" || err == nil && filepath.Clean(def) == root {
		return command
	}
	return "CLAUDE_CONFIG_DIR=" + shellQuote(root) + " " + command
}

// claudeRecord is what Jun reads of a line of a Claude Code session.
type claudeRecord struct {
	Type             string          `json:"type"`
	CWD              string          `json:"cwd"`
	GitBranch        string          `json:"gitBranch"`
	Timestamp        time.Time       `json:"timestamp"`
	SessionID        string          `json:"sessionId"`
	IsMeta           bool            `json:"isMeta"`
	IsSidechain      bool            `json:"isSidechain"`
	IsCompactSummary bool            `json:"isCompactSummary"`
	AITitle          string          `json:"aiTitle"`
	CustomTitle      string          `json:"customTitle"`
	LastPrompt       string          `json:"lastPrompt"`
	Message          json.RawMessage `json:"message"`
}

// claudeEncode is the name Claude Code gives a directory's folder:
// every character but ASCII letters and digits a dash.
func claudeEncode(dir string) string {
	return strings.Map(func(r rune) rune {
		if r < 128 && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, dir)
}

func claudeList(root string, dirs []string, c *cache) ([]Session, error) {
	return folderSessions(filepath.Join(root, "projects"), dirs, c, claudeEncode, claudeSummary)
}

func claudeSummary(path string) (Session, bool, error) {
	s := Session{Agent: "claude", ID: strings.TrimSuffix(filepath.Base(path), ".jsonl")}
	var aiTitle, customTitle string
	note := func(r claudeRecord) {
		aiTitle, customTitle = cmp.Or(r.AITitle, aiTitle), cmp.Or(r.CustomTitle, customTitle)
	}
	err := head(path, func(line []byte) bool {
		var r claudeRecord
		if json.Unmarshal(line, &r) != nil {
			return true
		}
		note(r)
		if r.CWD != "" && s.CWD == "" {
			s.CWD, s.Branch, s.Start = r.CWD, r.GitBranch, r.Timestamp
		}
		if prompt, ok := claudePrompt(r); ok && s.FirstPrompt == "" {
			s.FirstPrompt = prompt
		}
		return s.CWD == "" || s.FirstPrompt == ""
	})
	if err != nil {
		return Session{}, false, err
	}
	lines, err := tail(path)
	if err != nil {
		return Session{}, false, err
	}
	for _, line := range lines {
		var r claudeRecord
		if json.Unmarshal(line, &r) != nil {
			continue
		}
		note(r)
		if prompt, ok := claudePrompt(r); ok {
			s.LastPrompt = prompt
		}
		s.LastPrompt = cmp.Or(r.LastPrompt, s.LastPrompt)
	}
	if s.FirstPrompt == "" && s.LastPrompt == "" {
		return s, false, nil // nothing was asked
	}
	s.Title = cmp.Or(customTitle, aiTitle, firstLine(s.FirstPrompt), firstLine(s.LastPrompt))
	return s, true, nil
}

// claudePrompt is the user's text in a record, unless the record is no
// prompt: meta, a subagent's, a summary, tools' results or a command's.
func claudePrompt(r claudeRecord) (string, bool) {
	if r.Type != "user" || r.IsMeta || r.IsSidechain || r.IsCompactSummary {
		return "", false
	}
	text := contentText(r.Message)
	if strings.TrimSpace(text) == "" || strings.HasPrefix(strings.TrimSpace(text), "<") {
		return "", false
	}
	return text, true
}

func claudeRead(path string) ([]Message, error) {
	var messages []Message
	steps := 0
	err := eachLine(path, func(line []byte) {
		var r claudeRecord
		if json.Unmarshal(line, &r) != nil || r.IsSidechain {
			return
		}
		if prompt, ok := claudePrompt(r); ok {
			messages = append(messages, Message{User: true, Text: prompt, Steps: steps})
			steps = 0
			return
		}
		if r.Type != "assistant" {
			return
		}
		var m struct {
			Content []textBlock `json:"content"`
		}
		if json.Unmarshal(r.Message, &m) != nil {
			return
		}
		for _, b := range m.Content {
			if b.Type == "tool_use" {
				steps++
			}
			if b.Type == "text" && strings.TrimSpace(b.Text) != "" {
				messages = append(messages, Message{Text: b.Text, Steps: steps})
				steps = 0
			}
		}
	})
	return messages, err
}
