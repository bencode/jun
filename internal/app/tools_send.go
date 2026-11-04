package app

import (
	"fmt"

	"github.com/bencode/jun/internal/agent"
	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/tool"
)

// toolSend submits the text to the agent in the terminal, whatever the
// agent is doing: only a terminal running no agent cannot take it.
func (a *app) toolSend(req tool.Request) error {
	s, err := a.toolTerminal(req.Terminal)
	if err != nil {
		return err
	}
	if _, ok := a.agents[s.ID]; !ok {
		return fmt.Errorf("terminal %d runs no agent", s.ID)
	}
	if len(req.Text) == 0 || len(req.Text) > tool.MaxText {
		return fmt.Errorf("text must be 1 to %d bytes", tool.MaxText)
	}
	s.Send(req.Text)
	return nil
}

func (a *app) agentWorking(id terminal.ID) bool {
	info, ok := a.agents[id]
	return ok && info.Status == agent.Working
}
