package codegen

import (
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestProtobufScalarPresence(t *testing.T) {
	for _, dt := range []expr.DataType{expr.String, expr.Boolean, expr.Int, expr.Int32, expr.Int64, expr.UInt, expr.UInt32, expr.UInt64, expr.Float32, expr.Float64, expr.Bytes} {
		for _, required := range []bool{false, true} {
			t.Run(dt.Name(), func(t *testing.T) {
				field := &expr.AttributeExpr{Type: dt, Meta: expr.MetaExpr{"rpc:tag": {"1"}}}
				obj := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: field}}}
				if required {
					obj.Validation = &expr.ValidationExpr{Required: []string{"value"}}
				}
				sd := &ServiceData{Scope: codegen.NewNameScope()}
				require.Contains(t, protoBufMessageDef(obj, sd), "optional ")
				ctx := protoBufTypeContext("pb", sd.Scope, false)
				require.Equal(t, dt != expr.Bytes, ctx.IsPrimitivePointer("value", obj))
				ut := &expr.UserTypeExpr{TypeName: "Message", AttributeExpr: obj}
				validation := codegen.ValidationCode(obj, ut, ctx, true, false, false, "message")
				if required {
					require.Contains(t, validation, "message.Value == nil")
				}
			})
		}
	}
}

func TestProtobufScalarAliasLowering(t *testing.T) {
	for _, dt := range []expr.DataType{expr.String, expr.Bytes} {
		t.Run(dt.Name(), func(t *testing.T) {
			maximum := 3
			base := &expr.UserTypeExpr{TypeName: "Base", UID: "Base", AttributeExpr: &expr.AttributeExpr{
				Type: dt, Validation: &expr.ValidationExpr{MaxLength: &maximum},
			}}
			alias := &expr.UserTypeExpr{TypeName: "Alias", UID: "Alias", AttributeExpr: &expr.AttributeExpr{Type: base}}
			field := &expr.AttributeExpr{Type: alias, Meta: expr.MetaExpr{"rpc:tag": {"1"}}}
			object := &expr.AttributeExpr{
				Type:       &expr.Object{{Name: "value", Attribute: field}},
				Validation: &expr.ValidationExpr{Required: []string{"value"}},
			}
			sd := &ServiceData{Name: "Svc", Scope: codegen.NewNameScope()}
			message := makeProtoBufMessage(object, "Message", sd)
			physical := message.Find("value")
			require.Equal(t, dt, physical.Type)
			require.Equal(t, &maximum, physical.Validation.MaxLength)
			require.Same(t, alias, object.Find("value").Type)
			root := makeProtoBufMessage(&expr.AttributeExpr{Type: alias}, "Root", sd)
			require.Equal(t, dt, root.Find("field").Type, "root wrappers use the same scalar lowering")
			ctx := protoBufTypeContext("pb", sd.Scope, false)
			validation := codegen.ValidationCode(message, message.Type.(expr.UserType), ctx, true, false, false, "message")
			require.Contains(t, validation, "message.Value == nil")
			if dt == expr.Bytes {
				require.Contains(t, validation, "len(message.Value)")
				require.NotContains(t, validation, "*message.Value")
			} else {
				require.Contains(t, validation, "*message.Value")
			}
		})
	}
}
