package app

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/tool"
)

// messagesKept is how many messages Jun shows.
const messagesKept = 500

// message is what a terminal told the user. It lives as long as this run of
// Jun, as the terminals it links to do.
type message struct {
	ID      uint64
	Time    time.Time
	Kind    tool.MessageKind
	Title   string
	Text    string
	Project *layout.Project // the link's, told when it came; a pointer, as a project's name may change
	Link    *tool.Target    // where it takes the user; none for plain news
	Whole   bool            // a long text shows unfolded
}

// removeOldMessages deletes the file earlier versions kept messages in.
func removeOldMessages(configPath string) {
	if configPath == "" {
		return
	}
	err := os.Remove(filepath.Join(filepath.Dir(configPath), "messages.jsonl"))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Printf("Jun: %v", err)
	}
}

// toolMessage keeps a message, refusing one whose link leads nowhere. It
// changes nothing the user works on, so any terminal may send one.
func (a *app) toolMessage(req tool.Request, callers []int) error {
	if strings.TrimSpace(req.Title+req.Text) == "" || len(req.Text) > tool.MaxMessage || len(req.Title) > tool.MaxTitle {
		return fmt.Errorf("a title of at most %d bytes or a text of at most %d, not blank", tool.MaxTitle, tool.MaxMessage)
	}
	kind := cmp.Or(req.Kind, tool.MessageInfo)
	if !slices.Contains(tool.MessageKinds, kind) {
		return fmt.Errorf("no kind %q", kind)
	}
	caller := a.toolCaller(callers)
	if caller == 0 {
		return errors.New("caller is not a terminal of this Jun")
	}
	var project *layout.Project
	if req.Target != nil {
		link, ok := messageLinks[req.Target.Kind]
		if !ok {
			return fmt.Errorf("a message cannot link to %s", req.Target.Kind)
		}
		var err error
		if project, err = link.project(a, *req.Target, a.project(a.terminals.Get(caller).ProjectName)); err != nil {
			return err
		}
	}
	a.lastMessage++
	a.messages = append(a.messages, message{ID: a.lastMessage, Time: time.Now(), Kind: kind, Title: req.Title,
		Text: req.Text, Project: project, Link: req.Target})
	a.messages = a.messages[max(0, len(a.messages)-messagesKept):]
	return nil
}

// setMessageWhole shows a long text whole, or folds it. A row folding
// shrinks under the list's place in it, so the list goes back to the row.
func (a *app) setMessageWhole(id uint64, whole bool) {
	i := slices.IndexFunc(a.messages, func(m message) bool { return m.ID == id })
	if i < 0 {
		return
	}
	a.messages[i].Whole = whole
	if !whole {
		a.messagesList.ScrollIntoView(len(a.messages) - 1 - i)
	}
}

// messageLive tells whether the message still leads somewhere.
func (a *app) messageLive(m message) bool {
	return m.Link != nil && messageLinks[m.Link.Kind].live(a, m)
}

// openMessage takes the user to what the message links to. With reveal, a
// terminal shows in its tab even when an agent runs in it.
func (a *app) openMessage(m message, reveal bool) {
	a.report(messageLinks[m.Link.Kind].show(a, m, reveal))
}
