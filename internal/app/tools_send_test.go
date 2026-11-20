package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bencode/jun/internal/agent"
	"github.com/bencode/jun/internal/tool"
)

func TestSendSubmitsToAnAgentOnly(t *testing.T) {
	a, tt := testApp(t)
	tt.Frame()
	_, target, shell := testTerminals(t, a)
	a.agents[target.ID] = agentInfo{Kind: agent.Claude, Status: agent.Working} // busy or not, it takes it
	shellOf(t, target)                                                         // its shell runs

	if held, err := a.toolSend(tool.Request{Op: tool.OpSend, Terminal: uint64(target.ID), Text: "printf 'got %s\\n' it"}); err != nil || held {
		t.Fatal(held, err)
	}
	eventually(t, "the text to run", func() bool { return strings.Contains(target.Terminal.Text(), "\ngot it") })
	for req, want := range map[tool.Request]string{
		{Op: tool.OpSend, Terminal: 999, Text: "hi"}:              "no terminal 999",
		{Op: tool.OpSend, Terminal: uint64(shell.ID), Text: "hi"}: fmt.Sprintf("terminal %d runs no agent", shell.ID),
		{Op: tool.OpSend, Terminal: uint64(target.ID)}:            "text must be 1 to 1048576 bytes",
	} {
		if _, err := a.toolSend(req); err == nil || err.Error() != want {
			t.Errorf("%+v: %v, want %q", req, err, want)
		}
	}
}

func TestSendHoldsWhileTheUserHasADraft(t *testing.T) {
	a, tt := testApp(t)
	tt.Frame()
	_, target, _ := testTerminals(t, a)
	a.agents[target.ID] = agentInfo{Kind: agent.Claude, Status: agent.Idle}
	shellOf(t, target)

	// What Claude Code shows, the input box with the user's draft or empty.
	show := func(input string) {
		t.Helper()
		target.Terminal.Feed([]byte("\x1b[2J\x1b[H⏺ Done.\r\n──────\r\n❯ " + input + "\r\n──────\r\n"))
		text := strings.NewReplacer("\x1b[2m", "", "\x1b[0m", "").Replace(input)
		eventually(t, "the input box", func() bool { return strings.Contains(target.Terminal.Text(), "❯ "+text) })
	}
	send := func(text string) {
		t.Helper()
		if held, err := a.toolSend(tool.Request{Op: tool.OpSend, Terminal: uint64(target.ID), Text: text}); err != nil || !held {
			t.Fatalf("%q: held %v, %v", text, held, err)
		}
	}
	show("half a thought")
	send("printf 'got %s\\n' one")
	show("") // held texts wait their turn, input empty or not
	send("printf 'got %s\\n' two")
	if got := a.heldSends[target.ID]; len(got) != 2 {
		t.Fatalf("held %q", got)
	}
	// The user sees them wait, over the terminal and in Agents.
	shows := func(want string) bool {
		tt.Frame()
		return tt.HasText(want)
	}
	if !shows("✉ 2 waiting") || !strings.Contains(a.agentLabel(target), "2 waiting to send") {
		t.Fatalf("not shown held: %q", a.agentLabel(target))
	}

	show("still typing")
	a.sendHeld()
	if len(a.heldSends[target.ID]) != 2 {
		t.Fatal("submitted over the user's draft")
	}
	show("")
	a.sendHeld()
	eventually(t, "the first text to run", func() bool { return strings.Contains(target.Terminal.Text(), "\ngot one") })
	if !shows("✉ 1 waiting") {
		t.Fatal("the count not down to one")
	}
	if strings.Contains(target.Terminal.Text(), "\ngot two") {
		t.Fatal("submitted both at once")
	}
	show("\x1b[2mTry \"fix the bug\"\x1b[0m") // a placeholder is no draft
	a.sendHeld()
	eventually(t, "the second text to run", func() bool { return strings.Contains(target.Terminal.Text(), "\ngot two") })
	if _, ok := a.heldSends[target.ID]; ok {
		t.Fatal("held texts left after all were sent")
	}
	if shows("✉ 1 waiting") || strings.Contains(a.agentLabel(target), "waiting to send") {
		t.Fatal("still shown held once all were sent")
	}

	show("typing")
	send("never")
	delete(a.agents, target.ID) // the agent quits
	a.sendHeld()
	if _, ok := a.heldSends[target.ID]; ok {
		t.Fatal("kept texts for a terminal running no agent")
	}
}
