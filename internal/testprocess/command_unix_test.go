//go:build unix

package testprocess

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCommandOutput(t *testing.T) {
	for _, tc := range []struct {
		name   string
		code   string
		status int
	}{
		{"success", "printf out; printf err >&2", 0},
		{"failure", "printf out; printf err >&2; exit 7", 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := CommandContext(t.Context(), "/bin/sh", "-c", tc.code)
			output, err := cmd.CombinedOutput()
			require.Equal(t, "outerr", string(output))
			if tc.status == 0 {
				require.NoError(t, err)
			} else {
				var exitErr *exec.ExitError
				require.ErrorAs(t, err, &exitErr)
				require.Equal(t, tc.status, exitErr.ExitCode())
			}
		})
	}
}

func TestCommandReapsDescendants(t *testing.T) {
	for _, mode := range []string{"cancel", "normal-exit", "inherited-output"} {
		t.Run(mode, func(t *testing.T) {
			pidFile := filepath.Join(t.TempDir(), "pid")
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			script := "sleep 60 & echo $! > \"$1\"; wait"
			if mode != "cancel" {
				script = "sleep 60 & echo $! > \"$1\"; exit 0"
			}
			cmd := CommandContext(ctx, "/bin/sh", "-c", script, "test", pidFile)
			var output strings.Builder
			if mode == "inherited-output" {
				cmd.Stdout = &output
			}
			require.NoError(t, cmd.Start())
			pid := readPID(t, pidFile)
			if mode == "cancel" {
				cancel()
			}
			err := cmd.Wait()
			if mode == "normal-exit" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			requireGone(t, pid)
			requireGone(t, cmd.guard.command.Process.Pid)
		})
	}
}

func TestCommandStartFailure(t *testing.T) {
	for _, mode := range []string{"missing", "canceled"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			binary := "/bin/sh"
			if mode == "missing" {
				binary = filepath.Join(t.TempDir(), "missing")
			} else {
				cancel()
			}
			cmd := CommandContext(ctx, binary)
			require.Error(t, cmd.Start())
			requireGone(t, cmd.guard.command.Process.Pid)
		})
	}
}

func TestCommandParentDeath(t *testing.T) {
	if pidFile := os.Getenv("LOOM_TESTPROCESS_PID"); pidFile != "" {
		cmd := CommandContext(context.Background(), "/bin/sh", "-c", "echo $$ > \"$1\"; sleep 60 & wait", "test", pidFile)
		require.NoError(t, cmd.Start())
		require.NoError(t, cmd.Wait())
		return
	}
	pidFile := filepath.Join(t.TempDir(), "pid")
	parent := exec.Command(os.Args[0], "-test.run=^TestCommandParentDeath$")
	parent.Env = append(os.Environ(), "LOOM_TESTPROCESS_PID="+pidFile)
	require.NoError(t, parent.Start())
	pid := readPID(t, pidFile)
	require.NoError(t, parent.Process.Kill())
	require.Error(t, parent.Wait())
	requireGone(t, pid)
}

func readPID(t *testing.T, path string) int {
	t.Helper()
	var pid int
	require.Eventually(t, func() bool {
		data, err := os.ReadFile(path)
		if err != nil {
			return false
		}
		pid, err = strconv.Atoi(strings.TrimSpace(string(data)))
		return err == nil
	}, 5*time.Second, 10*time.Millisecond)
	t.Cleanup(func() {
		if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
			t.Errorf("kill leftover process: %v", err)
		}
	})
	return pid
}

func requireGone(t *testing.T, pid int) {
	t.Helper()
	require.Eventually(t, func() bool {
		return errors.Is(syscall.Kill(pid, 0), syscall.ESRCH)
	}, 5*time.Second, 10*time.Millisecond, "process %d outlived its owner", pid)
}
