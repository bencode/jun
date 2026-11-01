package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestFontPanelSetsAndResetsTextSize(t *testing.T) {
	a, tt := testApp(t)
	click(t, tt, "Settings")
	if err := tt.ChooseMenuItem("Font…"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	click(t, tt, "Text size")
	tt.Key(ui.Super, ui.KeyA)
	tt.Type("16")
	tt.Frame()
	if a.state.TextSize != 16 {
		t.Fatalf("typing 16 set the text size to %v", a.state.TextSize)
	}
	click(t, tt, "Reset")
	click(t, tt, "Done")
	if a.fontOpen || a.textSize() != defaultTextSize {
		t.Fatalf("after Reset and Done: open %v, size %v", a.fontOpen, a.textSize())
	}
}
