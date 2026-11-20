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
	"github.com/egoist/mygo/ui"
)

type agentInfo struct {
	Kind   agent.Kind
	Status agent.Status
}

// agentJob is what a poll reads, gathered on the UI thread.
type agentJob struct {
	pids    map[terminal.ID]int
	ttys    map[terminal.ID]string
	sources map[terminal.ID]*ghostty.Terminal
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
		sources: map[terminal.ID]*ghostty.Terminal{}}
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
	maps.DeleteFunc(a.viewSizes, func(id terminal.ID, _ ui.Rect) bool { return a.terminals.Get(id) == nil })
	return job
}

// pollAgents runs off the UI thread: ps and the terminals' reads take time.
func (a *app) pollAgents(job agentJob) {
	ttys, agents, err := readAgents(job)
	a.window.Update(func() {
		a.agentPolling = false
		if err != nil {
			log.Printf("Jun: poll agents: %v", err)
			return
		}
		a.ttys, a.agents = ttys, agents
		a.sendHeld()
		now := time.Now()
		a.publishAgents(now)
		a.fireTimers(now)
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

// agentSessions lists the agents as the Agents view does.
func (a *app) agentSessions() []*terminal.Session {
	return slices.Concat(a.agentGroups()...)
}

// agentGroups lists the agents in groups, each a lead and the agents it
// leads, however far down; an agent no running agent leads is a group of
// its own. A group comes as urgently as its most urgent agent, then by its
// lead's project and id; in it the lead comes first, the rest by id.
func (a *app) agentGroups() [][]*terminal.Session {
	projectOrder := func(name string) int {
		return slices.IndexFunc(a.state.Projects, func(p *layout.Project) bool { return p.Name == name })
	}
	// The terminals come in the order they started, that of their ids.
	var leads []*terminal.Session
	led := map[terminal.ID][]*terminal.Session{}
	for _, s := range a.terminals.All() {
		if _, ok := a.agents[s.ID]; !ok {
			continue
		}
		if root := a.agentRoot(s.ID); root == s.ID {
			leads = append(leads, s)
		} else {
			led[root] = append(led[root], s)
		}
	}
	groups := make([][]*terminal.Session, len(leads))
	for i, lead := range leads {
		groups[i] = append([]*terminal.Session{lead}, led[lead.ID]...)
	}
	slices.SortFunc(groups, func(x, y []*terminal.Session) int {
		return cmp.Or(
			cmp.Compare(statusOrder[a.urgency(x)], statusOrder[a.urgency(y)]),
			cmp.Compare(projectOrder(x[0].ProjectName), projectOrder(y[0].ProjectName)),
			cmp.Compare(x[0].ID, y[0].ID))
	})
	return groups
}

// agentRoot is the agent at the top of the leads above the agent, going up
// while the lead runs an agent.
func (a *app) agentRoot(id terminal.ID) terminal.ID {
	for {
		lead := a.lead(id)
		if _, ok := a.agents[lead]; !ok {
			return id
		}
		id = lead
	}
}

// urgency is the status of the group's most urgent agent.
func (a *app) urgency(group []*terminal.Session) agent.Status {
	return a.agents[slices.MinFunc(group, func(x, y *terminal.Session) int {
		return cmp.Compare(statusOrder[a.agents[x.ID].Status], statusOrder[a.agents[y.ID].Status])
	}).ID].Status
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

// revealTab goes to the tab, in whatever project and perspective it is.
func (a *app) revealTab(view layout.ViewID) error {
	previous := a.state.ActiveProject
	region, err := a.state.RevealView(view)
	if err != nil {
		return err
	}
	a.layoutChanged()
	a.focusRegion(region)
	if a.state.ActiveProject != previous {
		a.checkCurrentDirectory()
	}
	return nil
}

// enterProject makes the project the current one, and tells whether it is.
func (a *app) enterProject(name string) bool {
	if name != a.state.ActiveProject {
		a.selectProject(name)
	}
	return a.state.ActiveProject == name
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
		a.report(a.revealTab(view))
		return
	}
	if !a.enterProject(session.ProjectName) {
		return
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
