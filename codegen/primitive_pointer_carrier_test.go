package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

// TestPrimitivePointerCarrierAgreement checks the query against native declarations.
func TestPrimitivePointerCarrierAgreement(t *testing.T) {
	alias := func(name string, dataType expr.DataType) expr.DataType {
		return &expr.UserTypeExpr{TypeName: name, AttributeExpr: &expr.AttributeExpr{Type: dataType}}
	}
	for _, tc := range []struct {
		name     string
		dataType expr.DataType
		nullable bool
	}{
		{"scalar", expr.String, false},
		{"named scalar", alias("Text", expr.String), false},
		{"array", &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}, false},
		{"named array", alias("Tags", &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}), false},
		{"map", &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.String}}, false},
		{"object", &expr.Object{}, false},
		{"union", &expr.Union{TypeName: "Choice"}, false},
		{"bytes", expr.Bytes, false},
		{"named bytes", alias("Blob", expr.Bytes), false},
		{"any", expr.Any, false},
		{"named any", alias("Value", expr.Any), false},
		{"nullable scalar", expr.String, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			field := &expr.AttributeExpr{Type: tc.dataType, Nullable: tc.nullable}
			parent := &expr.AttributeExpr{Type: &expr.Object{&expr.NamedAttributeExpr{Name: "value", Attribute: field}}}
			context := &AttributeContext{Pointer: true}
			layout := expr.ServiceFieldLayout(parent, "value", field, true, false)
			want := expr.IsPrimitive(field.Type) && layout.Pointer
			require.Equal(t, want, context.IsPrimitivePointer("value", parent), "primitive query must agree with declaration")
		})
	}
}

// TestPrimitivePointerMissingCarrier checks that a context flag cannot create a field.
func TestPrimitivePointerMissingCarrier(t *testing.T) {
	context := &AttributeContext{Pointer: true}
	parent := &expr.AttributeExpr{Type: &expr.Object{}}
	require.False(t, context.IsPrimitivePointer("missing", parent))
}

// TestPrimitivePointerMappedRequiredness keeps semantic field identity intact.
func TestPrimitivePointerMappedRequiredness(t *testing.T) {
	for _, required := range []bool{false, true} {
		field := &expr.AttributeExpr{Type: expr.String}
		parent := &expr.AttributeExpr{Type: &expr.Object{{Name: "value:wire", Attribute: field}}}
		if required {
			parent.Validation = &expr.ValidationExpr{Required: []string{"value:wire"}}
		}
		context := &AttributeContext{}
		for _, name := range []string{"value", "value:wire"} {
			require.Equal(t, !required, context.IsPrimitivePointer(name, parent), name)
		}
	}
}
