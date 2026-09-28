//go:build unix

package harness

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

func TestCLIParentTimeoutProof(t *testing.T) {
	if binary := os.Getenv("LOOM_PARENT_TIMEOUT_CLI"); binary != "" {
		client := &CLIClient{binPath: binary, cliPath: filepath.Dir(binary), serverURL: "http://127.0.0.1:1"}
		_, err := client.CallMethod(context.Background(), "test", "hang", nil)
		require.NoError(t, err)
		return
	}
	client, err := NewCLIClient(writeFakeCLI(t, "unused"), "http://127.0.0.1:1")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, client.Close())
	})
	pidFile := filepath.Join(t.TempDir(), "cli.pid")
	command := exec.Command(os.Args[0], "-test.run=^TestCLIParentTimeoutProof$", "-test.timeout=3s")
	command.Env = append(os.Environ(), "LOOM_PARENT_TIMEOUT_CLI="+client.binPath, "FAKE_CLI_PID_FILE="+pidFile)
	output, err := command.CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(output), "test timed out")
	raw, err := os.ReadFile(pidFile)
	require.NoError(t, err, "the CLI must have started before its parent's timeout")
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
			t.Errorf("kill leftover CLI: %v", err)
		}
	})
	require.Eventually(t, func() bool {
		return errors.Is(syscall.Kill(pid, 0), syscall.ESRCH)
	}, 2*time.Second, 20*time.Millisecond, "CLI outlived its test process")
}
