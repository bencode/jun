package app

import (
	"fmt"

	"github.com/bencode/jun/internal/agent"
	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/theme"
	"github.com/bencode/jun/internal/tool"
	"github.com/egoist/mygo/ui"
)

// toolSend submits the text to the agent in the terminal, whatever the
// agent is doing: only a terminal running no agent cannot take it. While
// the user has a draft in the agent's input, Enter would submit the draft
// with the text: Jun holds the text, after any held before, and reports it
// held; sendHeld submits it once the input is empty.
func (a *app) toolSend(req tool.Request) (held bool, err error) {
	s, err := a.toolTerminal(req.Terminal)
	if err != nil {
		return false, err
	}
	if _, ok := a.agents[s.ID]; !ok {
		return false, fmt.Errorf("terminal %d runs no agent", s.ID)
	}
	if len(req.Text) == 0 || len(req.Text) > tool.MaxText {
		return false, fmt.Errorf("text must be 1 to %d bytes", tool.MaxText)
	}
	if len(a.heldSends[s.ID]) > 0 || a.hasDraft(s) {
		a.heldSends[s.ID] = append(a.heldSends[s.ID], req.Text)
		return true, nil
	}
	s.Send(req.Text)
	return false, nil
}

// sendHeld submits, to each agent with texts held, the first of them once
// its input is empty: one a poll, for the input to empty between them.
// The texts held for a terminal that no longer runs an agent go.
func (a *app) sendHeld() {
	for id, texts := range a.heldSends {
		s := a.terminals.Get(id)
		if _, ok := a.agents[id]; s == nil || !ok {
			delete(a.heldSends, id)
			continue
		}
		if a.hasDraft(s) {
			continue
		}
		s.Send(texts[0])
		if len(texts) == 1 {
			delete(a.heldSends, id)
		} else {
			a.heldSends[id] = texts[1:]
		}
	}
}

// heldBadge tells the user, over the agent's terminal at its bottom right,
// how many texts wait for their draft to go: the terminal keeps its size.
// It sits on the second row from the bottom, the input box's lower edge in
// the agents, leaving the last, their footer, to read: height is the
// terminal's, none before it is laid out.
func (a *app) heldBadge(c *ui.Context, s *terminal.Session, height float32) {
	n := len(a.heldSends[s.ID])
	if n == 0 {
		return
	}
	const badge = 18
	row := float32(badge)
	if _, rows := s.Terminal.Size(); rows > 0 && height > 0 {
		row = height / float32(rows)
	}
	text := fmt.Sprintf(a.text("✉ %d waiting", "✉ %d 条待发"), n)
	color := a.statusColor(agent.Waiting)
	ui.Box(c).Absolute().Right(12).Bottom(row+(row-badge)/2).Height(badge).Padding(0, 8).Radius(badge/2).Center().
		Background(a.theme.Surface).Border(1, color).
		Tooltip(a.text("Sent while you were typing; they go in once you send or clear your input",
			"你输入时发来的消息，发出或清空输入后送达")).
		Children(func() { ui.Text(c, text).FontSize(theme.Caption).TextColor(color) })
}

// hasDraft reports whether the agent's input holds text the user has not
// submitted.
func (a *app) hasDraft(s *terminal.Session) bool {
	info, ok := a.agents[s.ID]
	return ok && agent.HasDraft(info.Kind, s.Terminal.Snapshot())
}

func (a *app) agentWorking(id terminal.ID) bool {
	info, ok := a.agents[id]
	return ok && info.Status == agent.Working
}
