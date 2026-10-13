package codegen

import (
	"testing"

	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestJSONNativeStorageRetainsZero(t *testing.T) {
	for _, tc := range []struct {
		name     string
		typ      expr.DataType
		fallback any
	}{
		{"string", expr.String, "fallback"},
		{"bool", expr.Boolean, true},
		{"number", expr.Int, 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			field := &expr.AttributeExpr{Type: tc.typ, DefaultValue: tc.fallback}
			parent := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: field}}}
			require.Contains(t, NewNameScope().GoTypeDef(parent, false, true), `json:"value"`)
			require.Contains(t, NewNameScope().GoTypeDef(parent, false, false), `json:"value,omitzero"`)
		})
	}
}
