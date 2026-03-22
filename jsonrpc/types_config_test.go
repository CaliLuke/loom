package jsonrpc

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewStreamConfigAppliesOptionsAndValidation(t *testing.T) {
	handlerCalled := false
	handler := func(ctx context.Context, errorType StreamErrorType, err error, response *RawResponse) {
		handlerCalled = true
		require.NotNil(t, ctx)
		require.Equal(t, StreamErrorTimeout, errorType)
		require.EqualError(t, err, "boom")
		require.NotNil(t, response)
	}

	cfg := NewStreamConfig(
		WithRequestTimeout(-1),
		WithConnectionTimeout(0),
		WithCloseTimeout(-2),
		WithResultChannelBuffer(0),
		WithWebSocketBuffers(32, 64),
		WithRetryConfig(-1, 0, 500*time.Millisecond),
		WithCompression(true),
		WithPingInterval(0),
		WithErrorHandler(handler),
	)

	require.Equal(t, 30*time.Second, cfg.RequestTimeout)
	require.Equal(t, 10*time.Second, cfg.ConnectionTimeout)
	require.Equal(t, 5*time.Second, cfg.CloseTimeout)
	require.Equal(t, 1, cfg.ResultChannelBuffer)
	require.Equal(t, 1024, cfg.ReadBufferSize)
	require.Equal(t, 1024, cfg.WriteBufferSize)
	require.Equal(t, 0, cfg.MaxRetries)
	require.Equal(t, time.Second, cfg.RetryBackoffBase)
	require.Equal(t, 30*time.Second, cfg.RetryBackoffMax)
	require.True(t, cfg.EnableCompression)
	require.Equal(t, 30*time.Second, cfg.PingInterval)
	require.NotNil(t, cfg.ErrorHandler)

	cfg.ErrorHandler(context.Background(), StreamErrorTimeout, errors.New("boom"), &RawResponse{})
	require.True(t, handlerCalled)
}

func TestJSONRPCResponseHelpersAndRawRequest(t *testing.T) {
	success := MakeSuccessResponse("req-1", map[string]any{"ok": true})
	require.Equal(t, "2.0", success.JSONRPC)
	require.Equal(t, "req-1", success.ID)
	require.NotNil(t, success.Result)

	errResp := MakeErrorResponse("req-2", MethodNotFound, "", map[string]string{"field": "id"})
	require.Equal(t, "2.0", errResp.JSONRPC)
	require.Equal(t, MethodNotFound, errResp.Error.Code)
	require.Equal(t, "Method not found", errResp.Error.Message)
	require.Equal(t, "jsonrpc: code -32601: Method not found", errResp.Error.Error())

	unknownErr := MakeErrorResponse(nil, Code(12345), "", nil)
	require.Equal(t, "Unknown error", unknownErr.Error.Message)

	notification := MakeNotification("widgets.show", map[string]any{"id": "1"})
	require.Equal(t, "2.0", notification.JSONRPC)
	require.Equal(t, "widgets.show", notification.Method)
	require.Nil(t, notification.ID)

	require.Equal(t, "abc", IDToString("abc"))
	require.Equal(t, "42", IDToString(float64(42)))
	require.Equal(t, "", IDToString(true))

	var req RawRequest
	require.NoError(t, json.Unmarshal([]byte(`{"jsonrpc":"2.0","method":"widgets.show","params":{"id":"1"},"id":null}`), &req))
	require.True(t, req.HasID)
	require.Nil(t, req.ID)
	require.JSONEq(t, `{"id":"1"}`, string(req.Params))

	require.NoError(t, json.Unmarshal([]byte(`{"jsonrpc":"2.0","method":"widgets.notify"}`), &req))
	require.False(t, req.HasID)
	require.Nil(t, req.ID)

	rawErr := &RawErrorResponse{Code: -32000, Message: "bad gateway"}
	require.Equal(t, "jsonrpc: code -32000: bad gateway", rawErr.Error())
}
