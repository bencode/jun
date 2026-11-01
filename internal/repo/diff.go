package repo

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// lineCountLimit is the largest new file whose lines are counted; larger
// ones count as binary files do.
const lineCountLimit = 10 << 20

// Change is a file a diff changes.
type Change struct {
	Status         byte   // 'A' added, 'M' modified, 'D' deleted, 'R' renamed
	Path, OldPath  string // slash-separated; OldPath only for a rename
	Added, Deleted int    // lines; -1 for a binary file
}

// Commit is a commit of a branch: its short hash and subject.
type Commit struct{ Hash, Subject string }

// Compare is what a diff shows: a source's changes since it forked from
// Base, those not committed yet, or one commit's.
type Compare struct {
	Base        string
	Uncommitted bool
	Commit      string
}

// Diff is the changes from commit From to commit To, or to the worktree
// when To is "".
type Diff struct {
	From, To string
	Changes  []Change // by path
}

// Bases lists what a source can be compared to: the local branches, the
// latest first, then the remote branches they track.
func Bases(top string) ([]string, error) {
	out, err := git(top, "for-each-ref", "refs/heads", "--sort=-committerdate", "--format=%(refname:short)%00%(upstream:short)")
	if err != nil {
		return nil, err
	}
	var names, upstreams []string
	for _, line := range lines(out) {
		name, upstream, _ := strings.Cut(line, "\x00")
		names = append(names, name)
		if upstream != "" && !slices.Contains(upstreams, upstream) {
			upstreams = append(upstreams, upstream)
		}
	}
	upstreams = slices.DeleteFunc(upstreams, func(u string) bool { return slices.Contains(names, u) })
	return append(names, upstreams...), nil
}

// GuessBase picks the base the source most likely forked from: of the
// others, the one fewest of its commits are not in, then the one with
// fewest commits not in it. A base holding all of its commits, as a branch
// built on it, is picked only when every one does. "" when there is none.
func GuessBase(top string, s Source, bases []string) (string, error) {
	type candidate struct {
		name          string
		behind, ahead int
	}
	dir, tip := cmp.Or(s.Worktree, top), tipOf(s)
	var all []candidate
	for _, b := range bases {
		if s.Branch != "" && (b == s.Branch || strings.HasSuffix(b, "/"+s.Branch)) {
			continue
		}
		out, err := git(dir, "rev-list", "--left-right", "--count", b+"..."+tip)
		if err != nil {
			return "", err
		}
		f := strings.Fields(string(out))
		if len(f) != 2 {
			return "", fmt.Errorf("git rev-list: unexpected output %q", out)
		}
		behind, err1 := strconv.Atoi(f[0])
		ahead, err2 := strconv.Atoi(f[1])
		if err := errors.Join(err1, err2); err != nil {
			return "", err
		}
		all = append(all, candidate{b, behind, ahead})
	}
	apart := slices.DeleteFunc(slices.Clone(all), func(c candidate) bool { return c.ahead == 0 && c.behind > 0 })
	if len(apart) == 0 {
		apart = all
	}
	if len(apart) == 0 {
		return "", nil
	}
	best := slices.MinFunc(apart, func(x, y candidate) int {
		return cmp.Or(cmp.Compare(x.ahead, y.ahead), cmp.Compare(x.behind, y.behind))
	})
	return best.name, nil
}

// Changes lists the files the comparison changes, with their lines added
// and deleted. A worktree's changes include its files not committed, and
// those not tracked and not ignored.
func Changes(top string, s Source, c Compare) (Diff, error) {
	d, err := ends(top, s, c)
	if err != nil {
		return Diff{}, err
	}
	dir := cmp.Or(s.Worktree, top)
	args := []string{"diff", "--no-color", "--no-ext-diff", "--no-textconv", "-z", "-M"}
	revs := []string{d.From}
	if d.To != "" {
		revs = append(revs, d.To)
	}
	statuses, err := git(dir, append(append(args, "--name-status"), revs...)...)
	if err != nil {
		return Diff{}, err
	}
	counts, err := git(dir, append(append(args, "--numstat"), revs...)...)
	if err != nil {
		return Diff{}, err
	}
	if d.Changes, err = parseStatuses(statuses); err != nil {
		return Diff{}, err
	}
	lines, err := parseCounts(counts)
	if err != nil {
		return Diff{}, err
	}
	for i := range d.Changes {
		n, ok := lines[d.Changes[i].Path]
		if !ok {
			n = [2]int{-1, -1}
		}
		d.Changes[i].Added, d.Changes[i].Deleted = n[0], n[1]
	}
	if d.To == "" {
		out, err := git(s.Worktree, "ls-files", "-z", "--others", "--exclude-standard")
		if err != nil {
			return Diff{}, err
		}
		for _, name := range fields(out) {
			n, err := countLines(filepath.Join(s.Worktree, filepath.FromSlash(name)))
			if errors.Is(err, fs.ErrNotExist) {
				continue // removed since it was listed
			}
			if err != nil {
				return Diff{}, err
			}
			d.Changes = append(d.Changes, Change{Status: 'A', Path: name, Added: n})
		}
	}
	slices.SortFunc(d.Changes, func(x, y Change) int { return strings.Compare(x.Path, y.Path) })
	return d, nil
}

// ends resolves the commits the comparison runs between.
func ends(top string, s Source, c Compare) (Diff, error) {
	dir, tip := cmp.Or(s.Worktree, top), tipOf(s)
	if c.Commit != "" {
		to, err := commitOf(dir, c.Commit)
		if err != nil {
			return Diff{}, err
		}
		out, err := git(dir, "rev-list", "--parents", "-n", "1", to)
		if err != nil {
			return Diff{}, err
		}
		if f := strings.Fields(string(out)); len(f) > 1 {
			return Diff{From: f[1], To: to}, nil
		}
		empty, err := git(dir, "hash-object", "-t", "tree", os.DevNull)
		return Diff{From: strings.TrimSpace(string(empty)), To: to}, err
	}
	if c.Uncommitted {
		if s.Worktree == "" {
			return Diff{}, fmt.Errorf("%s is not checked out", s.Key())
		}
		head, err := commitOf(dir, "HEAD")
		return Diff{From: head}, err
	}
	if !validRev(c.Base) {
		return Diff{}, fmt.Errorf("invalid base %q", c.Base)
	}
	out, err := git(dir, "merge-base", c.Base, tip)
	if err != nil {
		return Diff{}, err
	}
	d := Diff{From: strings.TrimSpace(string(out))}
	if s.Worktree == "" {
		d.To, err = commitOf(dir, tip)
	}
	return d, err
}

// Commits lists the source's commits not in base, the latest first, at
// most limit.
func Commits(top string, s Source, base string, limit int) ([]Commit, error) {
	if !validRev(base) {
		return nil, fmt.Errorf("invalid base %q", base)
	}
	out, err := git(cmp.Or(s.Worktree, top), "log", "--no-color", "--format=%h%x00%s", "-n", strconv.Itoa(limit), base+".."+tipOf(s))
	if err != nil {
		return nil, err
	}
	var commits []Commit
	for _, line := range lines(out) {
		hash, subject, _ := strings.Cut(line, "\x00")
		commits = append(commits, Commit{hash, subject})
	}
	return commits, nil
}

// ReadAt returns a file as it is in commit rev, up to limit bytes, and its
// size: no data when the file is larger than limit.
func ReadAt(top, rev, name string, limit int64) (data []byte, size int64, err error) {
	if !validRev(rev) || !Valid(name) {
		return nil, 0, fmt.Errorf("invalid file %q at %q", name, rev)
	}
	object := rev + ":" + name
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

// tipOf is the revision a source's commits end at: its branch, else, in
// its detached worktree, HEAD.
func tipOf(s Source) string { return cmp.Or(s.Branch, "HEAD") }

func commitOf(dir, rev string) (string, error) {
	if !validRev(rev) {
		return "", fmt.Errorf("invalid revision %q", rev)
	}
	out, err := git(dir, "rev-parse", "--verify", rev+"^{commit}")
	return strings.TrimSpace(string(out)), err
}

// validRev reports whether rev can be given to git as a revision, not
// read as an option.
func validRev(rev string) bool {
	return rev != "" && !strings.HasPrefix(rev, "-") && !strings.ContainsAny(rev, " \t\n\x00")
}

// parseStatuses reads `git diff -z --name-status`.
func parseStatuses(out []byte) ([]Change, error) {
	f := strings.Split(string(out), "\x00")
	var changes []Change
	for i := 0; i < len(f) && f[i] != ""; i++ {
		if i+1 >= len(f) {
			return nil, fmt.Errorf("git diff: unexpected status %q", f[i])
		}
		c := Change{Status: f[i][0], Path: f[i+1]}
		i++
		if c.Status == 'R' || c.Status == 'C' {
			if i+1 >= len(f) {
				return nil, fmt.Errorf("git diff: rename of %q has no new name", c.Path)
			}
			c.OldPath, c.Path = c.Path, f[i+1]
			i++
		}
		if !strings.ContainsRune("AMDR", rune(c.Status)) {
			c.Status = 'M' // a type change, or a file not merged yet
		}
		changes = append(changes, c)
	}
	return changes, nil
}

// parseCounts reads `git diff -z --numstat`: lines added and deleted, by
// new path; -1 for binary files.
func parseCounts(out []byte) (map[string][2]int, error) {
	f := strings.Split(string(out), "\x00")
	counts := map[string][2]int{}
	for i := 0; i < len(f) && f[i] != ""; i++ {
		parts := strings.SplitN(f[i], "\t", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("git diff: unexpected count %q", f[i])
		}
		name := parts[2]
		if name == "" { // a rename: the old and new names follow
			if i+2 >= len(f) {
				return nil, fmt.Errorf("git diff: rename count has no names")
			}
			name = f[i+2]
			i += 2
		}
		counts[name] = [2]int{count(parts[0]), count(parts[1])}
	}
	return counts, nil
}

func count(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return -1 // "-", for a binary file
	}
	return n
}

// countLines counts a new file's lines; -1 when it is binary or too
// large.
func countLines(name string) (int, error) {
	info, err := os.Stat(name)
	if err != nil {
		return 0, err
	}
	if info.Size() > lineCountLimit {
		return -1, nil
	}
	data, err := os.ReadFile(name)
	if err != nil {
		return 0, err
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return -1, nil
	}
	n := bytes.Count(data, []byte("\n"))
	if len(data) > 0 && data[len(data)-1] != '\n' {
		n++
	}
	return n, nil
}
