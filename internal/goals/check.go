package goals

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Check tells what is wrong with the book, if anything: IDs are each
// one item's and none past the last given; a goal's parent is a goal not
// under it; a task's goal is there; fields hold their forms; a state goes
// with its times; a session touches a task once.
func Check(b Book) error {
	goals, tasks := map[int]bool{}, map[int]bool{}
	for _, g := range b.Goals {
		if g.ID < 1 || g.ID > b.LastGoal || goals[g.ID] {
			return fmt.Errorf("goal %d: ID is not its own, from 1 to the last given, %d", g.ID, b.LastGoal)
		}
		goals[g.ID] = true
		if err := checkGoal(g); err != nil {
			return fmt.Errorf("goal G%d: %w", g.ID, err)
		}
	}
	for _, g := range b.Goals {
		if g.Parent != 0 && (!goals[g.Parent] || len(b.Chain(g.ID)) > len(b.Goals)) {
			return fmt.Errorf("goal G%d: parent %d is no goal it can serve", g.ID, g.Parent)
		}
	}
	for _, t := range b.Tasks {
		if t.ID < 1 || t.ID > b.LastTask || tasks[t.ID] {
			return fmt.Errorf("task %d: ID is not its own, from 1 to the last given, %d", t.ID, b.LastTask)
		}
		tasks[t.ID] = true
		if !goals[t.Goal] {
			return fmt.Errorf("task T%d: goal %d is not there", t.ID, t.Goal)
		}
		if err := checkTask(t); err != nil {
			return fmt.Errorf("task T%d: %w", t.ID, err)
		}
	}
	return nil
}

func checkGoal(g Goal) error {
	if !slices.Contains(GoalStates, g.State) {
		return fmt.Errorf("no state %q", g.State)
	}
	if (g.State != GoalOpen) != (g.Ended != nil) {
		return fmt.Errorf("endedAt goes with done and dropped only")
	}
	if g.Due != "" && !isDate(g.Due) {
		return fmt.Errorf("due %q is not a date as 2026-11-28", g.Due)
	}
	if g.Period != "" && !isPeriod(g.Period) {
		return fmt.Errorf("period %q is none of a month as 2026-11, an ISO week as 2026-W48 or a day as 2026-11-23", g.Period)
	}
	return checkFields(g.Title, g.Outcome, true, g.Text, g.Link)
}

func checkTask(t Task) error {
	if !slices.Contains(TaskStates, t.State) {
		return fmt.Errorf("no state %q", t.State)
	}
	if (t.State == TaskDone || t.State == TaskDropped) != (t.Ended != nil) {
		return fmt.Errorf("endedAt goes with done and dropped only")
	}
	if t.State == TaskTodo && t.Started != nil {
		return fmt.Errorf("a task to do has no startedAt")
	}
	seen := map[[2]string]bool{}
	for _, s := range t.Sessions {
		key := [2]string{s.Agent, s.ID}
		if s.Agent == "" || s.ID == "" || seen[key] {
			return fmt.Errorf("%s session %q is not one of its own", s.Agent, s.ID)
		}
		seen[key] = true
	}
	return checkFields(t.Title, t.Outcome, false, t.Text, t.Link)
}

// checkFields checks what goals and tasks both have.
func checkFields(title, outcome string, outcomeNeeded bool, text, link string) error {
	if !isLine(title, MaxTitle) {
		return fmt.Errorf("title must be one line of 1 to %d bytes", MaxTitle)
	}
	if (outcome != "" || outcomeNeeded) && !isLine(outcome, MaxOutcome) {
		return fmt.Errorf("outcome must be one line of 1 to %d bytes", MaxOutcome)
	}
	if len(text) > MaxText {
		return fmt.Errorf("text has more than %d bytes: write it in a file and link to that", MaxText)
	}
	file, isFile := strings.CutPrefix(link, "file:")
	project, isProject := strings.CutPrefix(link, "project:")
	if link != "" && !(isFile && filepath.IsAbs(file)) && !(isProject && strings.TrimSpace(project) != "") {
		return fmt.Errorf("link %q is neither file:<absolute path> nor project:<name>", link)
	}
	return nil
}

func isLine(s string, limit int) bool {
	return strings.TrimSpace(s) != "" && !strings.ContainsAny(s, "\r\n") && len(s) <= limit
}

func isDate(s string) bool {
	_, err := time.Parse(time.DateOnly, s)
	return err == nil
}

var isoWeek = regexp.MustCompile(`^(\d{4})-W(\d{2})$`)

// isPeriod tells a month, an ISO week of a year that has it, or a day.
func isPeriod(s string) bool {
	if _, err := time.Parse("2006-01", s); err == nil {
		return true
	}
	if m := isoWeek.FindStringSubmatch(s); m != nil {
		year, _ := strconv.Atoi(m[1])
		week, _ := strconv.Atoi(m[2])
		_, weeks := time.Date(year, time.December, 28, 0, 0, 0, 0, time.UTC).ISOWeek()
		return week >= 1 && week <= weeks
	}
	return isDate(s)
}
