package loom

import (
	"encoding/json/jsontext"
	"encoding/json/v2"

	"github.com/CaliLuke/loom/internal/jsonkey"
)

// JSONOptions returns the JSON options used by generated unions, presence
// wrappers, and HTTP body codecs. Boolean map keys use the canonical member
// names "true" and "false", including named boolean types and nested maps.
// Ordinary boolean values retain JSON boolean syntax. Custom JSON and text
// codecs retain control of their representation. Callers that need stable
// object member ordering must also pass json.Deterministic(true).
func JSONOptions() json.Options {
	return jsonkey.BooleanKeys
}

// unmarshalPresenceJSON retains caller codecs ahead of the boolean-key fallback.
// Other semantic and syntactic options remain on the enclosing decoder.
func unmarshalPresenceJSON(dec *jsontext.Decoder, value any) error {
	caller, _ := json.GetOption(dec.Options(), json.WithUnmarshalers)
	fallback, _ := json.GetOption(JSONOptions(), json.WithUnmarshalers)
	return json.UnmarshalDecode(dec, value, json.WithUnmarshalers(json.JoinUnmarshalers(caller, fallback)))
}
