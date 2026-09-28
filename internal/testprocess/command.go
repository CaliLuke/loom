// Package testprocess owns subprocess lifetimes in integration test harnesses.
package testprocess

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"time"
)

// Cmd runs a command with cleanup of its process group on supported Unix hosts.
// Configure the embedded Cmd before Start; do not call its execution methods
// directly or override Cancel or SysProcAttr. Start and Wait must each be called
// at most once. Every successful Start must be followed by Wait.
type Cmd struct {
	*exec.Cmd
	guard *processGuard
}

// CommandContext constructs a command whose descendants share its lifetime.
// On Unix, loss of the parent process also kills the process group. Descendants
// must not detach into another group. Other platforms retain exec semantics.
func CommandContext(ctx context.Context, name string, args ...string) *Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.WaitDelay = time.Second
	return &Cmd{Cmd: cmd}
}

// Start establishes the process guardian before starting the command.
func (c *Cmd) Start() error {
	guard, err := startGuard(c.Cmd)
	if err != nil {
		return err
	}
	c.guard = guard
	if err := c.Cmd.Start(); err != nil {
		return errors.Join(err, guard.finish())
	}
	return nil
}

// Wait waits for the command and releases its guardian and descendants.
func (c *Cmd) Wait() error {
	return errors.Join(c.Cmd.Wait(), c.guard.finish())
}

// Run starts the command and waits for its completion.
func (c *Cmd) Run() error {
	if err := c.Start(); err != nil {
		return err
	}
	return c.Wait()
}

// CombinedOutput runs the command and returns its combined standard streams.
func (c *Cmd) CombinedOutput() ([]byte, error) {
	if c.Stdout != nil || c.Stderr != nil {
		return nil, errors.New("testprocess: output already configured")
	}
	var output bytes.Buffer
	c.Stdout = &output
	c.Stderr = &output
	err := c.Run()
	return output.Bytes(), err
}
