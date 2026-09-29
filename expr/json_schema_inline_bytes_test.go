package expr

import (
	"encoding/json/v2"
	"fmt"
	"regexp"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TestInlineJSONSchemaByteLengths(t *testing.T) {
	for _, tc := range []struct {
		name      string
		attribute *AttributeExpr
		accepted  []any
		rejected  []any
	}{
		{"maximum", &AttributeExpr{Type: Bytes, Validation: &ValidationExpr{MaxLength: intPtr(2)}}, []any{"", "aA==", "aGk=", "aGl="}, []any{"aGlq", "aGk=\n", "aGk=\r", "aGk=\u2028", "aGk=\u2029"}},
		{"minimum", &AttributeExpr{Type: Bytes, Validation: &ValidationExpr{MinLength: intPtr(2)}}, []any{"aGk=", "aGlq"}, []any{"", "aA=="}},
		{"unbounded", &AttributeExpr{Type: Bytes}, []any{"", "aGl="}, []any{"bad", "aGk", "aGk===", "____", "aGk= "}},
		{"impossible", &AttributeExpr{Type: Bytes, Validation: &ValidationExpr{MaxLength: intPtr(-1)}}, nil, []any{"", "aA=="}},
		{"nullable impossible", &AttributeExpr{Type: Bytes, Nullable: true, Validation: &ValidationExpr{MaxLength: intPtr(-1)}}, []any{nil}, []any{"", "aA=="}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := InlineJSONSchema(tc.attribute)
			require.NoError(t, err)
			var schema map[string]any
			require.NoError(t, json.Unmarshal(data, &schema))
			for _, value := range tc.accepted {
				require.True(t, inlineByteSchemaMatches(t, schema, value), "accepted value %q under %s", value, data)
			}
			for _, value := range tc.rejected {
				require.False(t, inlineByteSchemaMatches(t, schema, value), "rejected value %q under %s", value, data)
			}
		})
	}
}

// This evaluator covers the standard validation vocabulary emitted by this
// focused test; the generated contract gate independently checks full schemas.
func inlineByteSchemaMatches(t *testing.T, schema map[string]any, value any) bool {
	t.Helper()
	if typ, ok := schema["type"].(string); ok {
		if typ == "null" && value != nil || typ == "string" && value == nil {
			return false
		}
	}
	if inner, ok := schema["not"].(map[string]any); ok && inlineByteSchemaMatches(t, inner, value) {
		return false
	}
	for _, key := range []string{"allOf", "anyOf"} {
		if members, ok := schema[key].([]any); ok {
			matched := 0
			for _, member := range members {
				if inlineByteSchemaMatches(t, member.(map[string]any), value) {
					matched++
				}
			}
			if key == "allOf" && matched != len(members) || key == "anyOf" && matched == 0 {
				return false
			}
		}
	}
	if text, ok := value.(string); ok {
		for _, bound := range []string{"minLength", "maxLength"} {
			if length, exists := schema[bound].(float64); exists {
				actual := float64(utf8.RuneCountInString(text))
				if bound == "minLength" && actual < length || bound == "maxLength" && actual > length {
					return false
				}
			}
		}
		if pattern, exists := schema["pattern"].(string); exists {
			compiled, err := regexp.Compile(pattern)
			require.NoError(t, err, fmt.Sprint(schema))
			return compiled.MatchString(text)
		}
	}
	return true
}

func TestInlineJSONSchemaByteAliasConjunction(t *testing.T) {
	for _, nullable := range []bool{false, true} {
		for _, maximum := range []int{1, 2, 4} {
			t.Run(fmt.Sprintf("nullable=%t/max=%d", nullable, maximum), func(t *testing.T) {
				inner := &UserTypeExpr{TypeName: "Blob", AttributeExpr: &AttributeExpr{
					Type: Bytes, Nullable: nullable, Validation: &ValidationExpr{MinLength: intPtr(2)},
				}}
				alias := &UserTypeExpr{TypeName: "BlobAlias", AttributeExpr: &AttributeExpr{Type: inner}}
				attribute := &AttributeExpr{Type: alias, Validation: &ValidationExpr{MaxLength: intPtr(maximum)}}
				schema := mustInlineJSONSchema(t, attribute)
				require.Equal(t, nullable, inlineByteSchemaMatches(t, schema, nil))
				for length, wire := range []string{"", "aA==", "aGk=", "aGlq", "aGlqaw=="} {
					require.Equal(t, length >= 2 && length <= maximum, inlineByteSchemaMatches(t, schema, wire), "length=%d", length)
				}
			})
		}
	}
}

func TestInlineJSONSchemaByteComposition(t *testing.T) {
	bytes := &AttributeExpr{Type: Bytes, Validation: &ValidationExpr{MinLength: intPtr(2), MaxLength: intPtr(2), Values: []any{[]byte("hi")}}, DefaultValue: []byte("hi"), UserExamples: []*ExampleExpr{{Value: []byte("hi")}}}
	array := &AttributeExpr{Type: &Array{ElemType: bytes}, Validation: &ValidationExpr{MinLength: intPtr(1), MaxLength: intPtr(3)}}
	schema := mustInlineJSONSchema(t, array)
	require.Equal(t, float64(1), schema["minItems"])
	require.Equal(t, float64(3), schema["maxItems"])
	item := schema["items"].(map[string]any)
	require.Equal(t, "base64", item["contentEncoding"])
	require.NotContains(t, item, "minLength")
	require.NotContains(t, item, "maxLength")
	require.Equal(t, []any{"aGk="}, item["enum"])
	require.Equal(t, "aGk=", item["default"])
	require.Equal(t, []any{"aGk="}, item["examples"])
	maximum := int(^uint(0) >> 1)
	_, err := InlineJSONSchema(&AttributeExpr{Type: &Object{{Name: "blob", Attribute: &AttributeExpr{Type: Bytes, Validation: &ValidationExpr{MaxLength: &maximum}}}}})
	require.ErrorContains(t, err, `object field "blob"`)
	require.ErrorContains(t, err, "exact schema integer limit")
}

func TestInlineJSONSchemaByteEncodingOverrides(t *testing.T) {
	for _, metadata := range []MetaExpr{
		{"struct:field:type": {"other.Blob", "example.com/other"}},
		{"openapi:contentEncoding": {"base16"}},
		{"openapi:contentMediaType": {"application/custom"}},
		{"openapi:format": {""}},
	} {
		t.Run(fmt.Sprint(metadata), func(t *testing.T) {
			shared := &UserTypeExpr{TypeName: "Blob", AttributeExpr: &AttributeExpr{Type: Bytes}}
			attribute := &AttributeExpr{Type: &Object{
				{Name: "custom", Attribute: &AttributeExpr{Type: shared, Meta: metadata, Validation: &ValidationExpr{MaxLength: intPtr(2)}}},
				{Name: "ordinary", Attribute: &AttributeExpr{Type: shared}},
			}}
			schema := mustInlineJSONSchema(t, attribute)
			fields := schema["properties"].(map[string]any)
			custom := fields["custom"].(map[string]any)
			require.Equal(t, float64(2), custom["maxLength"], "preserve existing excluded schema")
			require.NotContains(t, custom, "contentEncoding")
			require.True(t, inlineByteSchemaMatches(t, custom, "hi"))
			ordinary := fields["ordinary"].(map[string]any)
			require.Equal(t, "base64", ordinary["contentEncoding"])
			require.False(t, inlineByteSchemaMatches(t, ordinary, "hi"))
		})
	}
}

func TestInlineJSONSchemaByteInheritedEncodingOverride(t *testing.T) {
	for _, metadata := range []MetaExpr{
		{"struct:field:type": {"other.Blob", "example.com/other"}},
		{"openapi:contentEncoding": {"base16"}},
		{"openapi:format": {""}},
	} {
		t.Run(fmt.Sprint(metadata), func(t *testing.T) {
			inner := &UserTypeExpr{TypeName: "CustomBlob", AttributeExpr: &AttributeExpr{Type: Bytes, Meta: metadata}}
			outer := &UserTypeExpr{TypeName: "Alias", AttributeExpr: &AttributeExpr{Type: inner}}
			schema := mustInlineJSONSchema(t, &AttributeExpr{Type: outer, Validation: &ValidationExpr{MaxLength: intPtr(2)}})
			require.NotContains(t, schema, "contentEncoding")
			require.True(t, inlineByteSchemaMatches(t, schema, "hi"))
			require.False(t, inlineByteSchemaMatches(t, schema, "long"))
		})
	}
}

func TestInlineJSONSchemaByteEffectiveBoundsBeforeProjection(t *testing.T) {
	huge := int(^uint(0) >> 1)
	for _, test := range []struct {
		name         string
		inner, outer ValidationExpr
		acceptsTwo   bool
	}{
		{"narrow huge maximum", ValidationExpr{MaxLength: &huge}, ValidationExpr{MaxLength: intPtr(2)}, true},
		{"empty huge minimum", ValidationExpr{MinLength: &huge}, ValidationExpr{MaxLength: intPtr(2)}, false},
		{"negative inner minimum", ValidationExpr{MinLength: intPtr(-5)}, ValidationExpr{MinLength: intPtr(2), MaxLength: intPtr(2)}, true},
		{"negative inner maximum", ValidationExpr{MaxLength: intPtr(-1)}, ValidationExpr{MaxLength: intPtr(2)}, false},
		{"contradictory inner", ValidationExpr{MinLength: intPtr(3), MaxLength: intPtr(1)}, ValidationExpr{MaxLength: intPtr(2)}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			inner := &UserTypeExpr{TypeName: "Blob", AttributeExpr: &AttributeExpr{Type: Bytes, Validation: &test.inner}}
			schema := mustInlineJSONSchema(t, &AttributeExpr{Type: inner, Validation: &test.outer})
			require.Equal(t, test.acceptsTwo, inlineByteSchemaMatches(t, schema, "aGk="))
			require.False(t, inlineByteSchemaMatches(t, schema, "YWJj"))
		})
	}
}
