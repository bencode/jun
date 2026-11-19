package app

import (
	"fmt"
	"strings"

	"github.com/bencode/jun/internal/tool"
)

// toolUpdateTerminal names the terminal, or drops its name for none. A
// name is no other running terminal's, whatever its case: agents find a
// terminal by it.
func (a *app) toolUpdateTerminal(req tool.Request) error {
	if err := tool.CheckName(req.Name); err != nil {
		return err
	}
	s, err := a.toolTerminal(req.Terminal)
	if err != nil {
		return err
	}
	if req.Name == "" {
		delete(a.names, s.ID)
		return nil
	}
	for _, other := range a.terminals.All() {
		if other.ID != s.ID && !other.Exited() && strings.EqualFold(a.names[other.ID], req.Name) {
			return fmt.Errorf("name %q is terminal %d's", req.Name, other.ID)
		}
	}
	a.names[s.ID] = req.Name
	return nil
}
