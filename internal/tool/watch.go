package tool

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

// Subscribe starts a watch: events holds what the agents are now, then what
// changes, and closes when the watcher falls behind. cancel ends it.
type Subscribe func() (events <-chan Event, cancel func())

const eventWriteTimeout = 5 * time.Second

// streamEvents writes the events to conn until the caller goes, Jun closes
// or the caller falls behind.
func streamEvents(conn *net.UnixConn, subscribe Subscribe, closed <-chan struct{}) error {
	if err := conn.SetDeadline(time.Time{}); err != nil {
		return err
	}
	events, cancel := subscribe()
	defer cancel()
	gone := make(chan struct{})
	go func() {
		defer close(gone)
		if _, err := io.Copy(io.Discard, conn); err != nil && !errors.Is(err, net.ErrClosed) {
			log.Printf("Jun: agent tools: watch: %v", err)
		}
	}()
	encoder := json.NewEncoder(conn)
	encoder.SetEscapeHTML(false)
	for {
		select {
		case event, ok := <-events:
			if !ok {
				return errors.New("the caller fell behind")
			}
			if err := conn.SetWriteDeadline(time.Now().Add(eventWriteTimeout)); err != nil {
				return err
			}
			if err := encoder.Encode(event); err != nil {
				return err
			}
		case <-gone:
			return nil
		case <-closed:
			return nil
		}
	}
}

// watch copies Jun's events to w as they come; only an error ends it.
func watch(path string, w io.Writer) (err error) {
	conn, err := net.DialTimeout("unix", path, 2*time.Second)
	if err != nil {
		return fmt.Errorf("cannot reach Jun at %s: %w", path, err)
	}
	defer func() { err = errors.Join(err, conn.Close()) }()
	if err := json.NewEncoder(conn).Encode(Request{Op: OpWatch}); err != nil {
		return err
	}
	lines := bufio.NewReader(conn)
	for {
		line, err := lines.ReadBytes('\n')
		if errors.Is(err, io.EOF) {
			return errors.New("the watch ended: Jun quit, or this reader fell behind")
		}
		if err != nil {
			return err
		}
		var answer Response // a refusal, in place of the events
		if err := json.Unmarshal(line, &answer); err != nil {
			return fmt.Errorf("bad event: %w", err)
		}
		if answer.Error != "" {
			return errors.New(answer.Error)
		}
		if _, err := w.Write(line); err != nil {
			return err
		}
	}
}
