package app

import (
	"context"
	"errors"
	"io"
	"log"
	"path"
	"slices"
	"strings"

	"github.com/bencode/jun/internal/repo"
	"github.com/egoist/mygo"
)

// fileNode is a row of Files: a source's root, with no path, or a file or
// directory in it.
type fileNode struct {
	source string // repo.Source.Key
	path   string
}

// listing is a source's files: each directory's children, sorted.
type listing struct {
	children map[string][]fileNode // by directory, "" for the top level
	loading  bool
	err      error
}

// repoState is what Files knows of a project's repository.
type repoState struct {
	top      string
	sources  []repo.Source
	err      error
	loading  bool
	loads    int // counts refreshes, so that a refresh's results replace an older one's
	listings map[string]*listing
	watches  map[string]*mygo.FileWatcher // by worktree
	version  int                          // counts changes to the worktrees' files
}

func (r *repoState) source(key string) (repo.Source, bool) {
	i := slices.IndexFunc(r.sources, func(s repo.Source) bool { return s.Key() == key })
	if i < 0 {
		return repo.Source{}, false
	}
	return r.sources[i], true
}

// background runs work off the UI thread, then done on it; at once
// without a window, as in tests.
func (a *app) background(work, done func()) {
	if a.window == nil {
		work()
		done()
		return
	}
	window := a.window
	go func() {
		work()
		window.Update(done)
	}()
}

// repoFor returns the repository of a project's directory, reading it the
// first time. Keyed by directory, it outlives renaming the project.
func (a *app) repoFor(dir string) *repoState {
	r := a.repos[dir]
	if r == nil {
		r = &repoState{listings: map[string]*listing{}, watches: map[string]*mygo.FileWatcher{}}
		a.repos[dir] = r
		a.refreshRepo(r, dir)
	}
	return r
}

// refreshRepo reads the repository's sources again, and forgets their
// files, which open items read again.
func (a *app) refreshRepo(r *repoState, dir string) {
	r.loads++
	load := r.loads
	r.loading = true
	var top string
	var sources []repo.Source
	var err error
	a.background(func() { top, sources, err = repo.Sources(dir) }, func() {
		if load != r.loads {
			return
		}
		r.top, r.sources, r.err, r.loading = top, sources, err, false
		r.listings = map[string]*listing{}
		r.version++
		for worktree, w := range r.watches {
			if !slices.ContainsFunc(sources, func(s repo.Source) bool { return s.Worktree == worktree }) {
				closeWatch(w)
				delete(r.watches, worktree)
			}
		}
	})
}

// listingFor returns the source's files, reading them the first time.
func (a *app) listingFor(r *repoState, s repo.Source) *listing {
	l := r.listings[s.Key()]
	if l == nil {
		l = &listing{loading: true}
		r.listings[s.Key()] = l
		a.loadListing(r, s, l)
	}
	return l
}

func (a *app) loadListing(r *repoState, s repo.Source, l *listing) {
	top, load := r.top, r.loads
	var files []string
	var err error
	a.background(func() { files, err = repo.Files(top, s) }, func() {
		if load != r.loads || r.listings[s.Key()] != l {
			return
		}
		l.children, l.err, l.loading = index(s.Key(), files), err, false
		if err == nil && s.Worktree != "" {
			a.watch(r, s.Worktree)
		}
	})
}

// index lists each directory's children, directories first, then by name,
// ignoring case.
func index(source string, files []string) map[string][]fileNode {
	children := map[string][]fileNode{}
	dirs := map[string]bool{}
	for _, f := range files {
		for dir := path.Dir(f); dir != "." && !dirs[dir]; dir = path.Dir(dir) {
			dirs[dir] = true
		}
	}
	add := func(p string) {
		parent := path.Dir(p)
		if parent == "." {
			parent = ""
		}
		children[parent] = append(children[parent], fileNode{source, p})
	}
	for dir := range dirs {
		add(dir)
	}
	for _, f := range files {
		add(f)
	}
	for _, kids := range children {
		slices.SortFunc(kids, func(x, y fileNode) int {
			if dirs[x.path] != dirs[y.path] {
				if dirs[x.path] {
					return -1
				}
				return 1
			}
			return strings.Compare(strings.ToLower(path.Base(x.path)), strings.ToLower(path.Base(y.path)))
		})
	}
	for dir := range dirs {
		if children[dir] == nil {
			children[dir] = []fileNode{}
		}
	}
	return children
}

// watch reads a worktree's files again as they change, while there is a
// window: watching needs the app running.
func (a *app) watch(r *repoState, worktree string) {
	if a.window == nil || r.watches[worktree] != nil {
		return
	}
	w, err := mygo.WatchFiles(context.Background(), worktree, mygo.FileWatchOptions{
		Recursive: true, HintsOnly: true,
		Ignore: func(p string, isDir bool) bool { return isDir && p == ".git" },
	})
	if err != nil {
		log.Printf("Jun: watch %s: %v", worktree, err)
		return
	}
	r.watches[worktree] = w
	window := a.window
	go func() {
		for {
			if _, err := w.Next(context.Background()); err != nil {
				if !errors.Is(err, io.EOF) {
					log.Printf("Jun: watch %s: %v", worktree, err)
				}
				return
			}
			window.Update(func() { a.worktreeChanged(r, worktree) })
		}
	}()
}

// worktreeChanged reads the files of the source checked out in the
// worktree again, and tells open files to.
func (a *app) worktreeChanged(r *repoState, worktree string) {
	r.version++
	i := slices.IndexFunc(r.sources, func(s repo.Source) bool { return s.Worktree == worktree })
	if i < 0 {
		return
	}
	s := r.sources[i]
	if l := r.listings[s.Key()]; l != nil && !l.loading {
		a.loadListing(r, s, l)
	}
}

func closeWatch(w *mygo.FileWatcher) {
	if err := w.Close(); err != nil {
		log.Printf("Jun: stop watching %s: %v", w.Root(), err)
	}
}
