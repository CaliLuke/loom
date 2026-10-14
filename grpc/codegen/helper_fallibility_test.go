package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func TestConversionFallibility(t *testing.T) {
	for _, tc := range []struct {
		name string
		leaf expr.DataType
		want bool
	}{
		{"scalar", expr.String, false},
		{"Any", expr.Any, true},
	} {
		for _, recursive := range []bool{false, true} {
			t.Run(tc.name, func(t *testing.T) {
				alias := &expr.UserTypeExpr{TypeName: "Alias", AttributeExpr: &expr.AttributeExpr{Type: tc.leaf}}
				fields := expr.Object{{Name: "value", Attribute: &expr.AttributeExpr{Type: alias}}}
				root := &expr.UserTypeExpr{TypeName: "Root", AttributeExpr: &expr.AttributeExpr{Type: &fields}}
				if recursive {
					fields = append(fields, &expr.NamedAttributeExpr{Name: "children", Attribute: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: root}}}})
				}
				attribute := &expr.AttributeExpr{Type: root}
				pure := &expr.AttributeExpr{Type: expr.String}
				require.Equal(t, tc.want || recursive, conversionCanFail(attribute, pure, serviceTypeContext("svc", codegen.NewNameScope()), serviceTypeContext("svc", codegen.NewNameScope())))
				require.Equal(t, tc.want, conversionCanFail(pure, attribute, serviceTypeContext("svc", codegen.NewNameScope()), serviceTypeContext("svc", codegen.NewNameScope())))
			})
		}
	}
}

func TestProtoHelperErrorContract(t *testing.T) {
	for _, parentFallible := range []bool{false, true} {
		for _, helperFallible := range []bool{false, true} {
			name := "pure helper"
			if helperFallible {
				name = "fallible helper"
			}
			if parentFallible {
				name += "/fallible parent"
			} else {
				name += "/pure parent"
			}
			t.Run(name, func(t *testing.T) {
				fieldType := expr.String
				if helperFallible {
					fieldType = expr.Any
				}
				source := &expr.AttributeExpr{Type: &expr.UserTypeExpr{
					TypeName: "Leaf",
					AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{
						{Name: "value", Attribute: &expr.AttributeExpr{Type: fieldType}},
					}},
				}}
				target := expr.DupAtt(source)
				scope := codegen.NewNameScope()
				ta := &transformAttrs{
					TransformAttrs: &codegen.TransformAttrs{
						SourceCtx: serviceTypeContext("svc", scope),
						TargetCtx: protoBufTypeContext("pb", scope, false),
						Prefix:    "svc",
					},
					proto:      true,
					errorAware: parentFallible,
				}
				bodyContext := objectHelperTransformAttrs(source, target, ta)
				require.NotSame(t, ta, bodyContext)
				require.Equal(t, helperFallible, bodyContext.errorAware)
				require.Equal(t, parentFallible, ta.errorAware, "helper analysis must not mutate its caller")
				helper, seen, err := buildObjectHelper(source, target, source.Type.(expr.UserType).Attribute(), false, ta, make(map[string]*codegen.TransformFunctionData))
				require.NoError(t, err)
				require.False(t, seen)
				require.Equal(t, helperFallible, helper.ErrorAware)
				call := renderTransformHelperCall(source, target, "value", ta)
				if helperFallible {
					require.Contains(t, call, "(value, transformErr)")
					require.Contains(t, helper.Code, "*transformErr = err")
				} else {
					require.Contains(t, call, "(value)")
					require.NotContains(t, helper.Code, "transformErr")
				}
			})
		}
	}
}
