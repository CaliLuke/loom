package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestCustomStringCrossingOccurrenceOrder(t *testing.T) {
	for _, customFirst := range []bool{false, true} {
		names := []string{"native", "custom"}
		if customFirst {
			names = []string{"custom", "native"}
		}
		sourceFields, targetFields := expr.Object{}, expr.Object{}
		for _, name := range names {
			source, target := &expr.AttributeExpr{Type: expr.String}, &expr.AttributeExpr{Type: expr.String}
			if name == "custom" {
				target.Meta = expr.MetaExpr{"struct:field:type": {"Custom"}}
			}
			sourceFields.Set(name, source)
			targetFields.Set(name, target)
		}
		source := &expr.AttributeExpr{Type: &sourceFields, Validation: &expr.ValidationExpr{Required: names}}
		target := &expr.AttributeExpr{Type: &targetFields, Validation: &expr.ValidationExpr{Required: names}}
		context := NewAttributeContext(false, false, false, "", NewNameScope())
		plan, err := GoTransform(source, target, "source", "target", context, context, "", true)
		require.NoError(t, err)
		require.True(t, plan.ErrorAware, "custom first: %t", customFirst)
		require.Contains(t, plan.Code, "loom.ParseStringValue[Custom]")
		require.Contains(t, plan.Code, "Native: source.Native")
	}
}

func TestCustomStringDefaultRecursivePlan(t *testing.T) {
	for _, defaults := range []bool{false, true} {
		for _, includeLeaf := range []bool{false, true} {
			for _, leafFirst := range []bool{false, true} {
				source, target := customDefaultCycle(includeLeaf, leafFirst)
				scope := NewNameScope()
				sourceContext := NewAttributeContext(true, false, false, "views", scope)
				targetContext := NewAttributeContext(false, false, defaults, "service", scope)
				plan, err := GoTransform(source, target, "source", "target", sourceContext, targetContext, "convert", true)
				require.NoError(t, err)
				require.Equal(t, defaults && includeLeaf, plan.ErrorAware)
				require.NotEmpty(t, plan.Helpers)
				for _, helper := range plan.Helpers {
					require.Equal(t, defaults && includeLeaf, helper.ErrorAware, helper.Name)
				}
			}
		}
	}
}

func TestCustomStringCopyNeedsNoCodec(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:field:type": {"Opaque"}}}
	context := NewAttributeContext(false, false, true, "", NewNameScope())
	plan, err := GoTransform(attribute, attribute, "source", "target", context, context, "", true)
	require.NoError(t, err)
	require.False(t, plan.ErrorAware)
	require.Equal(t, "target := source", plan.Code)
}

func customDefaultCycle(includeLeaf, leafFirst bool) (*expr.AttributeExpr, *expr.AttributeExpr) {
	build := func(prefix string) *expr.AttributeExpr {
		aFields, bFields := expr.Object{}, expr.Object{}
		a := &expr.UserTypeExpr{TypeName: prefix + "A", AttributeExpr: &expr.AttributeExpr{Type: &aFields}}
		b := &expr.UserTypeExpr{TypeName: prefix + "B", AttributeExpr: &expr.AttributeExpr{Type: &bFields}}
		leaf := &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:field:type": {"scalar.Token", "example.com/scalar"}}, DefaultValue: "token:default"}
		if includeLeaf && leafFirst {
			bFields.Set("id", leaf)
		}
		aFields.Set("next", &expr.AttributeExpr{Type: b})
		bFields.Set("next", &expr.AttributeExpr{Type: a})
		if includeLeaf && !leafFirst {
			bFields.Set("id", leaf)
		}
		return &expr.AttributeExpr{Type: a}
	}
	return build("Source"), build("Target")
}
