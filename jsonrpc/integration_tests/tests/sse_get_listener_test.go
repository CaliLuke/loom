package tests

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/testingx"
	"github.com/CaliLuke/loom/jsonrpc/integration_tests/harness"
)

// eventsStreamDesign declares the raw GET events/stream listener contract: a
// JSON-RPC service whose SSE method is literally named "events/stream" makes
// the generated server mount a GET route that opens the stream without a
// JSON-RPC request body.
const eventsStreamDesign = `package design

import . "github.com/CaliLuke/loom/dsl"

var _ = API("ticktock", func() {
	JSONRPC(func() {})
})

var _ = Service("clock", func() {
	JSONRPC(func() {
		POST("/rpc")
	})

	Method("events/stream", func() {
		Payload(func() {
			ID("id", String)
		})
		StreamingResult(func() {
			Attribute("value", String)
		})
		JSONRPC(func() {
			ServerSentEvents()
		})
	})
})
`

// eventsStreamImpl implements the regenerated clock service for the
// events/stream design. It replaces the fixture's Tick/Tock implementation.
const eventsStreamImpl = `package ticktock

import (
	"context"

	clock "example.com/ticktock/gen/clock"
)

type clocksrvc struct{}

func NewClock() clock.Service {
	return &clocksrvc{}
}

func (s *clocksrvc) EventsStream(ctx context.Context, p *clock.EventsStreamPayload, stream clock.EventsStreamServerStream) error {
	for _, value := range []string{"event-1", "event-2"} {
		if err := stream.Send(ctx, &clock.EventsStreamResult{Value: stringPtr(value)}); err != nil {
			return err
		}
	}
	return stream.SendAndClose(ctx, &clock.EventsStreamResult{Value: stringPtr("event-done")})
}

func stringPtr(v string) *string {
	return &v
}
`

// TestJSONRPCSSEGetEventsStreamListener proves the raw GET events/stream
// listener contract end to end — the branch the checked-in POST-initiated
// fixtures do not cover:
//
//   - A plain GET with Accept: text/event-stream (no JSON-RPC body) opens the
//     stream: 200, text/event-stream, and every Send value arrives framed as
//     a JSON-RPC 2.0 notification.
//   - The SendAndClose value is intentionally NOT delivered: the GET listener
//     carries no JSON-RPC request ID, and generated streams close ID-less
//     requests without a final response (jsonrpc/codegen/stream_sections.go,
//     "Notifications are closed without a final response"). The server just
//     closes the stream.
func TestJSONRPCSSEGetEventsStreamListener(t *testing.T) {
	t.Parallel()

	srcDir := filepath.Join("..", "fixtures", "ticktock")
	workDir := filepath.Join(t.TempDir(), "eventsstream")
	require.NoError(t, testingx.CopyTree(srcDir, workDir))
	require.NoError(t, testingx.PinLocalReplace(workDir, testingx.RepoRoot()))

	// Swap in the events/stream design and matching implementation; the
	// checked-in cmd wiring is method-name agnostic and is reused as is.
	// The fixture-root client tests target the Tick/Tock design, so drop
	// them from the copy.
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "design", "design.go"), []byte(eventsStreamDesign), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "clock.go"), []byte(eventsStreamImpl), 0o600))
	require.NoError(t, os.Remove(filepath.Join(workDir, "sse_client_close_test.go")))

	_, err := testingx.RunCmd(workDir, "go", "run", "-mod=mod", "github.com/CaliLuke/loom/cmd/loom", "gen", "example.com/ticktock/design", "-o", ".")
	require.NoError(t, err)

	server, err := harness.StartServer(t.Context(), workDir, 0)
	require.NoError(t, err)
	defer server.Stop() //nolint:errcheck

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL()+"/rpc", nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close() //nolint:errcheck

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, resp.Header.Get("Content-Type"), "text/event-stream")

	values := readSSEStreamValues(t, resp.Body)

	// The two Send values arrive; the SendAndClose value ("event-done") is
	// dropped by the ID-less close semantics and the stream ends cleanly.
	require.Equal(t, []string{"event-1", "event-2"}, values)
}

// readSSEStreamValues parses SSE frames off the response body until the
// server closes the stream, requiring every data frame to be a JSON-RPC 2.0
// envelope and returning the streamed result values in order.
func readSSEStreamValues(t *testing.T, body io.Reader) []string {
	t.Helper()

	var (
		values []string
		data   strings.Builder
	)
	flush := func() {
		if data.Len() == 0 {
			return
		}
		raw := data.String()
		data.Reset()
		var envelope map[string]any
		require.NoError(t, json.Unmarshal([]byte(raw), &envelope), "frame data %q", raw)
		require.Equal(t, "2.0", envelope["jsonrpc"], "frame data %q", raw)
		values = append(values, extractStreamValue(t, envelope))
	}

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case line == "":
			flush()
		case strings.HasPrefix(line, "data:"):
			data.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	require.NoError(t, scanner.Err(), "reading SSE stream (values so far: %v)", values)
	flush()
	return values
}

// extractStreamValue pulls the streamed result value out of a JSON-RPC
// envelope, accepting both notification (params) and response (result)
// framing so the assertions do not over-pin the frame shape.
func extractStreamValue(t *testing.T, envelope map[string]any) string {
	t.Helper()

	for _, key := range []string{"params", "result"} {
		payload, ok := envelope[key].(map[string]any)
		if !ok {
			continue
		}
		if value, ok := payload["value"].(string); ok {
			return value
		}
	}
	t.Fatalf("envelope carries no string value: %v", envelope)
	return ""
}
