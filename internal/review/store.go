package review

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bencode/jun/internal/config"
)

// Root is where reviews live, beside the layout file, so that a build with
// a layout of its own has reviews of its own.
func Root(configPath string) string { return filepath.Join(filepath.Dir(configPath), "reviews") }

// projectDir is the project's directory of reviews: named so that a person
// finds it, and told from others of the same name by the path.
func projectDir(root, project string) string {
	sum := sha256.Sum256([]byte(project))
	return filepath.Join(root, filepath.Base(project)+"-"+hex.EncodeToString(sum[:3]))
}

// Path is the review's file.
func Path(root string, r Review) string {
	return filepath.Join(projectDir(root, r.Project), r.ID+".json")
}

// List reads the project's reviews, the latest first. A file it cannot read
// is left out, and told in the error, beside the reviews it could.
func List(root, project string) ([]Review, error) {
	dir := projectDir(root, project)
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var reviews []Review
	var errs []error
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		r, err := read(filepath.Join(dir, entry.Name()))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		reviews = append(reviews, r)
	}
	slices.SortFunc(reviews, func(x, y Review) int {
		return cmp.Or(y.CreatedAt.Compare(x.CreatedAt), strings.Compare(y.ID, x.ID))
	})
	return reviews, errors.Join(errs...)
}

func read(path string) (Review, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Review{}, err
	}
	var r Review
	if err := json.Unmarshal(data, &r); err != nil {
		return Review{}, fmt.Errorf("review %s: %w", filepath.Base(path), err)
	}
	if r.ID+".json" != filepath.Base(path) || r.Project == "" || r.Source == "" {
		return Review{}, fmt.Errorf("review %s: not a review", filepath.Base(path))
	}
	return r, nil
}

// Save writes the review's file.
func Save(root string, r Review) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("encode review %s: %w", r.ID, err)
	}
	return config.WriteFile(Path(root, r), append(data, '\n'))
}

// Delete removes the review's file, which may be gone already.
func Delete(root string, r Review) error {
	if err := os.Remove(Path(root, r)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
