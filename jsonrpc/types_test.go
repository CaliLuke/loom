package jsonrpc

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResponseMarshalResultMember(t *testing.T) {
	var nilPointer *struct{}
	tests := []struct {
		name     string
		response *Response
		want     string
	}{
		{"nil result", MakeSuccessResponse(1, nil), `{"jsonrpc":"2.0","result":null,"id":1}`},
		{"empty string", MakeSuccessResponse("a", ""), `{"jsonrpc":"2.0","result":"","id":"a"}`},
		{"empty array", MakeSuccessResponse(1, []string{}), `{"jsonrpc":"2.0","result":[],"id":1}`},
		{"nil array", MakeSuccessResponse(1, []string(nil)), `{"jsonrpc":"2.0","result":[],"id":1}`},
		{"empty map", MakeSuccessResponse(1, map[string]int{}), `{"jsonrpc":"2.0","result":{},"id":1}`},
		{"empty struct", MakeSuccessResponse(1, struct{}{}), `{"jsonrpc":"2.0","result":{},"id":1}`},
		{"nil pointer", MakeSuccessResponse(1, nilPointer), `{"jsonrpc":"2.0","result":null,"id":1}`},
		{"zero number", MakeSuccessResponse(1, 0), `{"jsonrpc":"2.0","result":0,"id":1}`},
		{"false", MakeSuccessResponse(1, false), `{"jsonrpc":"2.0","result":false,"id":1}`},
		{"raw null", MakeSuccessResponse(1, jsontext.Value("null")), `{"jsonrpc":"2.0","result":null,"id":1}`},
		{"raw empty object", MakeSuccessResponse(1, jsontext.Value("{}")), `{"jsonrpc":"2.0","result":{},"id":1}`},
		{"empty raw value", MakeSuccessResponse(1, jsontext.Value(nil)), `{"jsonrpc":"2.0","result":null,"id":1}`},
		{"value", MakeSuccessResponse(jsontext.Value("7"), map[string]int{"n": 1}), `{"jsonrpc":"2.0","result":{"n":1},"id":7}`},
		{"null id", MakeSuccessResponse(nil, nil), `{"jsonrpc":"2.0","result":null,"id":null}`},
		{"literal without result", &Response{JSONRPC: "2.0", ID: 1}, `{"jsonrpc":"2.0","result":null,"id":1}`},
		{"error", MakeErrorResponse(1, InternalError, "", nil), `{"jsonrpc":"2.0","error":{"code":-32603,"message":"Internal error"},"id":1}`},
		{"error with null id", MakeErrorResponse(nil, ParseError, "", nil), `{"jsonrpc":"2.0","error":{"code":-32700,"message":"Parse error"},"id":null}`},
		{"error with data", MakeErrorResponse("e", InvalidParams, "bad", map[string]string{"field": "x"}), `{"jsonrpc":"2.0","error":{"code":-32602,"message":"bad","data":{"field":"x"}},"id":"e"}`},
		{"error with result", &Response{JSONRPC: "2.0", Result: "ignored", Error: &ErrorResponse{Code: InternalError, Message: "boom"}, ID: 1}, `{"jsonrpc":"2.0","error":{"code":-32603,"message":"boom"},"id":1}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.response)
			require.NoError(t, err)
			require.Equal(t, tt.want, string(got))
			got, err = json.Marshal(*tt.response)
			require.NoError(t, err)
			require.Equal(t, tt.want, string(got), "value receiver")
		})
	}
}

func TestResponseMarshalBatch(t *testing.T) {
	batch := []*Response{
		MakeSuccessResponse(1, nil),
		MakeErrorResponse(2, MethodNotFound, "", nil),
		MakeSuccessResponse(3, ""),
	}
	got, err := json.Marshal(batch)
	require.NoError(t, err)
	require.Equal(t, `[{"jsonrpc":"2.0","result":null,"id":1},{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"},"id":2},{"jsonrpc":"2.0","result":"","id":3}]`, string(got))
}

// TestResponseMarshalUsesCallerOptions checks that the result is encoded with
// the options of the caller, such as deterministic map ordering and
// indentation.
func TestResponseMarshalUsesCallerOptions(t *testing.T) {
	result := make(map[string]int)
	for _, k := range []string{"j", "b", "h", "a", "f", "c", "i", "e", "d", "g"} {
		result[k] = len(k)
	}
	want := `{"jsonrpc":"2.0","result":{"a":1,"b":1,"c":1,"d":1,"e":1,"f":1,"g":1,"h":1,"i":1,"j":1},"id":1}`
	for range 20 {
		got, err := json.Marshal(MakeSuccessResponse(1, result), json.Deterministic(true))
		require.NoError(t, err)
		require.Equal(t, want, string(got))
	}
	var buf bytes.Buffer
	require.NoError(t, json.MarshalWrite(&buf, MakeSuccessResponse(1, nil), jsontext.WithIndent("  ")))
	require.Equal(t, "{\n  \"jsonrpc\": \"2.0\",\n  \"result\": null,\n  \"id\": 1\n}", buf.String())
}

// TestRequestMarshalParamsMember checks that a request omits its "params"
// member only when it has no params, and keeps empty strings, arrays and
// objects and null values.
func TestRequestMarshalParamsMember(t *testing.T) {
	var nilPointer *struct{}
	tests := []struct {
		name    string
		request *Request
		want    string
	}{
		{"no params", &Request{JSONRPC: "2.0", Method: "m", ID: 1}, `{"jsonrpc":"2.0","method":"m","id":1}`},
		{"nil pointer", &Request{JSONRPC: "2.0", Method: "m", Params: nilPointer, ID: 1}, `{"jsonrpc":"2.0","method":"m","params":null,"id":1}`},
		{"raw null", &Request{JSONRPC: "2.0", Method: "m", Params: jsontext.Value("null"), ID: 1}, `{"jsonrpc":"2.0","method":"m","params":null,"id":1}`},
		{"empty string", &Request{JSONRPC: "2.0", Method: "m", Params: "", ID: 1}, `{"jsonrpc":"2.0","method":"m","params":"","id":1}`},
		{"empty array", &Request{JSONRPC: "2.0", Method: "m", Params: []string{}, ID: 1}, `{"jsonrpc":"2.0","method":"m","params":[],"id":1}`},
		{"empty map", &Request{JSONRPC: "2.0", Method: "m", Params: map[string]int{}, ID: 1}, `{"jsonrpc":"2.0","method":"m","params":{},"id":1}`},
		{"empty struct", &Request{JSONRPC: "2.0", Method: "m", Params: struct{}{}, ID: 1}, `{"jsonrpc":"2.0","method":"m","params":{},"id":1}`},
		{"raw empty object", &Request{JSONRPC: "2.0", Method: "m", Params: jsontext.Value("{}"), ID: 1}, `{"jsonrpc":"2.0","method":"m","params":{},"id":1}`},
		{"zero number", &Request{JSONRPC: "2.0", Method: "m", Params: 0, ID: 1}, `{"jsonrpc":"2.0","method":"m","params":0,"id":1}`},
		{"value", &Request{JSONRPC: "2.0", Method: "m", Params: map[string]int{"n": 1}, ID: "a"}, `{"jsonrpc":"2.0","method":"m","params":{"n":1},"id":"a"}`},
		{"notification", MakeNotification("m", map[string]int{"n": 1}), `{"jsonrpc":"2.0","method":"m","params":{"n":1}}`},
		{"notification with empty object", MakeNotification("m", struct{}{}), `{"jsonrpc":"2.0","method":"m","params":{}}`},
		{"notification without params", MakeNotification("m", nil), `{"jsonrpc":"2.0","method":"m"}`},
		{"empty id", &Request{JSONRPC: "2.0", Method: "m", Params: 1, ID: ""}, `{"jsonrpc":"2.0","method":"m","params":1}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.request)
			require.NoError(t, err)
			require.Equal(t, tt.want, string(got))
			got, err = json.Marshal(*tt.request)
			require.NoError(t, err)
			require.Equal(t, tt.want, string(got), "value receiver")
		})
	}
}

// TestRequestMarshalUsesCallerOptions checks that the params are encoded
// with the options of the caller, such as deterministic map ordering and
// indentation.
func TestRequestMarshalUsesCallerOptions(t *testing.T) {
	params := make(map[string]int)
	for _, k := range []string{"j", "b", "h", "a", "f", "c", "i", "e", "d", "g"} {
		params[k] = len(k)
	}
	want := `{"jsonrpc":"2.0","method":"m","params":{"a":1,"b":1,"c":1,"d":1,"e":1,"f":1,"g":1,"h":1,"i":1,"j":1},"id":1}`
	for range 20 {
		got, err := json.Marshal(&Request{JSONRPC: "2.0", Method: "m", Params: params, ID: 1}, json.Deterministic(true))
		require.NoError(t, err)
		require.Equal(t, want, string(got))
	}
	var buf bytes.Buffer
	require.NoError(t, json.MarshalWrite(&buf, &Request{JSONRPC: "2.0", Method: "m", Params: []int{1}, ID: 1}, jsontext.WithIndent("  ")))
	require.Equal(t, "{\n  \"jsonrpc\": \"2.0\",\n  \"method\": \"m\",\n  \"params\": [\n    1\n  ],\n  \"id\": 1\n}", buf.String())
}

// TestResponseMarshalRoundTrip checks that a client decoding a success
// response without a result sees the result member as null.
func TestResponseMarshalRoundTrip(t *testing.T) {
	data, err := json.Marshal(MakeSuccessResponse(1, nil))
	require.NoError(t, err)
	var members map[string]jsontext.Value
	require.NoError(t, json.Unmarshal(data, &members))
	require.Equal(t, "null", string(members["result"]))
	var raw RawResponse
	require.NoError(t, json.Unmarshal(data, &raw))
	require.Nil(t, raw.Error)
	require.Equal(t, "null", string(raw.Result))
}
