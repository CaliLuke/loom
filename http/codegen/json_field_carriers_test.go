package codegen

import (
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/internal/httpencoding"
	"github.com/stretchr/testify/require"
)

func TestJSONFormCarrierRetainsConstraints(t *testing.T) {
	value := &expr.UserTypeExpr{TypeName: "Address", AttributeExpr: &expr.AttributeExpr{
		Type:       &expr.Object{{Name: "street", Attribute: &expr.AttributeExpr{Type: expr.String}}},
		Validation: &expr.ValidationExpr{Required: []string{"street"}},
	}}
	field := &expr.AttributeExpr{Type: value}
	request := &transportir.Request{FormEncoded: true, Body: &expr.AttributeExpr{Type: &expr.Object{{Name: "address", Attribute: field}}}, FormFieldModes: map[string]httpencoding.Mode{"address": httpencoding.JSON}}
	nameJSONFormCarriers([]*transportir.Endpoint{{Name: "submit", Request: request, Response: &transportir.Response{}}})
	require.NotSame(t, value, field.Type)
	require.Equal(t, []string{"street"}, selectedTransportConstraints(field).Validation().Lowered().Required)
	require.Equal(t, "Address", value.TypeName)
}

func TestJSONQueryCarrierCustomBoundary(t *testing.T) {
	scope := codegen.NewNameScope()
	builder := &payloadBuilder{sds: &ServicesData{}, sd: &ServiceData{Scope: scope}}
	attribute := &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.String}}, Meta: expr.MetaExpr{"struct:field:type": {"custom.Map", "example.com/custom"}}}
	parameter := &ParamData{JSONField: true, Element: &Element{AttributeData: &AttributeData{VarName: "field", TypeRef: "custom.Map"}}}
	builder.jsonQueryCarrier(parameter, attribute)
	require.Equal(t, "custom.Map", parameter.JSONDecodeType)
	require.Equal(t, "field = decoded", parameter.JSONDecodeCode, "opaque custom codecs must not be traversed by the builtin collection converter")
}
