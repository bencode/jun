package goals

import (
	"slices"
	"strings"
)

// SelectGoals is the goals the filter picks, in the tree's order: each
// goal, then those under it.
func SelectGoals(b Book, f GoalFilter) []Goal {
	return slices.DeleteFunc(treeOrder(b), func(g Goal) bool {
		ended := g.State != GoalOpen
		return !picked(f.All, f.States, g.State, ended) || !matches(f.Match, g.Title, g.Outcome, g.Text)
	})
}

// SelectTasks is the tasks the filter picks, by their goals in the tree's
// order, then in their goal's.
func SelectTasks(b Book, f TaskFilter) []Task {
	rank := map[int]int{}
	for i, g := range treeOrder(b) {
		rank[g.ID] = i
	}
	under := map[int]bool{}
	for _, g := range b.Goals {
		under[g.ID] = f.Goal == 0 || slices.ContainsFunc(b.Chain(g.ID), func(above Goal) bool { return above.ID == f.Goal })
	}
	tasks := slices.DeleteFunc(slices.Clone(b.Tasks), func(t Task) bool {
		ended := t.State == TaskDone || t.State == TaskDropped
		return !picked(f.All, f.States, t.State, ended) || !under[t.Goal] ||
			f.Owner != "" && !strings.EqualFold(f.Owner, t.Owner) || f.Project != "" && !strings.EqualFold(f.Project, t.Project) ||
			!matches(f.Match, t.Title, t.Outcome, t.Text)
	})
	slices.SortStableFunc(tasks, func(x, y Task) int { return rank[x.Goal] - rank[y.Goal] })
	return tasks
}

// treeOrder is the goals, each followed by those under it.
func treeOrder(b Book) []Goal {
	var order []Goal
	var walk func(parent int)
	walk = func(parent int) {
		for _, g := range b.Goals {
			if g.Parent == parent && len(order) < len(b.Goals) {
				order = append(order, g)
				walk(g.ID)
			}
		}
	}
	walk(0)
	return order
}

// picked tells whether the state is one asked for: those given, all, or
// by default those not ended.
func picked[S comparable](all bool, states []S, state S, ended bool) bool {
	if len(states) > 0 {
		return slices.Contains(states, state)
	}
	return all || !ended
}

func matches(match string, fields ...string) bool {
	return match == "" || slices.ContainsFunc(fields, func(f string) bool {
		return strings.Contains(strings.ToLower(f), strings.ToLower(match))
	})
}
