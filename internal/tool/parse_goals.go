package tool

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
)

// parseGoal reads what follows `goal`: a list, a goal by id, or add,
// update or delete.
func parseGoal(args []string, stdin io.Reader) (Request, error) {
	return parseItem(args, stdin, itemCommand{
		name: "goal", list: OpGoals, one: OpGoal, add: OpAddGoal, update: OpUpdateGoal, remove: OpDeleteGoal,
		states:  []string{"open", "done", "dropped"},
		filters: []string{"--match"},
		adds:    []string{"--parent", "--outcome", "--due", "--period", "--link", "--text"},
		updates: []string{"--title", "--outcome", "--parent", "--due", "--period", "--link", "--text", "--rank", "--state"},
		usage: "goal takes nothing, --all or --state <state> (again for more) and --match <text>; <id>; " +
			"add [--parent <id>] --outcome <outcome> [--due <date>] [--period <period>] [--link <resource>] [--text <text>|-] <title>; " +
			"update <id> with --title, --outcome, --parent, --due, --period, --link, --text, --rank or --state; or delete <id>",
	})
}

// parseTask reads what follows `task`, as parseGoal.
func parseTask(args []string, stdin io.Reader) (Request, error) {
	return parseItem(args, stdin, itemCommand{
		name: "task", list: OpTasks, one: OpTask, add: OpAddTask, update: OpUpdateTask, remove: OpDeleteTask,
		states:  []string{"todo", "doing", "done", "dropped"},
		filters: []string{"--owner", "--project", "--goal", "--match"},
		adds:    []string{"--goal", "--owner", "--project", "--outcome", "--link", "--text"},
		updates: []string{"--state", "--owner", "--goal", "--project", "--title", "--outcome", "--link", "--text", "--rank"},
		usage: "task takes nothing, --all or --state <state> (again for more), --owner <role>, --project <name>, --goal <id> and --match <text>; <id>; " +
			"add --goal <id> [--owner <role>] [--project <name>] [--outcome <outcome>] [--link <resource>] [--text <text>|-] <title>; " +
			"update <id> with --state, --owner, --goal, --project, --title, --outcome, --link, --text or --rank; or delete <id>",
	})
}

// itemCommand is what goal and task each take.
type itemCommand struct {
	name                           string
	list, one, add, update, remove Op
	states, filters, adds, updates []string
	usage                          string
}

func parseItem(args []string, stdin io.Reader, c itemCommand) (Request, error) {
	usage := errors.New(c.usage)
	if len(args) == 0 || isFlag(args[0]) {
		return parseItemFilter(args, c)
	}
	if len(args) == 1 {
		id, err := itemID(c.name, args[0])
		return Request{Op: c.one, Item: &Item{ID: id}}, err
	}
	if args[0] == "add" {
		item, rest, err := parseItemFlags(args[1:], stdin, c, c.adds)
		if err != nil {
			return Request{}, err
		}
		if len(rest) != 1 || isFlag(rest[0]) {
			return Request{}, usage
		}
		title, err := itemLine("title", rest[0], MaxTitle)
		if err != nil {
			return Request{}, err
		}
		item.Title = &title
		if c.add == OpAddGoal && item.Outcome == nil || c.add == OpAddTask && item.Goal == nil {
			return Request{}, usage
		}
		return Request{Op: c.add, Item: &item}, nil
	}
	if args[0] != "update" && args[0] != "delete" || len(args) < 2 {
		return Request{}, usage
	}
	id, err := itemID(c.name, args[1])
	if err != nil {
		return Request{}, err
	}
	if args[0] == "delete" {
		if len(args) != 2 {
			return Request{}, usage
		}
		return Request{Op: c.remove, Item: &Item{ID: id}}, nil
	}
	item, rest, err := parseItemFlags(args[2:], stdin, c, c.updates)
	if err != nil {
		return Request{}, err
	}
	if len(rest) != 0 || len(args) == 2 {
		return Request{}, usage
	}
	item.ID = id
	return Request{Op: c.update, Item: &item}, nil
}

// parseItemFilter reads a list's options: --all or any --state, then the
// filters, each once.
func parseItemFilter(args []string, c itemCommand) (Request, error) {
	f := Filter{}
	seen := map[string]bool{}
	for len(args) > 0 {
		flag := args[0]
		if flag == "--all" && !f.All && len(f.States) == 0 {
			f.All, args = true, args[1:]
			continue
		}
		if len(args) < 2 || seen[flag] || flag != "--state" && !slices.Contains(c.filters, flag) || flag == "--state" && f.All {
			return Request{}, errors.New(c.usage)
		}
		value := strings.TrimSpace(args[1])
		if flag == "--state" {
			if !slices.Contains(c.states, value) {
				return Request{}, fmt.Errorf("state must be one of %v, not %q", c.states, value)
			}
			f.States = append(f.States, value)
		} else if flag == "--goal" {
			id, err := itemID("goal", value)
			if err != nil {
				return Request{}, err
			}
			f.Goal = id
		} else if value == "" {
			return Request{}, fmt.Errorf("%s takes a value, not blank", flag)
		} else {
			*map[string]*string{"--owner": &f.Owner, "--project": &f.Project, "--match": &f.Match}[flag] = value
		}
		seen[flag] = flag != "--state"
		args = args[2:]
	}
	return Request{Op: c.list, Filter: &f}, nil
}

// parseItemFlags reads the options of add or update, those allowed, each
// once, and returns what follows them.
func parseItemFlags(args []string, stdin io.Reader, c itemCommand, allowed []string) (Item, []string, error) {
	item := Item{}
	strs := map[string]**string{"--title": &item.Title, "--outcome": &item.Outcome, "--text": &item.Text, "--link": &item.Link,
		"--due": &item.Due, "--period": &item.Period, "--project": &item.Project, "--owner": &item.Owner, "--state": &item.State}
	seen := map[string]bool{}
	for len(args) > 1 && slices.Contains(allowed, args[0]) && !seen[args[0]] {
		flag, value := args[0], args[1]
		seen[flag], args = true, args[2:]
		if flag == "--parent" || flag == "--goal" || flag == "--rank" {
			n, err := strconv.Atoi(value)
			if err != nil || n < 0 || n == 0 && flag != "--parent" {
				return Item{}, nil, fmt.Errorf("%s must be a positive integer, or 0 for --parent to have none: %q", flag, value)
			}
			if flag == "--rank" {
				item.Rank = n
			} else if flag == "--parent" {
				item.Parent = &n
			} else {
				item.Goal = &n
			}
			continue
		}
		text, err := itemValue(flag, value, stdin, c)
		if err != nil {
			return Item{}, nil, err
		}
		*strs[flag] = &text
	}
	if len(args) > 0 && isFlag(args[0]) {
		return Item{}, nil, errors.New(c.usage)
	}
	return item, args, nil
}

// itemValue checks an option's value as far as its form goes: "" clears
// what may be cleared.
func itemValue(flag, value string, stdin io.Reader, c itemCommand) (string, error) {
	if flag == "--title" {
		return itemLine("title", value, MaxTitle)
	}
	if flag == "--state" {
		if !slices.Contains(c.states, value) {
			return "", fmt.Errorf("state must be one of %v, not %q", c.states, value)
		}
		return value, nil
	}
	if value == "" {
		return "", nil
	}
	if flag == "--outcome" {
		return itemLine("outcome", value, MaxOutcome)
	}
	if flag == "--text" {
		return readText(value, stdin, MaxMessage)
	}
	if flag == "--link" {
		target, err := parseTarget(value)
		if err != nil || target.Kind != TargetFile && target.Kind != TargetProject {
			return "", fmt.Errorf("--link takes file:<path> or project:<name>, which last: %q", value)
		}
		return target.String(), nil
	}
	return strings.TrimSpace(value), nil
}

func itemLine(field, value string, limit int) (string, error) {
	if strings.TrimSpace(value) == "" || strings.ContainsAny(value, "\r\n") || len(value) > limit {
		return "", fmt.Errorf("%s must be one line of 1 to %d bytes", field, limit)
	}
	return value, nil
}

func itemID(name, arg string) (int, error) {
	id, err := strconv.Atoi(arg)
	if err != nil || id < 1 {
		return 0, fmt.Errorf("%s id must be a positive integer: %q", name, arg)
	}
	return id, nil
}
