package tests

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"goa.design/goa/v3/jsonrpc/integration_tests/framework"
	"goa.design/goa/v3/jsonrpc/integration_tests/harness"
)

func TestSSEHeadersAreCommittedBeforeFirstEvent(t *testing.T) {
	t.Parallel()

	methodInfo, err := framework.ParseMethod("stream_object_sse")
	require.NoError(t, err)

	workDir := t.TempDir()
	generator := framework.NewGenerator(workDir, map[string]framework.MethodInfo{
		methodInfo.Name(): methodInfo,
	})
	require.NoError(t, generator.Generate())

	require.NoError(t, delayFirstSSEEvent(filepath.Join(workDir, "testsse.go"), 250*time.Millisecond))

	serverCtx, cancelServer := context.WithCancel(context.Background())
	defer cancelServer()

	server, err := harness.StartServer(serverCtx, workDir, 0)
	require.NoError(t, err)
	defer server.Stop() //nolint:errcheck

	client, err := harness.NewClient(server.URL(), nil)
	require.NoError(t, err)

	openCtx, cancelOpen := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancelOpen()

	resp, err := client.OpenSSE(openCtx, harness.JSONRPCRequest{
		Method: methodInfo.Name(),
		Params: map[string]any{
			"field1": "headers",
			"field2": 1,
			"field3": true,
		},
		ID: "req-1",
	})
	require.NoError(t, err)
	defer resp.Body.Close() //nolint:errcheck

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))
}

func delayFirstSSEEvent(path string, delay time.Duration) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	const needle = "if err := stream.Send(ctx, result); err != nil {"
	replacement := fmt.Sprintf("\ttime.Sleep(%d * time.Millisecond)\n\tif err := stream.Send(ctx, result); err != nil {", delay/time.Millisecond)

	updated := strings.Replace(string(content), needle, replacement, 1)
	if updated == string(content) {
		return os.ErrNotExist
	}

	return os.WriteFile(path, []byte(updated), 0644)
}
