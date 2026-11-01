package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// writeLines writes records as a JSONL file, changed at mod.
func writeLines(t *testing.T, path string, mod time.Time, records ...any) {
	t.Helper()
	var b strings.Builder
	for _, r := range records {
		line, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		b.Write(line)
		b.WriteByte('\n')
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, mod, mod); err != nil {
		t.Fatal(err)
	}
}

type m = map[string]any

func claudeUser(cwd, text string, extra m) m {
	r := m{"type": "user", "cwd": cwd, "gitBranch": "main", "timestamp": "2026-10-30T08:00:00Z", "message": m{"role": "user", "content": text}}
	for k, v := range extra {
		r[k] = v
	}
	return r
}

func claudeAssistant(blocks ...m) m {
	return m{"type": "assistant", "message": m{"role": "assistant", "content": blocks}}
}

func codexMetaLine(id, cwd string, extra m) m {
	p := m{"id": id, "cwd": cwd, "timestamp": "2026-10-30T09:00:00Z", "git": m{"branch": "feat/x"}, "base_instructions": strings.Repeat("x", 300<<10)}
	for k, v := range extra {
		p[k] = v
	}
	return m{"type": "session_meta", "payload": p}
}

func codexMessage(role, text string) m {
	kind := map[string]string{"user": "input_text", "assistant": "output_text"}[role]
	return m{"type": "response_item", "payload": m{"type": "message", "role": role, "content": []m{{"type": kind, "text": text}}}}
}

// agentHomes points both agents at empty directories of the test's.
func agentHomes(t *testing.T) (claude, codex string) {
	t.Helper()
	claude, codex = t.TempDir(), t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", claude)
	t.Setenv("CODEX_HOME", codex)
	return claude, codex
}

func TestListFindsTheDirectoriesSessionsAndHidesOthers(t *testing.T) {
	claude, codex := agentHomes(t)
	project := "/work/jun"
	worktree := "/work/jun/priv/worktrees/x"
	day := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	folder := filepath.Join(claude, "projects", claudeEncode(project))
	writeLines(t, filepath.Join(folder, "c1.jsonl"), day.Add(3*time.Hour),
		m{"type": "permission-mode", "permissionMode": "auto"},
		claudeUser(project, "<command-name>/model</command-name>", nil),
		claudeUser(project, "Plan the sessions list", nil),
		m{"type": "ai-title", "aiTitle": "Sessions plan"},
		claudeUser(project, "Ship it", nil))
	writeLines(t, filepath.Join(claude, "projects", claudeEncode(worktree), "c2.jsonl"), day.Add(1*time.Hour),
		claudeUser(worktree, "Fix the zoom", nil))
	// Not listed: a subagent's, one where nothing was asked, another project's.
	writeLines(t, filepath.Join(folder, "c1", "subagents", "agent-1.jsonl"), day, claudeUser(project, "Look around", m{"isSidechain": true}))
	writeLines(t, filepath.Join(folder, "c3.jsonl"), day, claudeUser(project, "Caveat", m{"isMeta": true}))
	writeLines(t, filepath.Join(claude, "projects", claudeEncode("/work/junk"), "c4.jsonl"), day, claudeUser("/work/junk", "Other", nil))

	sessions := filepath.Join(codex, "sessions", "2026", "10", "30")
	writeLines(t, filepath.Join(sessions, "rollout-a-x1.jsonl"), day.Add(2*time.Hour),
		codexMetaLine("x1", worktree, nil),
		codexMessage("user", "# AGENTS.md instructions for /work/jun"),
		codexMessage("user", "<environment_context>cwd</environment_context>"),
		codexMessage("user", "Review the diff"),
		codexMessage("assistant", "Looks fine."))
	writeLines(t, filepath.Join(sessions, "rollout-b-x2.jsonl"), day,
		codexMetaLine("x2", project, m{"parent_thread_id": "x1", "source": m{"subagent": m{"other": "guardian"}}}),
		codexMessage("user", "Judge this"))
	writeLines(t, filepath.Join(codex, "session_index.jsonl"), day,
		m{"id": "x1", "thread_name": "Old name"}, m{"id": "x1", "thread_name": "Diff review"})

	var s Scanner
	got, err := s.List([]string{project, "/work/jun/"})
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, len(got))
	for i, session := range got {
		ids[i] = session.ID
	}
	if !slices.Equal(ids, []string{"c1", "x1", "c2"}) {
		t.Fatalf("listed %v, want the latest first, others hidden", ids)
	}
	c1, x1, c2 := got[0], got[1], got[2]
	if c1.Title != "Sessions plan" || c1.FirstPrompt != "Plan the sessions list" || c1.LastPrompt != "Ship it" || c1.Branch != "main" {
		t.Fatalf("Claude session read as %+v", c1)
	}
	if x1.Agent != "codex" || x1.Title != "Diff review" || x1.FirstPrompt != "Review the diff" || x1.CWD != worktree || x1.Branch != "feat/x" {
		t.Fatalf("Codex session read as %+v", x1)
	}
	if c2.Title != "Fix the zoom" {
		t.Fatalf("an untitled session is titled %q, not by its prompt", c2.Title)
	}

	// Read again only as files change.
	writeLines(t, filepath.Join(folder, "c1.jsonl"), day.Add(5*time.Hour),
		claudeUser(project, "Plan the sessions list", nil), m{"type": "custom-title", "customTitle": "Renamed"})
	if got, err = s.List([]string{project}); err != nil || got[0].Title != "Renamed" {
		t.Fatalf("a changed session lists as %+v, %v", got[0], err)
	}
}

func TestReadKeepsPromptsAndAnswersAndCountsSteps(t *testing.T) {
	claude, codex := agentHomes(t)
	day := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	path := filepath.Join(claude, "projects", "-work", "c1.jsonl")
	writeLines(t, path, day,
		claudeUser("/work", "Find the bug", nil),
		claudeAssistant(m{"type": "thinking", "thinking": "hmm"}, m{"type": "tool_use", "name": "Bash"}),
		m{"type": "user", "message": m{"role": "user", "content": []m{{"type": "tool_result", "content": "ok"}}}},
		claudeAssistant(m{"type": "tool_use", "name": "Read"}),
		claudeAssistant(m{"type": "text", "text": "Found it."}),
		claudeUser("/work", "This session is being continued", m{"isCompactSummary": true}),
		claudeUser("/work", "Fix it", nil))
	want := []Message{{User: true, Text: "Find the bug"}, {Text: "Found it.", Steps: 2}, {User: true, Text: "Fix it"}}
	if got, err := Read(Session{Agent: "claude", Path: path}); err != nil || !slices.Equal(got, want) {
		t.Fatalf("Claude session read as %+v, %v", got, err)
	}

	path = filepath.Join(codex, "sessions", "rollout-x1.jsonl")
	writeLines(t, path, day,
		codexMetaLine("x1", "/work", nil),
		codexMessage("user", "<environment_context>cwd</environment_context>"),
		codexMessage("user", "Review the diff"),
		m{"type": "response_item", "payload": m{"type": "function_call", "name": "shell"}},
		m{"type": "response_item", "payload": m{"type": "reasoning"}},
		codexMessage("assistant", "Looks fine."))
	want = []Message{{User: true, Text: "Review the diff"}, {Text: "Looks fine.", Steps: 1}}
	if got, err := Read(Session{Agent: "codex", Path: path}); err != nil || !slices.Equal(got, want) {
		t.Fatalf("Codex session read as %+v, %v", got, err)
	}
}

func TestResumeCommandTakesOnlyPlainIDs(t *testing.T) {
	if got, err := ResumeCommand("claude", "16f0057a-2970"); err != nil || got != "claude --resume 16f0057a-2970" {
		t.Fatalf("Claude resumes with %q, %v", got, err)
	}
	if got, err := ResumeCommand("codex", "01a19689-cb83"); err != nil || got != "codex resume 01a19689-cb83" {
		t.Fatalf("Codex resumes with %q, %v", got, err)
	}
	for _, id := range []string{"", "x; rm -rf ~", "$(id)", "-h"} {
		if _, err := ResumeCommand("claude", id); err == nil {
			t.Fatalf("ID %q was taken", id)
		}
	}
}
