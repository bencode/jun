// Package settings reads and writes the user's settings file, a YAML file
// people edit by hand: Jun changes only the keys its own controls set,
// keeping the rest of the file, comments included.
package settings

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// Settings are the user's preferences; a zero value means Jun's default.
type Settings struct {
	Language string `yaml:"language,omitempty"` // "en" or "zh"
	Theme    string `yaml:"theme,omitempty"`    // a Ghostty theme's name; Jun's own when empty
	Font     struct {
		Size float32 `yaml:"size,omitempty"` // terminals' and file tabs' text size in DIPs
	} `yaml:"font,omitempty"`
	Markdown struct {
		Width string `yaml:"width,omitempty"` // one of MarkdownWidths; medium when empty
	} `yaml:"markdown,omitempty"`
	Claude struct {
		// ConfigDirs are the Claude Code configuration directories whose
		// sessions Sessions lists; Claude Code's own when empty.
		ConfigDirs []string `yaml:"configDirs,omitempty"`
	} `yaml:"claude,omitempty"`
}

// MarkdownWidths are the widths Markdown pages and session tabs read at,
// narrowest first.
var MarkdownWidths = []string{"narrow", "medium", "wide", "full"}

// Font sizes, in DIPs.
const MinFontSize, MaxFontSize, DefaultFontSize float32 = 8, 40, 13

// Template is what the settings tab offers while there is no file.
const Template = `# Jun settings. Saved changes apply at once.
#
# language: en          # en | zh
# theme: ""             # a Ghostty theme's name; Jun's own when empty
# font:
#   size: 13            # terminals' and file tabs' text, 8 to 40
# markdown:
#   width: medium       # narrow | medium | wide | full
# claude:
#   configDirs:         # Claude Code configuration folders whose sessions
#     - ~/.claude       # Sessions lists; Claude Code's own when none
`

// Path is the settings file: JUN_SETTINGS_PATH, else
// ~/.config/jun/settings.yaml.
func Path() (string, error) {
	if override := os.Getenv("JUN_SETTINGS_PATH"); override != "" {
		return filepath.Abs(override)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "jun", "settings.yaml"), nil
}

// Load reads the settings file; with no file, the settings are Jun's
// defaults.
func Load(path string) (Settings, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Settings{}, nil
	}
	if err != nil {
		return Settings{}, err
	}
	s, err := Parse(data)
	if err != nil {
		return Settings{}, fmt.Errorf("%s: %w", path, err)
	}
	return s, nil
}

// Parse reads settings, refusing keys and values Jun does not know, and
// expands ~ in directories.
func Parse(data []byte) (Settings, error) {
	var s Settings
	d := yaml.NewDecoder(bytes.NewReader(data))
	d.KnownFields(true)
	if err := d.Decode(&s); err != nil && !errors.Is(err, io.EOF) {
		return Settings{}, err
	}
	if s.Language != "" && s.Language != "en" && s.Language != "zh" {
		return Settings{}, fmt.Errorf("language: %q is neither en nor zh", s.Language)
	}
	if s.Font.Size != 0 && (s.Font.Size < MinFontSize || s.Font.Size > MaxFontSize) {
		return Settings{}, fmt.Errorf("font.size: %g is not between %g and %g", s.Font.Size, MinFontSize, MaxFontSize)
	}
	if w := s.Markdown.Width; w != "" && !slices.Contains(MarkdownWidths, w) {
		return Settings{}, fmt.Errorf("markdown.width: %q is not one of %s", w, strings.Join(MarkdownWidths, ", "))
	}
	for i, dir := range s.Claude.ConfigDirs {
		expanded, err := expand(dir)
		if err != nil {
			return Settings{}, fmt.Errorf("claude.configDirs: %w", err)
		}
		s.Claude.ConfigDirs[i] = expanded
	}
	return s, nil
}

// expand makes a directory absolute, from ~ or ~/… in the home directory.
func expand(dir string) (string, error) {
	if dir == "~" || strings.HasPrefix(dir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, dir[1:])
	}
	if !filepath.IsAbs(dir) {
		return "", fmt.Errorf("%q is not an absolute directory", dir)
	}
	return filepath.Clean(dir), nil
}

// Set gives the key, a path of names, the value in the settings file,
// creating the file or the key as needed and leaving the rest as it was.
func Set(path string, key []string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	var v yaml.Node
	if err := v.Encode(value); err != nil {
		return err
	}
	if len(doc.Content) == 0 {
		// Nothing but comments, if anything: keep them, and add the key.
		out, err := encode(&yaml.Node{Kind: yaml.MappingNode, Content: nest(key, &v)})
		if err != nil {
			return err
		}
		if len(data) > 0 && !bytes.HasSuffix(data, []byte("\n")) {
			data = append(data, '\n')
		}
		return Write(path, append(data, out...))
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return fmt.Errorf("%s: the settings are not a mapping of keys", path)
	}
	for i, name := range key {
		at := lookup(root, name)
		if at < 0 {
			root.Content = append(root.Content, nest(key[i:], &v)...)
			break
		}
		if i == len(key)-1 {
			v.LineComment = root.Content[at].LineComment
			root.Content[at] = &v
			break
		}
		if root.Content[at].Kind != yaml.MappingNode {
			return fmt.Errorf("%s: %s is not a mapping of keys", path, strings.Join(key[:i+1], "."))
		}
		root = root.Content[at]
	}
	out, err := encode(&doc)
	if err != nil {
		return err
	}
	return Write(path, out)
}

// lookup is the index of the value of the mapping's key, or -1.
func lookup(m *yaml.Node, name string) int {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == name {
			return i + 1
		}
	}
	return -1
}

// nest is a mapping's content giving the key, a path of names, the value.
func nest(key []string, v *yaml.Node) []*yaml.Node {
	value := v
	for i := len(key) - 1; i > 0; i-- {
		value = &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{{Kind: yaml.ScalarNode, Value: key[i]}, value}}
	}
	return []*yaml.Node{{Kind: yaml.ScalarNode, Value: key[0]}, value}
}

func encode(n *yaml.Node) ([]byte, error) {
	var b bytes.Buffer
	e := yaml.NewEncoder(&b)
	e.SetIndent(2)
	if err := e.Encode(n); err != nil {
		return nil, err
	}
	if err := e.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// Write replaces the settings file with data once it is written whole,
// creating its directory.
func Write(path string, data []byte) (err error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".settings-*.tmp")
	if err != nil {
		return err
	}
	defer func() {
		if cleanup := os.Remove(f.Name()); cleanup != nil && !errors.Is(cleanup, fs.ErrNotExist) {
			err = errors.Join(err, cleanup)
		}
	}()
	_, writeErr := f.Write(data)
	if err := errors.Join(writeErr, f.Close()); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
