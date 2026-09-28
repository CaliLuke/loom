//go:build unix

package testprocess

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

type processGuard struct {
	command *exec.Cmd
	lease   *os.File
	once    sync.Once
	err     error
}

func startGuard(command *exec.Cmd) (*processGuard, error) {
	reader, writer, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("create process lease: %w", err)
	}
	// Only this process retains the write end: os.Pipe descriptors are
	// close-on-exec. A fork in progress temporarily inherits the writer until
	// exec: Go applies Setpgid before exec closes it. Thus EOF cannot race ahead
	// of that child joining the group. No executed child retains the writer.
	// Parent panic or termination closes the remaining writer automatically.
	guardian := exec.Command("/bin/sh", "-c", "read lease; exec /bin/kill -KILL -- -$$")
	guardian.Stdin = reader
	guardian.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	guard := &processGuard{command: guardian, lease: writer}
	startErr := guardian.Start()
	closeErr := reader.Close()
	if startErr != nil {
		return nil, errors.Join(startErr, closeErr, guard.release())
	}
	if closeErr != nil {
		return nil, errors.Join(closeErr, guard.finish())
	}
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: guardian.Process.Pid}
	command.Cancel = guard.release
	return guard, nil
}

func (g *processGuard) release() error {
	g.once.Do(func() {
		g.err = g.lease.Close()
	})
	return g.err
}

func (g *processGuard) finish() error {
	if g == nil {
		return nil
	}
	err := g.release()
	waitErr := g.command.Wait()
	var exitErr *exec.ExitError
	if errors.As(waitErr, &exitErr) {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok && status.Signaled() && status.Signal() == syscall.SIGKILL {
			waitErr = nil
		}
	}
	return errors.Join(err, waitErr)
}
