// Package message keeps what agents tell the user: a file of messages, one
// JSON object a line, oldest first.
package message

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Keep is how many messages Jun shows and loads. The file holds more until
// Jun cuts it back with Save.
const Keep = 500

type Message struct {
	Time     time.Time `json:"time"`
	Terminal uint64    `json:"terminal"` // what it is about: its sender, or another terminal
	Project  string    `json:"project"`  // that terminal's
	Text     string    `json:"text"`
	// Live tells a message of this run of Jun: terminal ids start over with
	// each run, so an earlier message's Terminal is no longer its terminal.
	Live bool `json:"-"`
}

// Load reads the file, oldest first: the last Keep, and how many lines it
// holds. No file is no messages. A line it cannot read, as a crash leaves,
// is logged and skipped.
func Load(path string) ([]Message, int, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}
	var messages []Message
	lines := 0
	for line := range bytes.Lines(data) {
		lines++
		var m Message
		err := json.Unmarshal(line, &m)
		if err == nil && m.Time.IsZero() {
			err = errors.New("it has no time")
		}
		if err != nil {
			log.Printf("Jun: skip a message of %s: %v", path, err)
			continue
		}
		messages = append(messages, m)
	}
	return messages[max(0, len(messages)-Keep):], lines, nil
}

// Append adds a line to the file, on a line of its own after one a crash
// cut.
func Append(path string, m Message) (err error) {
	line, err := encode(m)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if size := info.Size(); size > 0 {
		last := make([]byte, 1)
		if _, err := file.ReadAt(last, size-1); err != nil {
			return err
		}
		if last[0] != '\n' {
			line = append([]byte{'\n'}, line...)
		}
	}
	_, err = file.Write(line)
	return err
}

func encode(m Message) ([]byte, error) {
	line, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("encode message: %w", err)
	}
	return append(line, '\n'), nil
}

// Save replaces the file with the messages, at once, so that a crash leaves
// the old one.
func Save(path string, messages []Message) error {
	var data []byte
	for _, m := range messages {
		line, err := encode(m)
		if err != nil {
			return err
		}
		data = append(data, line...)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".messages-*.tmp")
	if err != nil {
		return fmt.Errorf("save messages: %w", err)
	}
	_, err = temporary.Write(data)
	if err == nil {
		err = temporary.Sync()
	}
	if err = errors.Join(err, temporary.Close()); err == nil {
		err = os.Rename(temporary.Name(), path)
	}
	if err != nil {
		return errors.Join(fmt.Errorf("save messages: %w", err), os.Remove(temporary.Name()))
	}
	return nil
}
