package roles

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

// version is the roles file's format.
const version = 1

// Path is the roles file, beside Jun's configuration.
func Path(configPath string) string { return filepath.Join(filepath.Dir(configPath), "roles.json") }

type document struct {
	Version  int       `json:"version"`
	Roles    []Role    `json:"roles"`
	Sessions []Session `json:"sessions"`
}

// Load reads the roles; none before the file is there. A file that does
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
	b := Book{Roles: doc.Roles, Sessions: doc.Sessions}
	if err := Check(b); err != nil {
		return Book{}, fmt.Errorf("%q: %w", path, err)
	}
	if len(b.Roles) == 0 { // as the book's functions leave it
		return Book{}, nil
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
	fmt.Fprintf(&buf, "{\n  \"version\": %d,\n  \"roles\": ", version)
	if err := config.JSONLines(&buf, b.Roles); err != nil {
		return err
	}
	buf.WriteString(",\n  \"sessions\": ")
	if err := config.JSONLines(&buf, b.Sessions); err != nil {
		return err
	}
	buf.WriteString("\n}\n")
	return config.WriteFile(path, buf.Bytes())
}
