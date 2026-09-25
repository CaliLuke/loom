//go:build unix

package harness

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fakeCLIMain is a CLI that answers like the generated ones: with -verbose it
// writes the JSON-RPC response to stderr. The hang method writes the process
// ID to the file named by FAKE_CLI_PID_FILE and then never returns, like a CLI
// waiting on a server that does not answer.
const fakeCLIMain = `package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

func main() {
	flag.String("url", "", "server URL")
	flag.Bool("verbose", false, "verbose output")
	flag.Parse()
	if flag.Arg(1) == "hang" {
		pid := []byte(strconv.Itoa(os.Getpid()))
		if err := os.WriteFile(os.Getenv("FAKE_CLI_PID_FILE"), pid, 0o600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		time.Sleep(time.Hour)
	}
	fmt.Fprintln(os.Stderr, ` + "`" + `{"jsonrpc":"2.0","id":1,"result":{"answer":"REPLY"}}` + "`" + `)
}
`

func TestCLIClientKillsCLIOnTimeout(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "cli.pid")
	t.Setenv("FAKE_CLI_PID_FILE", pidFile)
	client, err := NewCLIClient(writeFakeCLI(t, "first"), "http://127.0.0.1:1")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, client.Close())
	})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := client.CallMethod(ctx, "test", "hang", nil)
		done <- err
	}()

	pid := waitForPID(t, pidFile)
	t.Cleanup(func() {
		// Kill a CLI that the client left behind so the test does not leak it.
		if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
			t.Errorf("kill leftover CLI process %d: %v", pid, err)
		}
	})
	cancel()
	select {
	case err := <-done:
		require.Error(t, err, "CallMethod must fail when its context ends")
	case <-time.After(10 * time.Second):
		require.Fail(t, "CallMethod did not return after its context ended")
	}

	require.Eventually(t, func() bool {
		return errors.Is(syscall.Kill(pid, 0), syscall.ESRCH)
	}, 5*time.Second, 50*time.Millisecond, "the CLI process %d outlived its canceled call", pid)
}

func TestCLIClientRunsBinaryBuiltOnce(t *testing.T) {
	workDir := writeFakeCLI(t, "first")
	client, err := NewCLIClient(workDir, "http://127.0.0.1:1")
	require.NoError(t, err)

	// A call after the source changes still runs the binary built by
	// NewCLIClient, so calls do not rebuild the CLI.
	writeFakeCLISource(t, workDir, "second")
	for range 2 {
		result, err := client.CallMethod(t.Context(), "test", "echo", nil)
		require.NoError(t, err)
		require.JSONEq(t, `{"answer":"first"}`, string(result))
	}

	require.NoError(t, client.Close())
	_, err = os.Stat(client.binPath)
	require.ErrorIs(t, err, os.ErrNotExist, "Close must remove the CLI binary")
	require.NoError(t, client.Close(), "Close must be idempotent")
}

func TestNewCLIClientReportsMissingCLI(t *testing.T) {
	client, err := NewCLIClient(t.TempDir(), "http://127.0.0.1:1")
	if err == nil {
		require.NoError(t, client.Close())
	}
	require.ErrorIs(t, err, ErrCLINotFound)
}

// waitForPID waits until the fake CLI writes its process ID to path.
func waitForPID(t *testing.T, path string) int {
	t.Helper()
	var pid int
	require.Eventually(t, func() bool {
		data, err := os.ReadFile(path)
		if err != nil {
			return false
		}
		pid, err = strconv.Atoi(strings.TrimSpace(string(data)))
		return err == nil
	}, 60*time.Second, 50*time.Millisecond, "the fake CLI did not start")
	return pid
}

// writeFakeCLI writes a module with fakeCLIMain, answering reply, at the path
// NewCLIClient looks for and returns the module directory.
func writeFakeCLI(t *testing.T, reply string) string {
	t.Helper()
	workDir := t.TempDir()
	writeFakeCLISource(t, workDir, reply)
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "go.mod"), []byte("module fakecli\n\ngo 1.27\n"), 0o600))
	return workDir
}

// writeFakeCLISource writes fakeCLIMain, answering reply, into the module in
// workDir.
func writeFakeCLISource(t *testing.T, workDir, reply string) {
	t.Helper()
	mainDir := filepath.Join(workDir, "cmd", "test_api-cli")
	require.NoError(t, os.MkdirAll(mainDir, 0o750))
	source := strings.ReplaceAll(fakeCLIMain, "REPLY", reply)
	require.NoError(t, os.WriteFile(filepath.Join(mainDir, "main.go"), []byte(source), 0o600))
}
