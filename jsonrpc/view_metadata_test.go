package jsonrpc

import (
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStreamViewEnvelope(t *testing.T) {
	for _, tc := range []struct {
		name    string
		message any
		want    string
	}{
		{"notification", &Request{JSONRPC: "2.0", Method: "event", Params: "body", View: "tiny"}, `{"jsonrpc":"2.0","method":"event","params":"body","loom_view":"tiny"}`},
		{"response", &Response{JSONRPC: "2.0", Result: "body", ID: 1, View: "tiny"}, `{"jsonrpc":"2.0","result":"body","id":1,"loom_view":"tiny"}`},
		{"error", &Response{JSONRPC: "2.0", Error: &ErrorResponse{Code: InternalError, Message: "failed"}, ID: 1, View: "ignored"}, `{"jsonrpc":"2.0","error":{"code":-32603,"message":"failed"},"id":1}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.message, json.Deterministic(true))
			require.NoError(t, err)
			require.Equal(t, tc.want, string(data))
		})
	}
}

func TestRawResponseViewMetadata(t *testing.T) {
	for _, tc := range []struct {
		name, wire, view string
		invalid          bool
	}{
		{"selected", `{"jsonrpc":"2.0","result":{},"id":1,"loom_view":"tiny"}`, "tiny", false},
		{"absent", `{"jsonrpc":"2.0","result":{},"id":1}`, "", false},
		{"empty", `{"jsonrpc":"2.0","result":{},"id":1,"loom_view":""}`, "", false},
		{"null", `{"jsonrpc":"2.0","result":{},"id":1,"loom_view":null}`, "", true},
		{"invalid", `{"jsonrpc":"2.0","result":{},"id":1,"loom_view":123}`, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, target := range []any{new(Request), new(Response)} {
				err := json.Unmarshal([]byte(tc.wire), target)
				if tc.invalid {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			}
			var response RawResponse
			err := json.Unmarshal([]byte(tc.wire), &response)
			if tc.invalid {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.view, string(response.View))
			require.Equal(t, "{}", string(response.Result))
		})
	}
}
