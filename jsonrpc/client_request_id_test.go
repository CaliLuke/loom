package jsonrpc

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"net/http/httptest"
	"testing"

	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

type countedID struct {
	calls int
}

func TestClientRequestIDPresence(t *testing.T) {
	for _, tc := range []struct {
		name         string
		value        any
		present      bool
		wire         string
		notification bool
	}{
		{"absent", nil, false, "", true},
		{"empty", "", true, "", true},
		{"value", "a", true, `"a"`, false},
		{"nullable-absent", loom.Nullable[string]{}, false, "", true},
		{"nullable-null", loom.NullValue[string](), true, "null", false},
		{"nullable-value", loom.NullableValue("b"), true, `"b"`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", nil)
			message := &Request{JSONRPC: "2.0", Method: "call"}
			require.NoError(t, SetClientRequestID(req, message, tc.value, tc.present))
			require.Equal(t, tc.notification, IsClientNotification(req))
			encoded, err := json.Marshal(message)
			require.NoError(t, err)
			var fields map[string]jsontext.Value
			require.NoError(t, json.Unmarshal(encoded, &fields))
			require.Equal(t, tc.wire, string(fields["id"]))
		})
	}
}

func TestClientRequestIDCodecOnce(t *testing.T) {
	value := &countedID{}
	req := httptest.NewRequest("POST", "/", nil)
	message := &Request{JSONRPC: "2.0", Method: "call"}
	require.NoError(t, SetClientRequestID(req, message, value, true))
	require.False(t, IsClientNotification(req))
	_, err := json.Marshal(message)
	require.NoError(t, err)
	require.False(t, IsClientNotification(req))
	require.Equal(t, 1, value.calls)
}

func TestRequestOrdinaryIDOmission(t *testing.T) {
	empty := ""
	for _, value := range []any{nil, "", &empty} {
		encoded, err := json.Marshal(Request{JSONRPC: "2.0", Method: "call", ID: value})
		require.NoError(t, err)
		require.JSONEq(t, `{"jsonrpc":"2.0","method":"call"}`, string(encoded))
	}
	req := httptest.NewRequest("POST", "/", nil)
	require.False(t, IsClientNotification(req))
	require.Error(t, SetClientRequestID(req, &Request{}, 42, true), "custom String storage must encode a String or null")
}

func (c *countedID) MarshalJSON() ([]byte, error) {
	c.calls++
	return []byte(`"once"`), nil
}
