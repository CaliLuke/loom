package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPBodyProjectionMetadataOwnership(t *testing.T) {
	declaration := &UserTypeExpr{TypeName: "Parent", AttributeExpr: &AttributeExpr{
		Type: &Object{{Name: "kind", Attribute: &AttributeExpr{Type: String}}},
		Meta: MetaExpr{
			"openapi:discriminator:propertyName": {"kind"},
			"openapi:discriminator:mapping":      {"cat=Cat"},
			"http:encoding:sourceType":           {"object"},
			"openapi:encoding:kind:style":        {"form"},
			"unrelated":                          {"declaration"},
		},
	}}
	for _, local := range []bool{false, true} {
		attribute := &AttributeExpr{Type: declaration, Meta: MetaExpr{"local": {"retained"}}}
		if local {
			attribute.Meta["openapi:discriminator:propertyName"] = []string{"category"}
		}
		for _, project := range []func(*AttributeExpr) *MappedAttributeExpr{
			func(attribute *AttributeExpr) *MappedAttributeExpr {
				return objectHTTPBody(&HTTPEndpointExpr{wireBindings: &httpWireBindingPlan{}}, attribute)
			},
			func(attribute *AttributeExpr) *MappedAttributeExpr {
				return responseBodyObject(attribute, &HTTPResponseExpr{})
			},
		} {
			body := project(attribute)
			want := "kind"
			if local {
				want = "category"
			}
			require.Equal(t, []string{want}, body.Meta["openapi:discriminator:propertyName"])
			require.Equal(t, []string{"cat=Cat"}, body.Meta["openapi:discriminator:mapping"])
			require.Equal(t, []string{"object"}, body.Meta["http:encoding:sourceType"])
			require.Equal(t, []string{"retained"}, body.Meta["local"])
			require.NotContains(t, body.Meta, "unrelated")
			require.NotContains(t, body.Meta, "openapi:encoding:kind:style")
			body.Meta["openapi:discriminator:propertyName"][0] = "changed"
			require.Equal(t, []string{"kind"}, declaration.Meta["openapi:discriminator:propertyName"])
			if local {
				require.Equal(t, []string{"category"}, attribute.Meta["openapi:discriminator:propertyName"])
			}
		}
	}
}
