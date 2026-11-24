package goals

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/bencode/jun/internal/config"
)

// version is the goals file's format.
const version = 1

// Path is the goals file, beside Jun's configuration.
func Path(configPath string) string { return filepath.Join(filepath.Dir(configPath), "goals.json") }

type document struct {
	Version  int    `json:"version"`
	LastGoal int    `json:"lastGoal"`
	LastTask int    `json:"lastTask"`
	Goals    []Goal `json:"goals"`
	Tasks    []Task `json:"tasks"`
}

// Load reads the goals; none before the file is there. A file that does
// not hold as Check tells is an error.
func Load(path string) (Book, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Book{}, nil
	}
	if err != nil {
		return Book{}, err
	}
	var doc document
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&doc); err != nil {
		return Book{}, fmt.Errorf("decode %q: %w", path, err)
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return Book{}, fmt.Errorf("decode %q: more than one JSON document", path)
	}
	if doc.Version != version {
		return Book{}, fmt.Errorf("%q: version %d is not one Jun knows", path, doc.Version)
	}
	b := Book{LastGoal: doc.LastGoal, LastTask: doc.LastTask, Goals: doc.Goals, Tasks: doc.Tasks}
	if err := Check(b); err != nil {
		return Book{}, fmt.Errorf("%q: %w", path, err)
	}
	if len(b.Goals) == 0 { // "[]" reads as none
		b.Goals = nil
	}
	if len(b.Tasks) == 0 {
		b.Tasks = nil
	}
	return b, nil
}

// Save replaces the file with the book, whole: one cut short by a stop
// leaves the one before.
func Save(path string, b Book) error {
	if err := Check(b); err != nil {
		return err
	}
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "{\n  \"version\": %d,\n  \"lastGoal\": %d,\n  \"lastTask\": %d,\n  \"goals\": ", version, b.LastGoal, b.LastTask)
	if err := config.JSONLines(&buf, b.Goals); err != nil {
		return err
	}
	buf.WriteString(",\n  \"tasks\": ")
	if err := config.JSONLines(&buf, b.Tasks); err != nil {
		return err
	}
	buf.WriteString("\n}\n")
	return config.WriteFile(path, buf.Bytes())
}
