package valuecontract

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/schematest"
)

// TestByteSchemaIndependentConstraints covers effective evaluated alias graphs
// that cannot all be authored directly by the public DSL. Actual generated DSL
// requests have a separate HTTP/JSON-RPC gate. Ajv supplies the independent
// schema semantics, including native ECMA-262 end-anchor behavior.
func TestByteSchemaIndependentConstraints(t *testing.T) {
	if os.Getenv("LOOM_OPENAPI_CONTRACT") != "1" {
		t.Skip("independent schema engine runs in make openapi-contract")
	}
	ptr := func(value int) *int {
		return &value
	}
	maxInt := int(^uint(0) >> 1)
	batches := make([]schematest.Batch, 0, 5)
	expectations := make([][]bool, 0, 5)
	for _, tc := range []struct {
		name     string
		inner    byteAliasBound
		outer    byteAliasBound
		nullable bool
		accept   func(int) bool
	}{
		{"redundant_overflow", byteAliasBound{Maximum: &maxInt}, byteAliasBound{Maximum: ptr(2)}, false, func(n int) bool {
			return n <= 2
		}},
		{"empty_before_overflow", byteAliasBound{Minimum: &maxInt}, byteAliasBound{Maximum: ptr(2)}, false, func(int) bool {
			return false
		}},
		{"negative_lower", byteAliasBound{Minimum: ptr(-3)}, byteAliasBound{Maximum: ptr(2)}, false, func(n int) bool {
			return n <= 2
		}},
		{"negative_upper", byteAliasBound{Maximum: ptr(-1)}, byteAliasBound{}, false, func(int) bool {
			return false
		}},
		{"nullable", byteAliasBound{Minimum: ptr(2)}, byteAliasBound{Maximum: ptr(2)}, true, func(n int) bool {
			return n == 2
		}},
	} {
		inner := &expr.UserTypeExpr{TypeName: tc.name, AttributeExpr: &expr.AttributeExpr{
			Type: expr.Bytes, Nullable: tc.nullable,
			Validation: &expr.ValidationExpr{MinLength: tc.inner.Minimum, MaxLength: tc.inner.Maximum},
		}}
		attribute := &expr.AttributeExpr{Type: inner, Validation: &expr.ValidationExpr{MinLength: tc.outer.Minimum, MaxLength: tc.outer.Maximum}}
		data, err := expr.InlineJSONSchema(attribute)
		require.NoError(t, err, tc.name)
		instances := []any{"", "aA==", "aGk=", "aGV5", "bG9uZw==", "aGl=", "aGm=", "aGn=", "aGk=\n", "aGk=\r", "aGk=\u2028", "aGk=\u2029", nil}
		lengths := []int{0, 1, 2, 3, 4, 2, 2, 2}
		expected := make([]bool, len(instances))
		for index, length := range lengths {
			expected[index] = tc.accept(length)
		}
		expected[len(expected)-1] = tc.nullable
		batches = append(batches, schematest.Batch{Schema: jsontext.Value(data), Instances: instances})
		expectations = append(expectations, expected)
	}
	validator := schematest.New(t)
	results := validator.Check(t, batches)
	for index, expected := range expectations {
		for sample, valid := range expected {
			if results[index][sample].Valid != valid {
				t.Errorf("constraint case %d sample %d: expected=%t got=%t: %s", index, sample, valid, results[index][sample].Valid, results[index][sample].Errors)
			}
		}
	}

	// A schema-unique String interpretation does not prove runtime uniqueness:
	// the same noncanonical text also decodes to the allowed Bytes enum value.
	enumSchema, err := expr.InlineJSONSchema(&expr.AttributeExpr{Type: expr.Bytes, Validation: &expr.ValidationExpr{Values: []any{[]byte("hi")}}})
	require.NoError(t, err)
	control := validator.Check(t, []schematest.Batch{
		{Schema: jsontext.Value(enumSchema), Instances: []any{"aGk=", "aGl="}},
		{Schema: map[string]any{"type": "string", "enum": []string{"aGl="}}, Instances: []any{"aGl="}},
	})
	require.True(t, control[0][0].Valid)
	require.False(t, control[0][1].Valid)
	require.True(t, control[1][0].Valid)
	var decoded []byte
	require.NoError(t, json.Unmarshal([]byte(`"aGl="`), &decoded))
	require.Equal(t, []byte("hi"), decoded)
}
