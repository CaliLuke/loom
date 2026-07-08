package framework

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/loom/jsonrpc/integration_tests/harness"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// executor handles test scenario execution
type executor struct {
	serverURL string
	config    executorConfig
}

// newExecutor creates a new test executor
func newExecutor(serverURL string, opts ...executorOption) *executor {
	config := executorConfig{
		WebSocketTimeout: 30 * time.Second,
		Debug:            false,
	}

	for _, opt := range opts {
		opt(&config)
	}

	return &executor{
		serverURL: serverURL,
		config:    config,
	}
}

// Execute runs a test scenario
func (e *executor) Execute(t *testing.T, scenario Scenario) {
	t.Helper()
	e.debugf(t, "execute scenario=%q transport=%s method=%s", scenario.Name, scenario.Transport, scenario.Request.GetMethod(scenario.Method))

	// Handle different scenario types
	switch {
	case len(scenario.Sequence) > 0:
		e.executeStreaming(t, scenario)
	case len(scenario.Batch) > 0:
		e.executeBatch(t, scenario)
	case scenario.RawRequest != "":
		e.executeRaw(t, scenario)
	default:
		e.executeSimple(t, scenario)
	}
}

// executeSimple handles basic request/response scenarios
func (e *executor) executeSimple(t *testing.T, scenario Scenario) {
	t.Helper()

	ctx := context.Background()

	// Create client based on transport
	switch scenario.Transport {
	case TransportHTTP:
		e.executeHTTP(ctx, t, scenario)
	case TransportWebSocket:
		e.executeWebSocket(ctx, t, scenario)
	case TransportSSE:
		e.executeSSE(ctx, t, scenario)
	default:
		require.Failf(t, "Unknown transport", "Unknown transport: %s", scenario.Transport)
	}
}

// executeHTTP handles HTTP transport scenarios
func (e *executor) executeHTTP(ctx context.Context, t *testing.T, scenario Scenario) {
	t.Helper()

	// Create client
	client, err := harness.NewClient(e.serverURL, nil)
	require.NoError(t, err, "Failed to create client")

	// Build request
	method := scenario.Request.GetMethod(scenario.Method)
	e.debugf(t, "http request method=%s", method)

	// Try CLI client first for non-streaming scenarios
	// Skip CLI if custom JSONRPC field is specified
	if e.config.WorkDir != "" && scenario.Request.JSONRPC == "" {
		cliClient, err := harness.NewCLIClient(e.config.WorkDir, e.serverURL)
		if err != nil {
		} else if cliClient.CanHandle(method, scenario.Request.Params) {
			// For CLI, we need to separate service and method
			// Default to "test" service if no dot in method name
			service := "test"
			methodName := method
			if parts := strings.Split(method, "."); len(parts) == 2 {
				service = parts[0]
				methodName = parts[1]
			}

			result, err := cliClient.CallMethod(ctx, service, methodName, scenario.Request.Params)
			if err != nil {
				if scenario.Expect.Error != nil {
					// Some invalid-method cases fail in CLI argument parsing before a
					// transport-level JSON-RPC request is sent. Fall back to HTTP
					// unless the CLI error already contains a JSON-RPC error object.
					if _, ok := extractExpectedErrorObject(err); ok {
						e.validateError(t, err, scenario.Expect.Error)
						return
					}
				} else {
					require.NoError(t, err, "CLI call failed")
				}
			} else {
				// With verbose flag, CLI now returns the raw transport-level response
				if result != nil {
					// Wrap in JSON-RPC envelope
					response := map[string]any{
						"jsonrpc": "2.0",
						"id":      scenario.Request.ID,
						"result":  result,
					}
					e.validateJSONRPCResponse(t, response, scenario.Expect)
				} else if !scenario.Expect.NoResponse {
					assert.Fail(t, "Expected response but got none")
				}
				return
			}
		}
	}

	// Fall back to direct client

	req := harness.JSONRPCRequest{
		Method: method,
		Params: scenario.Request.Params,
		ID:     scenario.Request.ID,
	}
	// Handle JSONRPC field:
	// - Not specified (empty string) → Use default "2.0"
	// - "-" → Omit the field entirely
	// - Any other value → Use that value
	if scenario.Request.JSONRPC == "-" {
		// Special value to omit the field
		emptyStr := ""
		req.JSONRPC = &emptyStr
	} else if scenario.Request.JSONRPC != "" {
		// Custom value specified
		req.JSONRPC = &scenario.Request.JSONRPC
	}
	// If JSONRPC is empty string (not specified), req.JSONRPC remains nil and defaults to "2.0"
	result, err := client.CallHTTP(ctx, req)
	if err != nil {
		if scenario.Expect.Error != nil {
			e.validateError(t, err, scenario.Expect.Error)
			return
		}
		require.NoError(t, err, "HTTP call failed")
	}

	// Handle notification case
	if scenario.Expect.NoResponse {
		assert.Nil(t, result, "Expected no response for notification")
		return
	}

	// Parse response
	if result != nil {
		e.debugf(t, "http response bytes=%d", len(result))
		var resp any
		err := json.Unmarshal(result, &resp)
		require.NoError(t, err, "Failed to parse response")
		e.validateJSONRPCResponse(t, resp, scenario.Expect)
	} else if !scenario.Expect.NoResponse {
		assert.Fail(t, "Expected response but got none")
	}
}

// executeWebSocket handles WebSocket transport scenarios
func (e *executor) executeWebSocket(ctx context.Context, t *testing.T, scenario Scenario) {
	t.Helper()

	// WebSocket scenarios always use sequence
	if len(scenario.Sequence) > 0 {
		e.executeWebSocketSequence(ctx, t, scenario)
		return
	}

	// If no sequence, create a simple send/receive sequence from request/expect
	if scenario.Request.Params != nil {
		// Pass method, params, and id as separate fields
		data := map[string]any{
			"method": scenario.Method,
			"params": scenario.Request.Params,
		}
		if scenario.Request.ID != nil {
			data["id"] = scenario.Request.ID
		}

		scenario.Sequence = []Action{
			{Type: "send", Data: data},
			{Type: "receive", Expect: scenario.Expect},
		}
		e.executeWebSocketSequence(ctx, t, scenario)
	}
}

// executeSSE handles Server-Sent Events scenarios
func (e *executor) executeSSE(ctx context.Context, t *testing.T, scenario Scenario) {
	t.Helper()

	client, err := harness.NewClient(e.serverURL, nil)
	require.NoError(t, err, "Failed to create client")

	method := scenario.Request.GetMethod(scenario.Method)
	e.debugf(t, "sse request method=%s", method)
	req := harness.JSONRPCRequest{
		Method: method,
		Params: scenario.Request.Params,
		ID:     scenario.Request.ID,
	}
	if scenario.Request.JSONRPC == "-" {
		emptyStr := ""
		req.JSONRPC = &emptyStr
	} else if scenario.Request.JSONRPC != "" {
		req.JSONRPC = &scenario.Request.JSONRPC
	}

	events, err := client.CallSSE(ctx, req)
	require.NoError(t, err, "SSE request failed")
	e.debugf(t, "sse event count=%d", len(events))

	if scenario.Expect.NoResponse {
		require.Len(t, events, 0, "Expected no SSE events for notification")
		return
	}

	require.Len(t, events, 1, "Expected exactly one SSE event")
	var response map[string]any
	err = json.Unmarshal(events[0].Data, &response)
	require.NoError(t, err, "Failed to unmarshal SSE response")

	require.Equal(t, "message", events[0].Type, "Unexpected SSE event type")
	e.validateJSONRPCResponse(t, response, scenario.Expect)
}

// executeStreaming handles streaming scenarios with sequences
func (e *executor) executeStreaming(t *testing.T, scenario Scenario) {
	t.Helper()

	ctx := context.Background()

	// Only WebSocket and SSE support streaming
	switch scenario.Transport {
	case TransportWebSocket:
		e.executeWebSocketSequence(ctx, t, scenario)
	case TransportSSE:
		e.executeSSESequence(ctx, t, scenario)
	default:
		require.Failf(t, "Unsupported transport", "Transport %s does not support streaming", scenario.Transport)
	}
}

// executeWebSocketSequence handles WebSocket streaming sequences
func (e *executor) executeWebSocketSequence(ctx context.Context, t *testing.T, scenario Scenario) {
	t.Helper()

	client, err := harness.NewClient(e.serverURL, nil)
	require.NoError(t, err, "Failed to create client")

	// Execute sequence steps
	for i, step := range scenario.Sequence {
		e.debugf(t, "websocket step=%d type=%s", i, step.Type)
		switch step.Type {
		case "connect":
			err := client.ConnectWebSocket(ctx)
			require.NoErrorf(t, err, "Step %d: failed to connect WebSocket", i)

		case "send":
			// Auto-connect if not connected
			if !client.IsConnected() {
				err := client.ConnectWebSocket(ctx)
				require.NoErrorf(t, err, "Step %d: failed to auto-connect WebSocket", i)
			}

			require.NotNilf(t, step.Data, "Step %d: send step requires data", i)

			// Extract method, params, and id from the data
			reqData, ok := step.Data.(map[string]any)
			require.Truef(t, ok, "Step %d: invalid request data format", i)

			req := harness.JSONRPCRequest{
				Method: reqData["method"].(string),
				Params: reqData["params"],
				ID:     reqData["id"],
			}

			// Mark HasID when id key present (even if it's null)
			if _, hasID := reqData["id"]; hasID {
				req.HasID = true
			}

			// Handle custom jsonrpc field if specified
			if jsonrpcVal, ok := reqData["jsonrpc"]; ok {
				if jsonrpcStr, ok := jsonrpcVal.(string); ok {
					if jsonrpcStr == "-" {
						// Special value to omit the field
						emptyStr := ""
						req.JSONRPC = &emptyStr
					} else {
						req.JSONRPC = &jsonrpcStr
					}
				}
			}
			// If not specified, JSONRPC remains nil and defaults to "2.0"

			err := client.SendWebSocket(ctx, req)
			require.NoErrorf(t, err, "Step %d: failed to send", i)

		case "receive":
			msg, err := client.ReceiveWebSocket(ctx)
			require.NoErrorf(t, err, "Step %d: failed to receive", i)

			var response map[string]any
			err = json.Unmarshal(msg, &response)
			require.NoErrorf(t, err, "Step %d: failed to unmarshal response", i)

			// Compare the response with expected
			if expected, ok := step.Expect.(map[string]any); ok {
				e.compareJSONRPCMessages(t, response, expected)
			} else {
				require.Failf(t, "Invalid expected value", "Step %d: expected value must be a map", i)
			}

		case "close":
			err := client.CloseWebSocket()
			require.NoErrorf(t, err, "Step %d: failed to close WebSocket", i)

		default:
			require.Failf(t, "Unknown step type", "Step %d: unknown step type: %s", i, step.Type)
		}

		// Apply delay if specified
		if step.Delay > 0 {
			time.Sleep(step.Delay)
		}
	}
}

// executeSSESequence handles SSE streaming sequences
func (e *executor) executeSSESequence(ctx context.Context, t *testing.T, scenario Scenario) {
	t.Helper()

	// SSE only supports server-to-client streaming
	require.True(t, scenario.Request.Params != nil || scenario.Request.ID != nil, "SSE requires an initial request")

	client, err := harness.NewClient(e.serverURL, nil)
	require.NoError(t, err, "Failed to create client")

	// Send request and get SSE events
	method := scenario.Request.GetMethod(scenario.Method)
	e.debugf(t, "sse sequence request method=%s steps=%d", method, len(scenario.Sequence))
	req := harness.JSONRPCRequest{
		Method: method,
		Params: scenario.Request.Params,
		ID:     scenario.Request.ID,
	}
	// Handle JSONRPC field:
	// - Not specified (empty string) → Use default "2.0"
	// - "-" → Omit the field entirely
	// - Any other value → Use that value
	if scenario.Request.JSONRPC == "-" {
		// Special value to omit the field
		emptyStr := ""
		req.JSONRPC = &emptyStr
	} else if scenario.Request.JSONRPC != "" {
		// Custom value specified
		req.JSONRPC = &scenario.Request.JSONRPC
	}
	// If JSONRPC is empty string (not specified), req.JSONRPC remains nil and defaults to "2.0"
	events, err := client.CallSSE(ctx, req)
	require.NoError(t, err, "SSE request failed")
	e.debugf(t, "sse sequence event count=%d", len(events))

	// Validate sequence
	require.Len(t, events, len(scenario.Sequence), "Event count mismatch")

	for i, step := range scenario.Sequence {
		require.Equalf(t, "receive", step.Type, "SSE only supports 'receive' steps, got %s", step.Type)

		require.Lessf(t, i, len(events), "Expected event at step %d, but no more events", i)

		// For SSE streaming, step.Expect contains the full expected JSON-RPC message.
		expectedMsg, ok := step.Expect.(map[string]any)
		require.True(t, ok, "Step %d: invalid expect format", i)

		e.validateSSEEvent(t, events[i], expectedMsg, i)
	}
}

func (e *executor) validateSSEEvent(t *testing.T, event harness.SSEEvent, expectedMsg map[string]any, step int) {
	t.Helper()

	require.Equalf(t, expectedSSEEventType(expectedMsg), event.Type, "Step %d: unexpected SSE event type", step)

	var response map[string]any
	err := json.Unmarshal(event.Data, &response)
	require.NoErrorf(t, err, "Failed to unmarshal event %d", step)
	e.compareJSONRPCMessages(t, response, expectedMsg)
}

func expectedSSEEventType(msg map[string]any) string {
	if _, ok := msg["error"]; ok {
		return "message"
	}
	if _, ok := msg["result"]; ok {
		return "message"
	}
	return "message"
}
