package openapiv3_test

import (
	"testing"

	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/stretchr/testify/require"
)

func TestAdditionalBytesProjection(t *testing.T) {
	for _, version := range []struct{ target, want string }{{"3.1", openapiv3.OpenAPICompatibilityVersion}, {"3.2", openapiv3.OpenAPIVersion}} {
		t.Run(version.target, func(t *testing.T) {
			spec := renderTypedSpec(t, testdata.AdditionalCompositeDSL, version.target, version.want)
			schemas := typedComponents(spec)
			record, ok := schemas["ByteRecord"].(map[string]any)
			require.True(t, ok)
			require.NotNil(t, record["additionalProperties"])
			requireTypedWire(t, spec, "ByteRecord", []typedWireCase{{`{"id":"record","x":"aGk="}`, true}, {`{"x":"aA=="}`, false}, {`{"x":"hi"}`, false}, {`{}`, false}})
			requireTypedWire(t, spec, "ByteNested", []typedWireCase{{`{"x":[{"child":"aGk="}]}`, true}, {`{"x":[{"child":"aA=="}]}`, false}})
			requireTypedWire(t, spec, "ByteNullable", []typedWireCase{{`{"x":null}`, true}, {`{"x":"aGk="}`, true}, {`{"x":"aA=="}`, false}, {`{}`, false}})
			// OpenAPI 3.1/3.2 retain occurrence predicates beside $ref. The
			// draft-07 wire validator above ignores reference siblings, so check
			// these authored assertions directly; prepared-plan tests exercise
			// their admission semantics.
			occurrences := schemas["ByteOccurrences"].(map[string]any)["properties"].(map[string]any)
			for _, field := range []struct{ name, required string }{{"base", ""}, {"record", "more"}, {"sibling", "other"}} {
				property := occurrences[field.name].(map[string]any)
				require.Equal(t, "#/components/schemas/ByteRecord", property["$ref"])
				if field.required == "" {
					require.NotContains(t, property, "required")
				} else {
					require.Equal(t, []any{field.required}, property["required"])
				}
			}
			requireTypedExamplesValid(t, spec)
		})
	}
}
