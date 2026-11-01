package app

import (
	"cmp"
	"fmt"
	"log"
	"maps"
	"slices"
	"time"

	"github.com/bencode/jun/internal/agent"
	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/terminal"
	ghostty "github.com/egoist/mygo/plugins/terminal"
)

type agentInfo struct {
	Kind   agent.Kind
	Status agent.Status
}

// agentJob is what a poll reads, gathered on the UI thread.
type agentJob struct {
	pids     map[terminal.ID]int
	ttys     map[terminal.ID]string
	sources  map[terminal.ID]*ghostty.Terminal
	previews map[terminal.ID]*preview
}

const agentPollInterval = time.Second

// startAgentPoller polls the shells' foreground processes until stop closes.
func (a *app) startAgentPoller(stop <-chan struct{}) {
	ticker := time.NewTicker(agentPollInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				a.window.Update(func() {
					if a.agentPolling {
						return
					}
					a.agentPolling = true
					go a.pollAgents(a.agentJob())
				})
			}
		}
	}()
}

func (a *app) agentJob() agentJob {
	job := agentJob{pids: map[terminal.ID]int{}, ttys: map[terminal.ID]string{},
		sources: map[terminal.ID]*ghostty.Terminal{}, previews: map[terminal.ID]*preview{}}
	for _, session := range a.terminals.All() {
		select {
		case <-session.Terminal.Done():
			continue
		default:
		}
		job.sources[session.ID] = session.Terminal
		if tty, ok := a.ttys[session.ID]; ok {
			job.ttys[session.ID] = tty
		} else if pid, ok := session.ShellPID(); ok {
			job.pids[session.ID] = pid
		}
	}
	a.closeStalePreviews()
	for id, p := range a.previews {
		if job.sources[id] != nil && time.Since(p.seen) < 2*agentPollInterval {
			job.previews[id] = p
		}
	}
	return job
}

// pollAgents runs off the UI thread: ps and the terminals' reads take time.
func (a *app) pollAgents(job agentJob) {
	ttys, agents, err := readAgents(job)
	for id, p := range job.previews {
		p.refresh(job.sources[id], 5)
	}
	a.window.Update(func() {
		a.agentPolling = false
		if err != nil {
			log.Printf("Jun: poll agents: %v", err)
			return
		}
		a.ttys, a.agents = ttys, agents
	})
}

func readAgents(job agentJob) (map[terminal.ID]string, map[terminal.ID]agentInfo, error) {
	ttys := maps.Clone(job.ttys)
	byPID, err := agent.TTYs(slices.Collect(maps.Values(job.pids)))
	if err != nil {
		return nil, nil, err
	}
	for id, pid := range job.pids {
		if tty, ok := byPID[pid]; ok {
			ttys[id] = tty
		}
	}
	foreground, err := agent.Foreground(slices.Collect(maps.Values(ttys)))
	if err != nil {
		return nil, nil, err
	}
	agents := map[terminal.ID]agentInfo{}
	for id, tty := range ttys {
		if kind, ok := foreground[tty]; ok {
			source := job.sources[id]
			agents[id] = agentInfo{kind, agent.DetectStatus(kind, source.Title(), source.Text())}
		}
	}
	return ttys, agents, nil
}

var statusOrder = map[agent.Status]int{agent.Waiting: 0, agent.Working: 1, agent.Idle: 2}

// agentSessions lists the agents most in need of the user first.
func (a *app) agentSessions() []*terminal.Session {
	projectOrder := func(name string) int {
		return slices.IndexFunc(a.state.Projects, func(p *layout.Project) bool { return p.Name == name })
	}
	sessions := slices.DeleteFunc(a.terminals.All(), func(s *terminal.Session) bool {
		_, ok := a.agents[s.ID]
		return !ok
	})
	slices.SortFunc(sessions, func(x, y *terminal.Session) int {
		return cmp.Or(
			cmp.Compare(statusOrder[a.agents[x.ID].Status], statusOrder[a.agents[y.ID].Status]),
			cmp.Compare(projectOrder(x.ProjectName), projectOrder(y.ProjectName)),
			cmp.Compare(x.ID, y.ID))
	})
	return sessions
}

// waitingByProject counts the agents waiting for the user, by project.
func (a *app) waitingByProject() map[string]int {
	counts := map[string]int{}
	for id, info := range a.agents {
		if session := a.terminals.Get(id); session != nil && info.Status == agent.Waiting {
			counts[session.ProjectName]++
		}
	}
	return counts
}

// revealSession shows the session's tab. When its tab was closed, it
// reopens where new tabs go, in the session's project.
func (a *app) revealSession(id terminal.ID) {
	session := a.terminals.Get(id)
	if session == nil {
		a.report(fmt.Errorf("terminal %d is gone", id))
		return
	}
	for view, bound := range a.bindings {
		if bound != id {
			continue
		}
		previous := a.state.ActiveProject
		region, err := a.state.RevealView(view)
		if err != nil {
			a.report(err)
			return
		}
		a.layoutChanged()
		a.focusRegion(region)
		if a.state.ActiveProject != previous {
			a.checkCurrentDirectory()
		}
		return
	}
	if session.ProjectName != a.state.ActiveProject {
		a.selectProject(session.ProjectName)
		if a.state.ActiveProject != session.ProjectName {
			return
		}
	}
	target := a.tabsTarget()
	if target == nil {
		a.report(fmt.Errorf("no region to reopen terminal %d in", id))
		return
	}
	view, err := a.state.OpenView(target.ID, layout.ViewTerminal, session.ProjectName)
	if err != nil {
		a.report(err)
		return
	}
	a.bindings[view] = id
	a.layoutChanged()
	a.focusRegion(target.ID)
}
