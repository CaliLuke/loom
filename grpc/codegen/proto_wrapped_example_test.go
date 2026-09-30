package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestWrappedTypeExampleCacheOrder(t *testing.T) {
	for _, tc := range []struct {
		name string
		typ  expr.DataType
	}{
		{"primitive", expr.String},
		{"union", &expr.Union{TypeName: "Choice", Values: []*expr.NamedAttributeExpr{{Name: "text", Attribute: &expr.AttributeExpr{Type: expr.String}}}}},
	} {
		for _, serviceFirst := range []bool{false, true} {
			name := "wrapper first"
			if serviceFirst {
				name = "service first"
			}
			t.Run(tc.name+"/"+name, func(t *testing.T) {
				named := &expr.UserTypeExpr{TypeName: "Named", UID: "Named", AttributeExpr: &expr.AttributeExpr{Type: tc.typ}}
				wrapper := &expr.AttributeExpr{Type: named}
				wrapAttr(wrapper, "Wrapped", true, &ServiceData{Name: "svc"})
				random := expr.NewRandom("cache-order")
				var serviceExample, wrapperExample any
				if serviceFirst {
					serviceExample = named.Example(random)
					wrapperExample = wrapper.Example(random)
				} else {
					wrapperExample = wrapper.Example(random)
					serviceExample = named.Example(random)
				}
				require.IsType(t, "", serviceExample)
				object, ok := wrapperExample.(map[string]any)
				require.True(t, ok, "wrapper example has type %T", wrapperExample)
				require.Len(t, object, 1)
				require.IsType(t, "", object["field"])
				cli, ok := protoJSONExample(wrapper, random).(map[string]any)
				require.True(t, ok, "CLI must describe a protobuf message")
				require.Len(t, cli, 1)
				for _, value := range cli {
					require.IsType(t, "", value)
				}
			})
		}
	}
}

func TestProtoWrappedNamedExampleUsesEffectiveConstraints(t *testing.T) {
	const validUUID = "550e8400-e29b-41d4-a716-446655440000"
	base := &expr.UserTypeExpr{
		TypeName: "FormatBase",
		UID:      "FormatBase",
		AttributeExpr: &expr.AttributeExpr{
			Type:       expr.String,
			Validation: &expr.ValidationExpr{Values: []any{"not-a-uuid", validUUID}},
		},
	}
	derived := &expr.UserTypeExpr{
		TypeName: "FormatDerived",
		UID:      "FormatDerived",
		AttributeExpr: &expr.AttributeExpr{
			Type:       base,
			Validation: &expr.ValidationExpr{Format: expr.FormatUUID},
		},
	}
	payload := &expr.AttributeExpr{
		Type: &expr.Object{{Name: "formatted", Attribute: &expr.AttributeExpr{Type: derived}}},
		Validation: &expr.ValidationExpr{
			Required: []string{"formatted"},
		},
	}

	serviceExample, ok := payload.Example(expr.NewRandom("effective-proto-example")).(map[string]any)
	require.True(t, ok)
	require.Equal(t, validUUID, serviceExample["formatted"])

	message := makeProtoBufMessage(payload, "CheckRequest", &ServiceData{Name: "svc"})
	protoExample, ok := protoJSONExample(message, expr.NewRandom("effective-proto-example")).(map[string]any)
	require.True(t, ok)
	require.Equal(t, validUUID, protoExample["formatted"])
}
