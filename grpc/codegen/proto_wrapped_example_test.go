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
