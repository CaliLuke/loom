package representation

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestFormTagProjectionIsOccurrenceLocal(t *testing.T) {
	inner := &expr.UserTypeExpr{TypeName: "Inner", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{
		{Name: "member", Attribute: &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:tag:form": {"innerForm"}, "struct:tag:json": {"innerJSON"}}}},
	}}}
	field := &expr.AttributeExpr{Type: inner, Meta: expr.MetaExpr{"struct:tag:form": {"outerForm"}, "struct:tag:json": {"outerJSON"}}}
	source := &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Outer", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: field}}}}}
	for _, codec := range []expr.ValueCodec{expr.ValueCodecForm, expr.ValueCodecMultipart} {
		form, json := TargetGraph(source), TargetGraph(source)
		normalizeParameterTarget(form, codec, nil)
		normalizeParameterTarget(json, expr.ValueCodecJSON, nil)
		formField := expr.AsObject(form.Type).Attribute("value")
		jsonField := expr.AsObject(json.Type).Attribute("value")
		formName, _, err := FieldWireNames(formField, "value")
		require.NoError(t, err)
		require.Equal(t, "outerForm", formName)
		jsonName, _, err := FieldWireNames(jsonField, "value")
		require.NoError(t, err)
		require.Equal(t, "outerJSON", jsonName)
		innerName, _, err := FieldWireNames(expr.AsObject(formField.Type).Attribute("member"), "member")
		require.NoError(t, err)
		require.Equal(t, "innerJSON", innerName, "JSON-valued field members keep JSON names")
		require.Equal(t, []string{"outerJSON"}, field.Meta["struct:tag:json"])
		require.NotContains(t, field.Meta, "struct:tag:json:name")
	}
}
