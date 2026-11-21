package app

import (
	"errors"
	"fmt"

	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/tool"
)

// lead is the terminal of the agent that leads the terminal, while it
// runs: ids are not used again, so a lead that ended leads nothing.
func (a *app) lead(id terminal.ID) terminal.ID {
	lead := a.leads[id]
	if s := a.terminals.Get(lead); s == nil || s.Exited() {
		return 0
	}
	return lead
}

// toolJoin makes the agent in the terminal of the request the caller's
// lead, in place of any other, and its role the lead of the caller's.
func (a *app) toolJoin(req tool.Request, callers []int) error {
	caller, err := a.toolCaller(callers, req.Caller)
	if err != nil {
		return err
	}
	lead, err := a.toolTerminal(req.Terminal)
	if err != nil {
		return err
	}
	if _, ok := a.agents[lead.ID]; !ok {
		return fmt.Errorf("terminal %d runs no agent", lead.ID)
	}
	if lead.ID == caller {
		return errors.New("a terminal cannot join itself")
	}
	for above := a.lead(lead.ID); above != 0; above = a.lead(above) {
		if above == caller {
			return fmt.Errorf("terminal %d is led by you", lead.ID)
		}
	}
	a.leads[caller] = lead.ID
	return a.keepLead(caller, lead.ID)
}

// toolLeave drops the caller's lead, if it has one, and its role's.
func (a *app) toolLeave(req tool.Request, callers []int) error {
	caller, err := a.toolCaller(callers, req.Caller)
	if err != nil {
		return err
	}
	delete(a.leads, caller)
	return a.keepLead(caller, 0)
}
