package codegen

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func TestCollectionWrapperContentsPresence(t *testing.T) {
	for _, kind := range []string{"array", "map"} {
		for _, minimum := range []int{0, 1} {
			for _, position := range []string{"root", "named", "array element", "map value"} {
				t.Run(fmt.Sprintf("%s/min%d/%s", kind, minimum, position), func(t *testing.T) {
					collection := collectionWrapperInput(kind, minimum)
					input := collection
					switch position {
					case "named":
						input = &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Named", UID: "Named", AttributeExpr: collection}}
					case "array element":
						input = &expr.AttributeExpr{Type: &expr.Array{ElemType: collection}}
					case "map value":
						input = &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: collection}}
					}
					sd := &ServiceData{Name: "Svc", Scope: codegen.NewNameScope()}
					message := makeProtoBufMessage(input, "Root", sd)
					wrapper := message
					switch position {
					case "array element":
						wrapper = expr.AsArray(message.Find("field").Type).ElemType
					case "map value":
						wrapper = expr.AsMap(message.Find("field").Type).ElemType
					}
					require.False(t, wrapper.IsRequired("field"), "message presence must not become contents requiredness")
					require.Equal(t, &minimum, wrapper.Find("field").Validation.MinLength)
					require.Equal(t, &minimum, collection.Validation.MinLength, "normalization must not mutate source")
				})
			}
		}
	}
}

func TestCollectionWrapperContainingPresence(t *testing.T) {
	collection := &expr.UserTypeExpr{TypeName: "Items", UID: "Items", AttributeExpr: collectionWrapperInput("array", 0)}
	input := &expr.AttributeExpr{
		Type:       &expr.Object{{Name: "items", Attribute: &expr.AttributeExpr{Type: collection, Meta: expr.MetaExpr{"rpc:tag": {"1"}}}}},
		Validation: &expr.ValidationExpr{Required: []string{"items"}},
	}
	sd := &ServiceData{Name: "Svc", Scope: codegen.NewNameScope()}
	message := makeProtoBufMessage(input, "Root", sd)
	require.True(t, message.IsRequired("items"))
	require.False(t, message.Find("items").IsRequired("field"))
	ctx := protoBufTypeContext("pb", sd.Scope, false)
	validation := codegen.ValidationCode(message, message.Type.(expr.UserType), ctx, true, false, false, "message")
	require.Contains(t, validation, "message.Items == nil")
}

func TestCollectionWrapperNonCollectionRequiredControls(t *testing.T) {
	union := &expr.Union{TypeName: "Choice", Values: []*expr.NamedAttributeExpr{{Name: "text", Attribute: &expr.AttributeExpr{Type: expr.String}}}}
	for _, dt := range []expr.DataType{expr.String, union} {
		t.Run(dt.Name(), func(t *testing.T) {
			sd := &ServiceData{Name: "Svc", Scope: codegen.NewNameScope()}
			message := makeProtoBufMessage(&expr.AttributeExpr{Type: dt}, "Scalar", sd)
			field := "field"
			if expr.IsUnion(dt) {
				field = unionWrapperField(message).Name
			}
			require.True(t, message.IsRequired(field))
		})
	}
}

func collectionWrapperInput(kind string, minimum int) *expr.AttributeExpr {
	var dt expr.DataType = &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Int}}
	if kind == "map" {
		dt = &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.Int}}
	}
	return &expr.AttributeExpr{Type: dt, Validation: &expr.ValidationExpr{MinLength: &minimum}}
}
