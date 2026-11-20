package roles

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Dir is where the roles live, beside Jun's configuration: a file each,
// that lines are only added to. Its first line names the role; each after
// it is a Change.
func Dir(configPath string) string { return filepath.Join(filepath.Dir(configPath), "roles") }

type header struct {
	Name string `json:"name"`
}

// file is the role's file: named so that a person finds it, and told from
// others by its key.
func file(dir, name string) string {
	sum := sha256.Sum256([]byte(Key(name)))
	base := strings.TrimLeft(strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r < ' ' {
			return '_'
		}
		return r
	}, name), ".")
	return filepath.Join(dir, base+"-"+hex.EncodeToString(sum[:3])+".jsonl")
}

// Load reads the roles in dir; none when it does not exist yet. A file's
// last line, when cut short, is left out: Jun stopped while writing it.
func Load(dir string) (Book, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if err != nil {
		return nil, err
	}
	book := Book{}
	for _, path := range paths {
		r, err := loadFile(path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if len(r.Sessions) == 0 {
			continue
		}
		if _, ok := book[Key(r.Name)]; ok {
			return nil, fmt.Errorf("%s: role %q is another file's too", path, r.Name)
		}
		book[Key(r.Name)] = r
	}
	return book, nil
}

// loadFile replays a role's file: a session moved to another role was
// dropped here first, so each file is read on its own.
func loadFile(path string) (Role, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Role{}, err
	}
	if i := bytes.LastIndexByte(data, '\n'); i < len(data)-1 {
		data = data[:i+1] // cut short
	}
	if len(data) == 0 {
		return Role{}, nil // cut short before its name
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(nil, 1<<20)
	var h header
	if !scanner.Scan() || json.Unmarshal(scanner.Bytes(), &h) != nil || h.Name == "" {
		return Role{}, errors.New("its first line does not name the role")
	}
	book := Book{}
	for line := 2; scanner.Scan(); line++ {
		var c Change
		if err := json.Unmarshal(scanner.Bytes(), &c); err != nil || (c.Add == nil) == (c.Drop == nil) {
			return Role{}, fmt.Errorf("line %d is no change", line)
		}
		c.Name = h.Name
		book = apply(book, c)
	}
	if err := scanner.Err(); err != nil {
		return Role{}, err
	}
	if r, ok := book[Key(h.Name)]; ok {
		return r, nil
	}
	return Role{Name: h.Name}, nil
}

// Write adds the changes to the roles' files, in order, making the files
// that are not there yet.
func Write(dir string, changes []Change) error {
	if len(changes) == 0 {
		return nil
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	for _, c := range changes {
		if err := add(file(dir, c.Name), c); err != nil {
			return err
		}
	}
	return nil
}

// add appends the change, first dropping a line cut short, for the next
// to start a line of its own.
func add(path string, c Change) (err error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	size, err := whole(f)
	if err != nil {
		return err
	}
	var lines []byte
	if size == 0 {
		h, _ := json.Marshal(header{Name: c.Name})
		lines = append(h, '\n')
	}
	line, err := json.Marshal(c)
	if err != nil {
		return err
	}
	_, err = f.Write(append(append(lines, line...), '\n'))
	return err
}

// whole cuts the file back to its last full line: its size after.
func whole(f *os.File) (int64, error) {
	info, err := f.Stat()
	if err != nil || info.Size() == 0 {
		return 0, err
	}
	last := make([]byte, 1)
	if _, err := f.ReadAt(last, info.Size()-1); err != nil {
		return 0, err
	}
	if last[0] == '\n' {
		return info.Size(), nil
	}
	data := make([]byte, info.Size())
	if _, err := f.ReadAt(data, 0); err != nil {
		return 0, err
	}
	size := int64(bytes.LastIndexByte(data, '\n') + 1)
	return size, f.Truncate(size)
}
