package app

import "testing"

func TestShortTitleKeepsShellDirectoryOnly(t *testing.T) {
	cases := map[string]string{
		"bencode@Weis-MacBook-Pro:~/work/lesscap/jun": "jun",
		"bencode@Weis-MacBook-Pro:~":                  "~",
		"bencode@host:/":                              "/",
		"bencode@host:/tmp/my dir/":                   "my dir",
		"✳ Claude Code":                               "✳ Claude Code",
		"vim README.md":                               "vim README.md",
		"":                                            "",
	}
	for title, want := range cases {
		if got := shortTitle(title); got != want {
			t.Errorf("shortTitle(%q) = %q, want %q", title, got, want)
		}
	}
}
