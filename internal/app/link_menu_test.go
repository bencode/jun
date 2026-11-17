package app

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
)

// TestLinkMenu checks that a link in a table's cell has a menu of its own,
// opening its whole destination rather than the row's text.
func TestLinkMenu(t *testing.T) {
	const src = "| Name | Link |\n| --- | --- |\n| docs | [guide](https://example.com/a/very/long/path?q=1) |\n"
	a := &app{}
	var opened []string
	open := func(dest string) { opened = append(opened, dest) }
	o := &markdown.Options{OnLink: open, LinkMenu: a.linkMenu(open)}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(10).Children(func() { markdown.View(c, src, o) })
	}, 600, 300)
	if err := tt.RightClick("guide"); err != nil {
		t.Fatal(err)
	}
	if want := []string{"Open Link", "Copy Link", "-", "Copy", "-", "Select All"}; !slices.Equal(tt.Menu(), want) {
		t.Fatalf("menu %q, want %q", tt.Menu(), want)
	}
	if err := tt.ChooseMenuItem("Open Link"); err != nil {
		t.Fatal(err)
	}
	if want := []string{"https://example.com/a/very/long/path?q=1"}; !slices.Equal(opened, want) {
		t.Errorf("opened %q, want %q", opened, want)
	}
}
