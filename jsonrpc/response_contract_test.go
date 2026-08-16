package jsonrpc

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateResponseContract(t *testing.T) {
	tests := []struct {
		name     string
		contract ResponseContractCase
		response string
		status   int
		wantErr  string
	}{
		{
			name:     "success",
			contract: ResponseContractCase{ID: "tools.call.success", Kind: ResponseContractSuccess, HasResult: true},
			response: `{"jsonrpc":"2.0","id":1,"result":{"value":"ok"}}`,
			status:   http.StatusOK,
		},
		{
			name:     "error",
			contract: ResponseContractCase{ID: "tools.call.error.forbidden.4403", Kind: ResponseContractError, ErrorCode: 4403, ErrorName: "forbidden", ErrorDataType: "ToolError"},
			response: `{"jsonrpc":"2.0","id":1,"error":{"code":4403,"message":"forbidden","data":{"name":"forbidden"}}}`,
			status:   http.StatusOK,
		},
		{
			name:     "notification",
			contract: ResponseContractCase{ID: "tools.call.notification", Kind: ResponseContractNotification},
			status:   http.StatusOK,
		},
		{
			name:     "missing result",
			contract: ResponseContractCase{ID: "tools.call.success", Kind: ResponseContractSuccess, HasResult: true},
			response: `{"jsonrpc":"2.0","id":1}`,
			status:   http.StatusOK,
			wantErr:  "result member is missing",
		},
		{
			name:     "wrong code",
			contract: ResponseContractCase{ID: "tools.call.error.forbidden.4403", Kind: ResponseContractError, ErrorCode: 4403, ErrorDataType: "ToolError"},
			response: `{"jsonrpc":"2.0","id":1,"error":{"code":-32603,"message":"internal","data":{}}}`,
			status:   http.StatusOK,
			wantErr:  "error code is -32603, want 4403",
		},
		{
			name:     "notification body",
			contract: ResponseContractCase{ID: "tools.call.notification", Kind: ResponseContractNotification},
			response: `{"jsonrpc":"2.0","id":null,"result":null}`,
			status:   http.StatusOK,
			wantErr:  "notification produced a response body",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			observation := &ResponseContractObservation{Response: &http.Response{
				StatusCode: test.status,
				Body:       io.NopCloser(strings.NewReader(test.response)),
			}}
			err := ValidateResponseContract(observation, test.contract)
			if test.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestValidateResponseContractServerSSE(t *testing.T) {
	success := ResponseContractCase{
		ID: "events.watch.success", Kind: ResponseContractSuccess, HasResult: true,
		Stream: &StreamingResponseContract{Transport: "sse", Terminal: "final_response"},
	}
	require.NoError(t, ValidateResponseContract(&ResponseContractObservation{
		Response: &http.Response{StatusCode: http.StatusOK},
		Events: []ResponseContractEvent{
			{Type: "notification", Data: []byte(`{"jsonrpc":"2.0","method":"events/watch.event","params":"one"}`)},
			{Type: "message", Data: []byte(`{"jsonrpc":"2.0","id":1,"result":"done"}`)},
		},
		TerminalError: io.EOF,
	}, success))

	suppressed := ResponseContractCase{
		ID: "events.watch.notification", Kind: ResponseContractNotification,
		Stream: &StreamingResponseContract{Transport: "sse", Terminal: "suppressed"},
	}
	require.NoError(t, ValidateResponseContract(&ResponseContractObservation{
		Response:      &http.Response{StatusCode: http.StatusOK},
		Events:        []ResponseContractEvent{{Type: "notification", Data: []byte(`{"jsonrpc":"2.0","method":"events/watch.event","params":"one"}`)}},
		TerminalError: io.EOF,
	}, suppressed))
	require.ErrorContains(t, ValidateResponseContract(&ResponseContractObservation{
		Response:      &http.Response{StatusCode: http.StatusOK},
		Events:        []ResponseContractEvent{{Type: "message", Data: []byte(`{"jsonrpc":"2.0","id":1,"result":"done"}`)}},
		TerminalError: io.EOF,
	}, suppressed), "notification stream produced a terminal response")
	require.ErrorContains(t, ValidateResponseContract(&ResponseContractObservation{
		Response:      &http.Response{StatusCode: http.StatusOK},
		TerminalError: errors.New("reset"),
	}, success), "terminal error, want clean EOF")
}
