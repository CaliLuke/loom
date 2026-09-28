package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestProtoGoFieldsAvoidMethodsAndGetters(t *testing.T) {
	for _, tc := range []struct {
		name   string
		fields []string
		want   []string
	}{
		{"reserved methods", []string{"reset", "descriptor", "string", "proto_message", "marshal", "unmarshal", "extension_range_array", "extension_map"}, []string{"Reset_", "Descriptor_", "String_", "ProtoMessage_", "Marshal_", "Unmarshal_", "ExtensionRangeArray_", "ExtensionMap_"}},
		{"getter after field", []string{"label", "get_label"}, []string{"Label", "GetLabel_"}},
		{"getter before field", []string{"get_label", "label"}, []string{"GetLabel", "Label_"}},
		{"getter chain", []string{"label", "get_label", "get_get_label"}, []string{"Label", "GetLabel_", "GetGetLabel"}},
		{"initialism", []string{"OAuth2Token"}, []string{"OAuth2Token"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			obj := make(expr.Object, len(tc.fields))
			for i, field := range tc.fields {
				obj[i] = &expr.NamedAttributeExpr{Name: field, Attribute: &expr.AttributeExpr{Type: expr.String}}
			}
			names := newProtoMessageNames(&expr.AttributeExpr{Type: &obj})
			for i, field := range tc.fields {
				require.Equal(t, tc.want[i], names.goField(field), field)
			}
		})
	}
}
