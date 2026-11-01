// Package repo reads what Files shows of a Git repository: its local
// branches and worktrees, their files, and a file's content, all through
// the git command.
package repo

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

// ErrNotRepository is what Sources returns for a directory in no repository.
var ErrNotRepository = errors.New("not a Git repository")

// Source is a place Files reads a repository's files from: a local branch,
// read from the worktree that checks it out, if any, else from its commit;
// or a detached worktree.
type Source struct {
	Branch   string    // "" for a detached worktree
	Worktree string    // absolute; "" when no worktree checks the branch out
	Commit   string    // the tip's short hash
	Time     time.Time // the tip's commit time
}

// Key identifies a source across refreshes: its branch, else its worktree.
func (s Source) Key() string { return cmp.Or(s.Branch, s.Worktree) }

// Sources lists the repository of dir: its top level and its sources, the
// one checked out where dir is first, then by commit time, newest first.
// It returns ErrNotRepository when dir is in no repository.
func Sources(dir string) (top string, sources []Source, err error) {
	out, err := git(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		if strings.Contains(err.Error(), "not a git repository") {
			return "", nil, ErrNotRepository
		}
		return "", nil, err
	}
	top = resolve(strings.TrimSpace(string(out)))
	worktrees, err := worktrees(top)
	if err != nil {
		return "", nil, err
	}
	out, err = git(top, "for-each-ref", "refs/heads", "--format=%(refname:short)%00%(objectname:short)%00%(committerdate:unix)")
	if err != nil {
		return "", nil, err
	}
	for _, line := range lines(out) {
		f := strings.Split(line, "\x00")
		if len(f) != 3 {
			return "", nil, fmt.Errorf("git for-each-ref: unexpected line %q", line)
		}
		when, err := unix(f[2])
		if err != nil {
			return "", nil, err
		}
		sources = append(sources, Source{Branch: f[0], Worktree: worktrees.branches[f[0]], Commit: f[1], Time: when})
	}
	for _, w := range worktrees.detached {
		out, err := git(top, "log", "-1", "--format=%h%x00%ct", w.head)
		if err != nil {
			return "", nil, err
		}
		short, date, _ := strings.Cut(strings.TrimSpace(string(out)), "\x00")
		when, err := unix(date)
		if err != nil {
			return "", nil, err
		}
		sources = append(sources, Source{Worktree: w.path, Commit: short, Time: when})
	}
	slices.SortStableFunc(sources, func(a, b Source) int {
		if (a.Worktree == top) != (b.Worktree == top) {
			if a.Worktree == top {
				return -1
			}
			return 1
		}
		return cmp.Or(b.Time.Compare(a.Time), cmp.Compare(a.Key(), b.Key()))
	})
	return top, sources, nil
}

// worktreeMap maps branches to the worktrees checking them out, and keeps
// the detached worktrees.
type worktreeMap struct {
	branches map[string]string
	detached []struct{ path, head string }
}

// worktrees reads `git worktree list --porcelain`, skipping bare and
// prunable worktrees.
func worktrees(top string) (worktreeMap, error) {
	out, err := git(top, "worktree", "list", "--porcelain", "-z")
	if err != nil {
		return worktreeMap{}, err
	}
	result := worktreeMap{branches: map[string]string{}}
	for _, block := range strings.Split(string(out), "\x00\x00") {
		attrs := map[string]string{}
		for _, field := range strings.Split(block, "\x00") {
			key, value, _ := strings.Cut(field, " ")
			attrs[key] = value
		}
		dir, head, branch := attrs["worktree"], attrs["HEAD"], strings.TrimPrefix(attrs["branch"], "refs/heads/")
		_, bare := attrs["bare"]
		_, prunable := attrs["prunable"]
		_, detached := attrs["detached"]
		if dir == "" || bare || prunable {
			continue
		}
		dir = resolve(dir)
		if detached {
			result.detached = append(result.detached, struct{ path, head string }{dir, head})
		} else if branch != "" {
			result.branches[branch] = dir
		}
	}
	return result, nil
}

// Files lists a source's files, relative to the top level, slash-separated:
// a worktree's tracked files that are still there and its untracked files
// that are not ignored; a branch's committed files.
func Files(top string, s Source) ([]string, error) {
	if s.Worktree == "" {
		out, err := git(top, "ls-tree", "-r", "-z", "--name-only", s.Branch)
		return fields(out), err
	}
	out, err := git(s.Worktree, "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	gone, err := git(s.Worktree, "ls-files", "-z", "--deleted")
	if err != nil {
		return nil, err
	}
	// Unmerged files are listed once per stage; deleted ones are left out.
	seen := map[string]bool{}
	for _, f := range fields(gone) {
		seen[f] = true
	}
	var files []string
	for _, f := range fields(out) {
		if !seen[f] {
			seen[f] = true
			files = append(files, f)
		}
	}
	slices.Sort(files)
	return files, nil
}

// Read returns a file of a source, up to limit bytes, and its size: no
// data when the file is larger than limit.
func Read(top string, s Source, name string, limit int64) (data []byte, size int64, err error) {
	if !Valid(name) {
		return nil, 0, fmt.Errorf("invalid path %q", name)
	}
	if s.Worktree != "" {
		f, err := os.Open(filepath.Join(s.Worktree, filepath.FromSlash(name)))
		if err != nil {
			return nil, 0, err
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil {
			return nil, 0, err
		}
		if info.IsDir() {
			return nil, 0, fmt.Errorf("%s is a directory", name)
		}
		if info.Size() > limit {
			return nil, info.Size(), nil
		}
		data, err = io.ReadAll(io.LimitReader(f, limit))
		return data, info.Size(), err
	}
	object := s.Branch + ":" + name
	out, err := git(top, "cat-file", "-s", object)
	if err != nil {
		return nil, 0, err
	}
	size, err = strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil || size > limit {
		return nil, size, err
	}
	data, err = git(top, "cat-file", "blob", object)
	return data, size, err
}

// Valid reports whether name is a slash-separated path relative to a
// repository's top level, inside it.
func Valid(name string) bool {
	return name != "" && path.Clean(name) == name && !path.IsAbs(name) && name != ".." && !strings.HasPrefix(name, "../")
}

// git runs git in dir without taking optional locks, which agents working
// in the repository could be waiting on, and returns its output.
func git(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("git %s: %s", args[0], msg)
		}
		return nil, fmt.Errorf("git %s: %w", args[0], err)
	}
	return out, nil
}

// resolve follows symlinks in dir, as the temporary directories of macOS
// have, so that git's paths compare equal; dir as it is when it cannot.
func resolve(dir string) string {
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		return real
	}
	return dir
}

func lines(out []byte) []string {
	return slices.DeleteFunc(strings.Split(string(out), "\n"), func(s string) bool { return s == "" })
}

func fields(out []byte) []string {
	return slices.DeleteFunc(strings.Split(string(out), "\x00"), func(s string) bool { return s == "" })
}

func unix(s string) (time.Time, error) {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("git: commit time %q: %w", s, err)
	}
	return time.Unix(n, 0), nil
}
