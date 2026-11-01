package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bencode/jun/internal/layout"
)

type document struct {
	Version int           `json:"version"`
	Layout  *layout.State `json:"layout"`
}

func DefaultPath() (string, error) {
	if override := os.Getenv("JUN_CONFIG_PATH"); override != "" {
		return filepath.Abs(override)
	}
	directory, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(directory, "Jun", "config.json"), nil
}

func Load(path string) (*layout.State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	var doc struct {
		Version int             `json:"version"`
		Layout  json.RawMessage `json:"layout"`
	}
	if err := decodeJSON(data, &doc); err != nil {
		return nil, fmt.Errorf("decode config %q: %w", path, err)
	}
	state, err := decodeLayout(doc.Version, doc.Layout)
	if err != nil {
		return nil, fmt.Errorf("config %q: %w", path, err)
	}
	return state, nil
}

// Save replaces the file only after a complete snapshot is written and synced.
func Save(path string, state *layout.State) (err error) {
	if err := state.Validate(); err != nil {
		return fmt.Errorf("save config %q: %w", path, err)
	}
	data, err := json.MarshalIndent(document{Version: 3, Layout: state}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config %q: %w", path, err)
	}
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	file, err := os.CreateTemp(directory, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("create config temporary file: %w", err)
	}
	defer func() {
		if cleanup := os.Remove(file.Name()); cleanup != nil && !errors.Is(cleanup, os.ErrNotExist) {
			err = errors.Join(err, fmt.Errorf("remove config temporary file: %w", cleanup))
		}
	}()
	_, writeErr := file.Write(append(data, '\n'))
	var syncErr error
	if writeErr == nil {
		syncErr = file.Sync()
	}
	if err := errors.Join(writeErr, syncErr, file.Close()); err != nil {
		return fmt.Errorf("write config %q: %w", path, err)
	}
	if err := os.Rename(file.Name(), path); err != nil {
		return fmt.Errorf("replace config %q: %w", path, err)
	}
	return nil
}
