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

	if err := a.toolSend(tool.Request{Op: tool.OpSend, Terminal: uint64(target.ID), Text: "printf 'got %s\\n' it"}); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the text to run", func() bool { return strings.Contains(target.Terminal.Text(), "\ngot it") })
	for req, want := range map[tool.Request]string{
		{Op: tool.OpSend, Terminal: 999, Text: "hi"}:              "no terminal 999",
		{Op: tool.OpSend, Terminal: uint64(shell.ID), Text: "hi"}: fmt.Sprintf("terminal %d runs no agent", shell.ID),
		{Op: tool.OpSend, Terminal: uint64(target.ID)}:            "text must be 1 to 1048576 bytes",
	} {
		if err := a.toolSend(req); err == nil || err.Error() != want {
			t.Errorf("%+v: %v, want %q", req, err, want)
		}
	}
}
