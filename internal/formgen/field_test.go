package formgen

import (
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/httpencoding"
	"github.com/stretchr/testify/require"
)

func TestCapturedFieldCalls(t *testing.T) {
	for _, tc := range []struct {
		name           string
		field          Field
		encode, decode string
	}{
		{"native items", Field{Codec: httpencoding.Codec{Style: httpencoding.Form, Application: httpencoding.Items}}, "loomhttp.EncodeFormValue(values, key, value)", "loomhttp.DecodeFormValue(values, key, &value)"},
		{"JSON items", Field{Codec: httpencoding.Codec{Style: httpencoding.JSON, Application: httpencoding.Items}}, "loomhttp.EncodeJSONField(values, key, value, loomhttp.FieldItems)", "loomhttp.DecodeJSONField(values, key, &value, false, loomhttp.FieldItems)"},
		{"nullable JSON value", Field{Codec: httpencoding.Codec{Style: httpencoding.JSON}, AllowsNull: true}, "loomhttp.EncodeJSONField(values, key, value, loomhttp.FieldValue)", "loomhttp.DecodeJSONField(values, key, &value, true, loomhttp.FieldValue)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.encode, tc.field.EncodeCall("values", "key", "value"))
			require.Equal(t, tc.decode, tc.field.DecodeCall("values", "key", "value"))
		})
	}
}

func TestCapturedFieldPresence(t *testing.T) {
	field := &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}}
	parent := &expr.AttributeExpr{Type: &expr.Object{{Name: "items", Attribute: field}}}
	context := codegen.NewAttributeContext(false, false, true, "", codegen.NewNameScope())
	require.Empty(t, Presence(context, parent, "items", field, "body.Items"))
	context.JSONPresence = true
	require.Equal(t, "body.Items.Present()", Presence(context, parent, "items", field, "body.Items"))
	captured := Field{Codec: httpencoding.Codec{Style: httpencoding.JSON, Application: httpencoding.Items}}
	out := captured.Encode(Access{Values: "values", Name: "key", Value: "body.Items", Presence: "body.Items.Present()", Required: true, Label: "items"})
	require.Contains(t, out, "if body.Items.Present()")
	require.Contains(t, out, "has no encoded items")
	require.Contains(t, out, "is absent")
	require.NotContains(t, out, "len(body.Items)")
}
