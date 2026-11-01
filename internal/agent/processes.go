package agent

import (
	"errors"
	"fmt"
	"maps"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// ParseForeground reads `ps -o pid=,pgid=,tpgid=,tty=,comm=` output and
// returns the agent leading each terminal's foreground process group.
func ParseForeground(output string) map[string]Kind {
	agents := map[string]Kind{}
	for _, line := range strings.Split(output, "\n") {
		f := strings.Fields(line)
		if len(f) < 5 || f[0] != f[2] || f[3] == "??" {
			continue
		}
		if kind, ok := KindOf(strings.Join(f[4:], " ")); ok {
			agents[f[3]] = kind
		}
	}
	return agents
}

// Foreground returns the agent in the foreground of each of the terminals.
func Foreground(ttys []string) (map[string]Kind, error) {
	if len(ttys) == 0 {
		return map[string]Kind{}, nil
	}
	out, err := ps("-o", "pid=,pgid=,tpgid=,tty=,comm=", "-t", strings.Join(ttys, ","))
	if err != nil {
		return nil, fmt.Errorf("list foreground processes: %w", err)
	}
	return ParseForeground(out), nil
}

// ParseBusy reads `ps -o pid=,pgid=,tpgid=,tty=,comm=` output and returns
// the command leading each terminal's foreground process group, for the
// terminals (tty → shell pid) where that is not their shell.
func ParseBusy(output string, shells map[string]int) map[string]string {
	busy := map[string]string{}
	for _, line := range strings.Split(output, "\n") {
		f := strings.Fields(line)
		if len(f) < 5 || f[0] != f[2] {
			continue
		}
		shell, ok := shells[f[3]]
		if pid, err := strconv.Atoi(f[0]); ok && err == nil && pid != shell {
			busy[f[3]] = filepath.Base(strings.Join(f[4:], " "))
		}
	}
	return busy
}

// Busy returns the command in the foreground of each of the terminals
// (tty → shell pid) whose shell is not idle.
func Busy(shells map[string]int) (map[string]string, error) {
	if len(shells) == 0 {
		return map[string]string{}, nil
	}
	out, err := ps("-o", "pid=,pgid=,tpgid=,tty=,comm=", "-t", strings.Join(slices.Collect(maps.Keys(shells)), ","))
	if err != nil {
		return nil, fmt.Errorf("list foreground processes: %w", err)
	}
	return ParseBusy(out, shells), nil
}

// TTYs returns the terminal of each running process.
func TTYs(pids []int) (map[int]string, error) {
	result := map[int]string{}
	if len(pids) == 0 {
		return result, nil
	}
	list := make([]string, len(pids))
	for i, pid := range pids {
		list[i] = strconv.Itoa(pid)
	}
	out, err := ps("-o", "pid=,tty=", "-p", strings.Join(list, ","))
	if err != nil {
		return nil, fmt.Errorf("list terminals: %w", err)
	}
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) != 2 || f[1] == "??" {
			continue
		}
		if pid, err := strconv.Atoi(f[0]); err == nil {
			result[pid] = f[1]
		}
	}
	return result, nil
}

// ps runs ps. It exits 1 when some of the processes or terminals asked
// for are gone, still printing the others.
func ps(args ...string) (string, error) {
	out, err := exec.Command("ps", args...).Output()
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		return string(out), nil
	}
	return string(out), err
}
