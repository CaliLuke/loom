package tests

import (
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/integration_tests/harness"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestHTTPSSEFixtureRejectsBeforeStreamCommit(t *testing.T) {
	t.Parallel()

	serverCtx, cancelServer := context.WithCancel(context.Background())
	defer cancelServer()

	fixtureDir := filepath.Join("..", "fixtures", "ticktock")
	server, err := harness.StartServer(serverCtx, fixtureDir, 0)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, server.Stop())
	}()

	reqCtx, cancelReq := context.WithTimeout(context.Background(), time.Second)
	defer cancelReq()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, server.URL()+"/guarded", nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	require.False(t, strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream"))
	require.NotContains(t, string(body), "event:")
}

func TestHTTPSSEFixtureRegeneratesAndBuilds(t *testing.T) {
	t.Parallel()

	srcDir := filepath.Join("..", "fixtures", "ticktock")
	workDir := filepath.Join(t.TempDir(), "ticktock")
	require.NoError(t, testingx.CopyTree(srcDir, workDir))
	require.NoError(t, testingx.PinLocalReplace(workDir, testingx.RepoRoot()))

	_, err := testingx.RunCmd(workDir, "go", "run", "-mod=mod", "github.com/CaliLuke/loom/cmd/loom", "gen", "example.com/http-ticktock/design", "-o", ".")
	require.NoError(t, err)
	testingx.RequireTreeMatches(t, filepath.Join(srcDir, "gen"), filepath.Join(workDir, "gen"))

	_, err = testingx.RunCmd(workDir, "go", "test", "./...")
	require.NoError(t, err)
}
