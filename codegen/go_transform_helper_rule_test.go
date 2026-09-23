package codegen

import (
	"fmt"
	"regexp"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

// TestGoTransformCallsExactlyCollectedHelpers checks that generated transform
// code calls a helper function exactly when GoTransform also returns that
// helper. It covers every combination of user type and inline object on the
// source and target side, for object fields, array elements, map values,
// nullable array elements and map values, and union branches.
func TestGoTransformCallsExactlyCollectedHelpers(t *testing.T) {
	element := func(user bool, name string) *expr.AttributeExpr {
		object := &expr.Object{{Name: "x", Attribute: &expr.AttributeExpr{Type: expr.String}}}
		if !user {
			return &expr.AttributeExpr{Type: object}
		}
		return &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: name, AttributeExpr: &expr.AttributeExpr{Type: object}}}
	}
	shapes := []struct {
		name string
		wrap func(*expr.AttributeExpr) *expr.AttributeExpr
	}{
		{"field", func(elem *expr.AttributeExpr) *expr.AttributeExpr {
			return elem
		}},
		{"array", func(elem *expr.AttributeExpr) *expr.AttributeExpr {
			return &expr.AttributeExpr{Type: &expr.Array{ElemType: elem}}
		}},
		{"map", func(elem *expr.AttributeExpr) *expr.AttributeExpr {
			return &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: elem}}
		}},
		{"nullable-array", func(elem *expr.AttributeExpr) *expr.AttributeExpr {
			return &expr.AttributeExpr{Type: &expr.Array{ElemType: nullable(elem)}}
		}},
		{"nullable-map", func(elem *expr.AttributeExpr) *expr.AttributeExpr {
			return &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: nullable(elem)}}
		}},
		{"union", func(elem *expr.AttributeExpr) *expr.AttributeExpr {
			return &expr.AttributeExpr{Type: &expr.Union{
				TypeName: "Choice",
				Values:   []*expr.NamedAttributeExpr{{Name: "elem", Attribute: elem}},
			}}
		}},
	}
	helperCall := regexp.MustCompile(`\btransform[A-Za-z0-9_]*\(`)
	for _, shape := range shapes {
		for _, sourceUser := range []bool{false, true} {
			for _, targetUser := range []bool{false, true} {
				name := fmt.Sprintf("%s/source-user=%t/target-user=%t", shape.name, sourceUser, targetUser)
				t.Run(name, func(t *testing.T) {
					wrapper := func(elem *expr.AttributeExpr) *expr.AttributeExpr {
						return &expr.AttributeExpr{Type: &expr.Object{{Name: "f", Attribute: shape.wrap(elem)}}}
					}
					source := wrapper(element(sourceUser, "SourceElem"))
					target := wrapper(element(targetUser, "TargetElem"))
					ctx := NewAttributeContext(false, false, true, "", NewNameScope())
					code, helpers, err := GoTransform(source, target, "source", "target", ctx, ctx, "", true)
					require.NoError(t, err)

					collected := make([]string, 0, len(helpers))
					bodies := code
					for _, helper := range helpers {
						collected = append(collected, helper.Name+"(")
						bodies += "\n" + helper.Code
					}
					called := helperCall.FindAllString(bodies, -1)
					slices.Sort(called)
					called = slices.Compact(called)
					assert.ElementsMatch(t, collected, called, "generated code must call exactly the collected helpers:\n%s", bodies)
					if sourceUser && targetUser {
						assert.Len(t, helpers, 1, "a user type pair converts through one helper")
					} else {
						assert.Empty(t, helpers, "an inline object side converts in place")
					}
				})
			}
		}
	}
}

func nullable(att *expr.AttributeExpr) *expr.AttributeExpr {
	dup := *att
	dup.Nullable = true
	return &dup
}
