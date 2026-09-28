package cli

import (
	json "encoding/json/v2"

	"github.com/CaliLuke/loom/internal/jsonkey"
)

// UnmarshalJSON decodes a generated CLI flag using JSON v2 semantics with
// support for boolean map keys named "true" and "false", including nested
// maps and named boolean types. Custom JSON and text decoders take precedence.
func UnmarshalJSON(data []byte, value any) error {
	return json.Unmarshal(data, value, jsonkey.BooleanKeys)
}
