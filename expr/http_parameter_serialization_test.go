package expr

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/httpencoding"
)

func TestHTTPValueShape(t *testing.T) {
	plain := &AttributeExpr{Type: String}
	integer := &AttributeExpr{Type: Int}
	typed := &AttributeExpr{Type: &Object{{Name: "field", Attribute: integer}}, AdditionalAttributes: plain}
	unreserved := httpencoding.ValueDomain{Unreserved: true}
	for _, tc := range []struct {
		name      string
		attribute *AttributeExpr
		want      httpencoding.BranchShape
	}{
		{"typed object", typed, httpencoding.BranchShape{Shape: httpencoding.UnsupportedShape}},
		{"typed object alias", &AttributeExpr{Type: &UserTypeExpr{TypeName: "Typed", AttributeExpr: typed}}, httpencoding.BranchShape{Shape: httpencoding.UnsupportedShape}},
		{"string", plain, httpencoding.BranchShape{Shape: httpencoding.Scalar}},
		{"bytes", &AttributeExpr{Type: Bytes}, httpencoding.BranchShape{Shape: httpencoding.Scalar}},
		{"integer", integer, httpencoding.BranchShape{Shape: httpencoding.Scalar, Domain: unreserved}},
		{"boolean", &AttributeExpr{Type: Boolean}, httpencoding.BranchShape{Shape: httpencoding.Scalar, Domain: unreserved}},
		{"alias", &AttributeExpr{Type: &UserTypeExpr{TypeName: "Alias", AttributeExpr: plain}}, httpencoding.BranchShape{Shape: httpencoding.Scalar}},
		{"scalar array", &AttributeExpr{Type: &Array{ElemType: integer}}, httpencoding.BranchShape{Shape: httpencoding.Sequence, Domain: unreserved}},
		{"object array", &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: &Object{}}}}, httpencoding.BranchShape{Shape: httpencoding.UnsupportedShape}},
		{"object mapped names", &AttributeExpr{Type: &Object{{Name: "field:wire", Attribute: integer}}, Validation: &ValidationExpr{Required: []string{"field"}}}, httpencoding.BranchShape{Shape: httpencoding.Object, Properties: []httpencoding.PropertyFact{{WireName: "wire", Required: true, Kind: httpencoding.Integer, Domain: unreserved}}}},
		{"string integer map", &AttributeExpr{Type: &Map{KeyType: plain, ElemType: integer}}, httpencoding.BranchShape{Shape: httpencoding.ScalarMap, Domain: unreserved}},
		{"integer string map", &AttributeExpr{Type: &Map{KeyType: integer, ElemType: plain}}, httpencoding.BranchShape{Shape: httpencoding.ScalarMap, Keys: httpencoding.KeyDomain{Unreserved: true}}},
		{"nested map", &AttributeExpr{Type: &Map{KeyType: plain, ElemType: &AttributeExpr{Type: &Array{ElemType: plain}}}}, httpencoding.BranchShape{Shape: httpencoding.Nested}},
		{"uuid", &AttributeExpr{Type: String, Validation: &ValidationExpr{Format: FormatUUID}}, httpencoding.BranchShape{Shape: httpencoding.Scalar, Domain: unreserved}},
		{"date", &AttributeExpr{Type: String, Validation: &ValidationExpr{Format: FormatDate}}, httpencoding.BranchShape{Shape: httpencoding.Scalar, Domain: unreserved}},
		{"plain enum", &AttributeExpr{Type: String, Validation: &ValidationExpr{Values: []any{"a-b", "x_y"}}}, httpencoding.BranchShape{Shape: httpencoding.Scalar, Domain: unreserved}},
		{"reserved enum", &AttributeExpr{Type: String, Validation: &ValidationExpr{Values: []any{"a/b"}}}, httpencoding.BranchShape{Shape: httpencoding.Scalar}},
		{"tagged union", &AttributeExpr{Type: &Union{Values: []*NamedAttributeExpr{{Name: "s", Attribute: plain}}}}, httpencoding.BranchShape{Shape: httpencoding.Nested}},
		{"untagged union", &AttributeExpr{Type: &Union{Untagged: true, Values: []*NamedAttributeExpr{{Name: "s", Attribute: plain}, {Name: "i", Attribute: integer}}}}, httpencoding.BranchShape{Shape: httpencoding.Union, Branches: []httpencoding.BranchShape{{Shape: httpencoding.Scalar}, {Shape: httpencoding.Scalar, Domain: unreserved}}}},
		{"nullable", &AttributeExpr{Type: String, Nullable: true}, httpencoding.BranchShape{Shape: httpencoding.UnsupportedShape}},
		{"any", &AttributeExpr{Type: Any}, httpencoding.BranchShape{Shape: httpencoding.UnsupportedShape}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, HTTPValueShape(tc.attribute))
		})
	}
}

func TestHTTPParameterSerialization(t *testing.T) {
	for _, prefix := range []string{"openapi:", "openapi:encoding:field:"} {
		t.Run(prefix, func(t *testing.T) {
			base := &UserTypeExpr{TypeName: "Base", AttributeExpr: &AttributeExpr{Type: String, Meta: MetaExpr{prefix + "style": {"form"}, prefix + "explode": {"true"}}}}
			attribute := &AttributeExpr{Type: base, Meta: MetaExpr{prefix + "style": {"deepObject"}, prefix + "explode": {"false"}, prefix + "allowReserved": {"false"}, prefix + "contentType": {"application/json"}}}
			no := false
			require.Equal(t, httpencoding.Serialization{Style: "deepObject", Explode: &no, AllowReserved: &no, MediaType: "application/json"}, HTTPParameterSerialization(attribute, prefix))
			require.Equal(t, httpencoding.Serialization{}, HTTPParameterSerialization(&AttributeExpr{Type: String}, prefix))
		})
	}
}
