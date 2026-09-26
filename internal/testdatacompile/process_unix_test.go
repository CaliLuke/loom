//go:build !windows

package testdatacompile

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// cancelProcessGroup ends the CLI's compiler/generator descendants too. Killing
// only the CLI leaves those children running with inherited stdout pipes.
func cancelProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
}

// TestCommandTimeoutEndsDescendants proves that cancellation closes child pipes.
func TestCommandTimeoutEndsDescendants(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := runCommand(ctx, t.TempDir(), nil, "sh", "-c", "sleep 10 & wait")
	require.Error(t, err)
	require.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)
	require.Less(t, time.Since(start), 2*time.Second, "a surviving child retains the output pipe")
}
