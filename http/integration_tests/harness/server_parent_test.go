//go:build unix

package harness

import (
	"context"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestServerParentTermination(t *testing.T) {
	if workDir := os.Getenv("LOOM_SERVER_PARENT_WORKDIR"); workDir != "" {
		server, err := StartServer(context.Background(), workDir, 0)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(os.Getenv("LOOM_SERVER_PARENT_URL"), []byte(server.URL()), 0o600))
		<-time.After(time.Minute)
		require.NoError(t, server.Stop())
		return
	}
	workDir := writeFakeServer(t, false)
	urlFile := filepath.Join(t.TempDir(), "url")
	parent := exec.Command(os.Args[0], "-test.run=^TestServerParentTermination$", "-test.timeout=45s")
	parent.Env = append(os.Environ(), "LOOM_SERVER_PARENT_WORKDIR="+workDir, "LOOM_SERVER_PARENT_URL="+urlFile)
	require.NoError(t, parent.Start())
	t.Cleanup(func() {
		if parent.ProcessState == nil {
			require.NoError(t, parent.Process.Kill())
			require.Error(t, parent.Wait())
		}
	})
	var address string
	require.Eventually(t, func() bool {
		raw, err := os.ReadFile(urlFile)
		if err != nil {
			return false
		}
		parsed, err := url.Parse(string(raw))
		if err != nil {
			return false
		}
		address = parsed.Host
		return address != ""
	}, 30*time.Second, 20*time.Millisecond)
	require.NoError(t, parent.Process.Kill())
	require.Error(t, parent.Wait())
	require.Eventually(t, func() bool {
		conn, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
		if err != nil {
			return true
		}
		require.NoError(t, conn.Close())
		return false
	}, 5*time.Second, 20*time.Millisecond, "server still accepts connections after parent termination")
}
