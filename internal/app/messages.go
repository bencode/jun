package app

import (
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/bencode/jun/internal/message"
	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/tool"
	"github.com/egoist/mygo/ui"
)

// loadMessages reads the messages beside the configuration. Without a
// configuration path, already reported, Jun serves no tools, so no message
// comes.
func (a *app) loadMessages(configPath string) {
	if configPath == "" {
		return
	}
	a.messagesPath = filepath.Join(filepath.Dir(configPath), "messages.jsonl")
	messages, lines, err := message.Load(a.messagesPath)
	if err != nil {
		a.report(fmt.Errorf("%s: %w", a.text("Messages", "消息"), err))
	}
	a.messages, a.messageLines = messages, lines
	a.cutMessages(message.Keep)
}

// toolMessage keeps a message about the caller, or the terminal it names.
// It changes nothing the user works on, so any terminal may send one. One
// the file does not take still shows, as the error says, for the sender not
// to send it again.
func (a *app) toolMessage(req tool.Request, callers []int) error {
	if strings.TrimSpace(req.Text) == "" || len(req.Text) > tool.MaxMessage {
		return fmt.Errorf("text must be 1 to %d bytes, not blank", tool.MaxMessage)
	}
	caller := a.toolCaller(callers)
	if caller == 0 {
		return errors.New("caller is not a terminal of this Jun")
	}
	id := terminal.ID(req.About)
	if id == 0 {
		id = caller
	}
	s := a.terminals.Get(id)
	if s == nil || s.Exited() {
		return fmt.Errorf("no terminal %d", id)
	}
	m := message.Message{Time: time.Now(), Terminal: uint64(id), Project: s.ProjectName, Text: req.Text, Live: true}
	a.messages = append(a.messages, m)
	a.messages = a.messages[max(0, len(a.messages)-message.Keep):]
	if err := message.Append(a.messagesPath, m); err != nil {
		return fmt.Errorf("shown, but not saved: %w", err)
	}
	a.messageLines++
	a.cutMessages(2 * message.Keep)
	return nil
}

// cutMessages cuts the file back to what Jun keeps once it holds more than
// limit lines. No message is at stake, so a failure is only logged: the next
// call past the limit tries again.
func (a *app) cutMessages(limit int) {
	if a.messageLines <= limit {
		return
	}
	if err := message.Save(a.messagesPath, a.messages); err != nil {
		log.Printf("Jun: %v", err)
		return
	}
	a.messageLines = len(a.messages)
}

// messageTerminal is the running terminal the message is about, if any.
func (a *app) messageTerminal(m message.Message) (terminal.ID, bool) {
	id := terminal.ID(m.Terminal)
	s := a.terminals.Get(id)
	return id, m.Live && s != nil && !s.Exited()
}

// openMessage acts as an agent's row does, or shows a terminal running no
// agent, such as a loop's.
func (a *app) openMessage(e ui.Element, id terminal.ID) func() {
	if _, isAgent := a.agents[id]; isAgent {
		return a.agentClick(e, id)
	}
	return func() { a.revealSession(id) }
}
