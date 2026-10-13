package cli

import (
	json "encoding/json/v2"

	loom "github.com/CaliLuke/loom/pkg"
)

// UnmarshalJSON decodes a generated CLI flag using JSON v2 semantics with
// support for boolean map keys named "true" and "false", including nested
// maps and named boolean types. Custom JSON and text decoders take precedence.
func UnmarshalJSON(data []byte, value any) error {
	return json.Unmarshal(data, value, loom.JSONOptions())
}

// UnmarshalNonNullableJSON decodes one non-nullable CLI JSON body while retaining
// JSONOptions and custom decoder behavior. Optional owns the root null contract.
func UnmarshalNonNullableJSON[T any](data []byte, value *T) error {
	if value == nil {
		return json.Unmarshal(data, value, loom.JSONOptions())
	}
	var decoded loom.Optional[T]
	if err := json.Unmarshal(data, &decoded, loom.JSONOptions()); err != nil {
		return err
	}
	actual, _ := decoded.Value()
	*value = actual
	return nil
}
