package repo

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func run(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func write(t *testing.T, dir, name, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSourcesFilesAndRead(t *testing.T) {
	base := t.TempDir()
	main := filepath.Join(base, "main")
	run(t, base, "init", "-q", "-b", "main", main)
	write(t, main, ".gitignore", "*.log\n")
	write(t, main, "README.md", "hello\n")
	write(t, main, "gone.txt", "soon deleted\n")
	run(t, main, "add", ".")
	run(t, main, "commit", "-q", "-m", "one", "--date", "2026-01-01T00:00:00")
	run(t, main, "branch", "feat/b")
	run(t, main, "checkout", "-q", "feat/b")
	write(t, main, "b.txt", "on b\n")
	run(t, main, "add", ".")
	run(t, main, "commit", "-q", "-m", "b")
	run(t, main, "checkout", "-q", "main")
	run(t, main, "worktree", "add", "-q", "-b", "feat/a", filepath.Join(base, "a"))
	write(t, main, "new/draft.go", "package draft\n")
	write(t, main, "debug.log", "ignored\n")
	if err := os.Remove(filepath.Join(main, "gone.txt")); err != nil {
		t.Fatal(err)
	}

	top, sources, err := Sources(filepath.Join(main, "new"))
	if err != nil {
		t.Fatal(err)
	}
	keys := []string{}
	for _, s := range sources {
		keys = append(keys, s.Key())
	}
	if top != resolve(main) || !reflect.DeepEqual(keys, []string{"main", "feat/a", "feat/b"}) {
		t.Fatalf("top %s, sources %v", top, keys)
	}
	if sources[0].Worktree != top || sources[1].Worktree != resolve(filepath.Join(base, "a")) || sources[2].Worktree != "" {
		t.Fatalf("worktrees %+v", sources)
	}

	files, err := Files(top, sources[0])
	if err != nil || !reflect.DeepEqual(files, []string{".gitignore", "README.md", "new/draft.go"}) {
		t.Fatalf("main files %v, %v", files, err)
	}
	files, err = Files(top, sources[2])
	if err != nil || !reflect.DeepEqual(files, []string{".gitignore", "README.md", "b.txt", "gone.txt"}) {
		t.Fatalf("feat/b files %v, %v", files, err)
	}
	if data, size, err := Read(top, sources[0], "new/draft.go", 1<<20); err != nil || string(data) != "package draft\n" || size != 14 {
		t.Fatalf("worktree read %q %d %v", data, size, err)
	}
	if data, size, err := Read(top, sources[2], "b.txt", 1<<20); err != nil || string(data) != "on b\n" || size != 5 {
		t.Fatalf("branch read %q %d %v", data, size, err)
	}
	if data, size, err := Read(top, sources[2], "b.txt", 2); err != nil || data != nil || size != 5 {
		t.Fatalf("a file over the limit was read: %q %d %v", data, size, err)
	}
	if _, _, err := Read(top, sources[0], "../outside", 1<<20); err == nil {
		t.Fatal("a path outside the repository was read")
	}
	if _, _, err := Sources(base); !errors.Is(err, ErrNotRepository) {
		t.Fatalf("a directory outside any repository: %v", err)
	}
}
