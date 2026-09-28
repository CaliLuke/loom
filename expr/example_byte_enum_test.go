package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestByteExampleEnumCoercions(t *testing.T) {
	for _, datatype := range []DataType{Bytes, &UserTypeExpr{TypeName: "Blob", AttributeExpr: &AttributeExpr{Type: Bytes}}} {
		for _, allowed := range []any{"hi", []byte("hi")} {
			attribute := &AttributeExpr{Type: datatype, Validation: &ValidationExpr{Values: []any{allowed}}}
			for _, value := range []any{"hi", []byte("hi")} {
				require.True(t, exampleMatchesAttribute(attribute, value))
			}
			require.False(t, exampleMatchesAttribute(attribute, "different"))
			require.Equal(t, []any{allowed}, attribute.Validation.Values)
		}
	}
	for _, allowed := range []any{"", []byte{}, []byte(nil)} {
		attribute := &AttributeExpr{Type: Bytes, Validation: &ValidationExpr{Values: []any{allowed}}}
		for _, value := range []any{"", []byte{}, []byte(nil)} {
			require.True(t, exampleMatchesAttribute(attribute, value))
		}
	}
	for _, datatype := range []DataType{String, Any} {
		attribute := &AttributeExpr{Type: datatype, Validation: &ValidationExpr{Values: []any{"hi"}}}
		require.False(t, exampleMatchesAttribute(attribute, []byte("hi")))
	}
}
