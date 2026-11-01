package repo

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestChangesSinceForkUncommittedAndOneCommit(t *testing.T) {
	base := t.TempDir()
	main := filepath.Join(base, "main")
	run(t, base, "init", "-q", "-b", "main", main)
	write(t, main, "keep.txt", "one\ntwo\n")
	write(t, main, "old.txt", "moved\nlines\nstay\n")
	write(t, main, "gone.txt", "bye\n")
	run(t, main, "add", ".")
	run(t, main, "commit", "-q", "-m", "start")
	run(t, main, "branch", "develop")
	run(t, main, "checkout", "-q", "develop")
	write(t, main, "dev.txt", "on develop\n")
	run(t, main, "add", ".")
	run(t, main, "commit", "-q", "-m", "develop")
	run(t, main, "checkout", "-q", "main")
	feature := filepath.Join(base, "feature")
	run(t, main, "worktree", "add", "-q", "-b", "feature", feature, "develop")
	write(t, feature, "keep.txt", "one\nTWO\nthree\n")
	run(t, feature, "mv", "old.txt", "new.txt")
	run(t, feature, "rm", "-q", "gone.txt")
	run(t, feature, "commit", "-q", "-am", "work")
	write(t, feature, "keep.txt", "one\nTWO\nthree\nfour\n")
	write(t, feature, "draft.txt", "a\nb")
	if err := os.WriteFile(filepath.Join(feature, "blob.bin"), []byte{0, 1, 2}, 0o644); err != nil {
		t.Fatal(err)
	}

	top, sources, err := Sources(feature)
	if err != nil {
		t.Fatal(err)
	}
	var s Source
	for _, source := range sources {
		if source.Branch == "feature" {
			s = source
		}
	}
	bases, err := Bases(top)
	if err != nil {
		t.Fatal(err)
	}
	if guess, err := GuessBase(top, s, bases); err != nil || guess != "develop" {
		t.Fatalf("guessed %q, %v; want develop", guess, err)
	}

	all, err := Changes(top, s, Compare{Base: "develop"})
	if err != nil {
		t.Fatal(err)
	}
	want := []Change{
		{Status: 'A', Path: "blob.bin", Added: -1},
		{Status: 'A', Path: "draft.txt", Added: 2},
		{Status: 'D', Path: "gone.txt", Deleted: 1},
		{Status: 'M', Path: "keep.txt", Added: 3, Deleted: 1},
		{Status: 'R', Path: "new.txt", OldPath: "old.txt"},
	}
	if all.To != "" || !reflect.DeepEqual(all.Changes, want) {
		t.Fatalf("changes since develop to %q: %+v", all.To, all.Changes)
	}
	if data, _, err := ReadAt(top, all.From, "keep.txt", 1<<20); err != nil || string(data) != "one\ntwo\n" {
		t.Fatalf("the fork's keep.txt is %q, %v", data, err)
	}

	uncommitted, err := Changes(top, s, Compare{Uncommitted: true})
	if err != nil {
		t.Fatal(err)
	}
	want = []Change{want[0], want[1], {Status: 'M', Path: "keep.txt", Added: 1}}
	if !reflect.DeepEqual(uncommitted.Changes, want) {
		t.Fatalf("uncommitted changes: %+v", uncommitted.Changes)
	}

	commits, err := Commits(top, s, "develop", 10)
	if err != nil || len(commits) != 1 || commits[0].Subject != "work" {
		t.Fatalf("commits since develop: %+v, %v", commits, err)
	}
	one, err := Changes(top, s, Compare{Commit: commits[0].Hash})
	if err != nil {
		t.Fatal(err)
	}
	if one.To == "" || len(one.Changes) != 3 || one.Changes[1] != (Change{Status: 'M', Path: "keep.txt", Added: 2, Deleted: 1}) {
		t.Fatalf("the commit's changes: %+v", one.Changes)
	}
	if _, err := Changes(top, s, Compare{Base: "--output=x"}); err == nil {
		t.Fatal("a base read as an option was accepted")
	}
}
