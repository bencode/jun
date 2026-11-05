package message

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAppendThenLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "messages.jsonl")
	if messages, _, err := Load(path); err != nil || len(messages) != 0 {
		t.Fatalf("no file: %v, %v", messages, err)
	}
	at := time.Date(2026, 11, 5, 12, 24, 0, 0, time.UTC)
	for _, text := range []string{"first", "two\nlines"} {
		if err := Append(path, Message{Time: at, Terminal: 13, Project: "jun", Text: text}); err != nil {
			t.Fatal(err)
		}
	}
	messages, _, err := Load(path)
	if err != nil || len(messages) != 2 {
		t.Fatalf("loaded %v, %v", messages, err)
	}
	if m := messages[1]; !m.Time.Equal(at) || m.Terminal != 13 || m.Project != "jun" || m.Text != "two\nlines" {
		t.Fatalf("read back %+v", m)
	}
}

func TestLoadKeepsTheLastAndCountsTheLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "messages.jsonl")
	for i := range Keep + 3 {
		if err := Append(path, Message{Time: time.Now(), Text: fmt.Sprint(i)}); err != nil {
			t.Fatal(err)
		}
	}
	messages, lines, err := Load(path)
	if err != nil || len(messages) != Keep || messages[0].Text != "3" || lines != Keep+3 {
		t.Fatalf("loaded %d of %d lines, from %v: %v", len(messages), lines, messages[0], err)
	}
	if err := Save(path, messages); err != nil {
		t.Fatal(err)
	}
	if _, lines, err := Load(path); err != nil || lines != Keep {
		t.Fatalf("saved %d lines: %v", lines, err)
	}
}

func TestLoadSkipsWhatACrashLeftAndTheNextLineSurvives(t *testing.T) {
	path := filepath.Join(t.TempDir(), "messages.jsonl")
	if err := Append(path, Message{Time: time.Now(), Text: "whole"}); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("null\n{\"text\":\"cut"); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if messages, _, err := Load(path); err != nil || len(messages) != 1 || messages[0].Text != "whole" {
		t.Fatalf("loaded %v, %v", messages, err)
	}
	if err := Append(path, Message{Time: time.Now(), Text: "next"}); err != nil {
		t.Fatal(err)
	}
	if messages, _, err := Load(path); err != nil || len(messages) != 2 || messages[1].Text != "next" {
		t.Fatalf("after the crash: %v, %v", messages, err)
	}
}
