package tool

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"slices"
	"time"

	"golang.org/x/sys/unix"
)

// Handler answers a request. callers are the process that connected and
// its ancestors, nearest first, down to launchd's child.
type Handler func(req Request, callers []int) Response

type Server struct {
	listener *net.UnixListener
}

const exchangeTimeout = 10 * time.Second

// Start serves handle at path, a socket only the user can connect to. A
// socket left there by a Jun that ended is replaced; one that answers is
// another Jun's, and Start fails.
func Start(path string, handle Handler) (*Server, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return nil, fmt.Errorf("%s is not a socket", path)
		}
		if conn, err := net.DialTimeout("unix", path, time.Second); err == nil {
			return nil, errors.Join(fmt.Errorf("another Jun serves %s", path), conn.Close())
		}
		if err := os.Remove(path); err != nil {
			return nil, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return nil, errors.Join(err, listener.Close())
	}
	s := &Server{listener: listener}
	go s.serve(handle)
	return s, nil
}

// Close stops serving and removes the socket; connections being answered
// finish on their own.
func (s *Server) Close() error { return s.listener.Close() }

func (s *Server) serve(handle Handler) {
	for {
		conn, err := s.listener.AcceptUnix()
		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			log.Printf("Jun: agent tools stopped: %v", err)
			return
		}
		go answer(conn, handle)
	}
}

func answer(conn *net.UnixConn, handle Handler) {
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Jun: agent tools: %v", err)
		}
	}()
	if err := conn.SetDeadline(time.Now().Add(exchangeTimeout)); err != nil {
		log.Printf("Jun: agent tools: %v", err)
		return
	}
	if err := json.NewEncoder(conn).Encode(respond(conn, handle)); err != nil {
		log.Printf("Jun: agent tools: %v", err)
	}
}

func respond(conn *net.UnixConn, handle Handler) Response {
	uid, pid, err := peer(conn)
	if err != nil {
		return Response{Error: fmt.Sprintf("caller: %v", err)}
	}
	if uid != uint32(os.Getuid()) {
		return Response{Error: "not your user"}
	}
	callers, err := ancestry(pid, parent)
	if err != nil {
		return Response{Error: fmt.Sprintf("caller: %v", err)}
	}
	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		return Response{Error: fmt.Sprintf("read request: %v", err)}
	}
	var req Request
	if err := json.Unmarshal(line, &req); err != nil {
		return Response{Error: "bad request"}
	}
	return handle(req, callers)
}

// peer is who connected, as the system tells: neither can be forged.
func peer(conn *net.UnixConn) (uint32, int, error) {
	raw, err := conn.SyscallConn()
	if err != nil {
		return 0, 0, err
	}
	var uid uint32
	var pid int
	var peerErr error
	if err := raw.Control(func(fd uintptr) {
		cred, err := unix.GetsockoptXucred(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERCRED)
		if err != nil {
			peerErr = err
			return
		}
		uid = cred.Uid
		pid, peerErr = unix.GetsockoptInt(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERPID)
	}); err != nil {
		return 0, 0, err
	}
	return uid, pid, peerErr
}

func parent(pid int) (int, error) {
	info, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		return 0, err
	}
	return int(info.Eproc.Ppid), nil
}

// ancestry is pid and its ancestors, nearest first, ending before launchd.
func ancestry(pid int, parent func(int) (int, error)) ([]int, error) {
	var chain []int
	for pid > 1 {
		if slices.Contains(chain, pid) {
			return nil, fmt.Errorf("process %d is its own ancestor", pid)
		}
		chain = append(chain, pid)
		next, err := parent(pid)
		if err != nil {
			return nil, fmt.Errorf("parent of process %d: %w", pid, err)
		}
		pid = next
	}
	return chain, nil
}
