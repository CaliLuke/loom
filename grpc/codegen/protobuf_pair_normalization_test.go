package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func TestProtobufPairNormalizationPreservesCanonicalShapes(t *testing.T) {
	leaf := &expr.AttributeExpr{Type: expr.String}
	union := &expr.AttributeExpr{Type: &expr.Union{TypeName: "Choice", Values: []*expr.NamedAttributeExpr{{Name: "text", Attribute: leaf}}}}
	array := &expr.AttributeExpr{Type: &expr.Array{ElemType: leaf}}
	mapping := &expr.AttributeExpr{Type: &expr.Map{KeyType: leaf, ElemType: leaf}}
	object := &expr.AttributeExpr{Type: &expr.Object{{Name: "text:wire", Attribute: leaf}}}
	for _, test := range []struct {
		name    string
		value   *expr.AttributeExpr
		wrapped bool
	}{
		{"union", union, true},
		{"array", array, true},
		{"map", mapping, true},
		{"scalar", leaf, true},
		{"object", object, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			message := test.value
			if test.wrapped {
				message = wrapperAttribute(test.value)
			}
			canonical := unwrapAttr(message)
			for _, toProto := range []bool{false, true} {
				source, target := message, test.value
				wantSource, wantTarget := canonical, test.value
				if toProto {
					source, target = test.value, message
					wantSource, wantTarget = test.value, canonical
				}
				gotSource, gotTarget, err := normalizedTransformAttrs(source, target, &transformAttrs{proto: toProto})
				require.NoError(t, err)
				require.Same(t, wantSource, gotSource)
				require.Same(t, wantTarget, gotTarget)
				if test.wrapped {
					require.True(t, expr.IsObject(message.Type), "normalization must not rewrite the message")
					require.Same(t, canonical, unwrapAttr(message))
				}
			}
		})
	}
	_, _, err := normalizedTransformAttrs(leaf, &expr.AttributeExpr{Type: expr.Boolean}, &transformAttrs{proto: true})
	require.Error(t, err)
}

func TestProtobufUnionPairFallibility(t *testing.T) {
	for _, custom := range []bool{false, true} {
		name := map[bool]string{false: "native", true: "custom"}[custom]
		t.Run(name, func(t *testing.T) {
			text := &expr.AttributeExpr{Type: expr.String}
			if custom {
				text.Meta = expr.MetaExpr{"struct:field:type": {"scalar.Text", "example.com/customgrpc/scalar"}}
			}
			union := &expr.Union{TypeName: "Choice"}
			choice := &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Choice", UID: "choice", AttributeExpr: &expr.AttributeExpr{Type: union}}}
			record := &expr.AttributeExpr{Type: &expr.Object{
				{Name: "again", Attribute: choice},
				{Name: "text:wire", Attribute: text},
			}}
			union.Values = []*expr.NamedAttributeExpr{
				{Name: "record", Attribute: record},
				{Name: "texts", Attribute: &expr.AttributeExpr{Type: &expr.Array{ElemType: text}}},
				{Name: "labels", Attribute: &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: text}}},
			}
			message := wrapperAttribute(expr.DupAtt(choice))
			scope := codegen.NewNameScope()
			serviceContext := serviceTypeContext("svc", scope)
			messageContext := protoBufTypeContext("pb", scope, false)
			require.Equal(t, custom, protobufRepresentationCanFail(choice, message, serviceContext, messageContext))
			require.Equal(t, custom, protobufRepresentationCanFail(message, choice, messageContext, serviceContext))
			_, originalText := record.FindAttribute("text")
			require.Same(t, text, originalText)
			require.Same(t, choice, (*expr.AsObject(record.Type))[0].Attribute)
		})
	}
}

func TestProtobufPairAnalysisDoesNotAllocateMessageNames(t *testing.T) {
	union := &expr.Union{TypeName: "Choice", Values: []*expr.NamedAttributeExpr{
		{Name: "text", Attribute: &expr.AttributeExpr{Type: expr.String}},
	}}
	service := &expr.AttributeExpr{Type: &expr.UserTypeExpr{
		TypeName: "Choice", UID: "choice", AttributeExpr: &expr.AttributeExpr{Type: union},
	}}
	message := &expr.AttributeExpr{Type: &expr.UserTypeExpr{
		TypeName: "Choice", UID: "choice#message", AttributeExpr: wrapperAttribute(expr.DupAtt(service.Type.(expr.UserType).Attribute())),
	}}
	for _, toProto := range []bool{false, true} {
		control := protoBufTypeContext("pb", codegen.NewNameScope(), false)
		want := control.Scope.Ref(message, "pb")
		protoContext := protoBufTypeContext("pb", codegen.NewNameScope(), false)
		serviceContext := serviceTypeContext("svc", codegen.NewNameScope())
		if toProto {
			require.False(t, protobufRepresentationCanFail(service, message, serviceContext, protoContext))
		} else {
			require.False(t, protobufRepresentationCanFail(message, service, protoContext, serviceContext))
		}
		require.Equal(t, want, protoContext.Scope.Ref(message, "pb"))
	}
}
