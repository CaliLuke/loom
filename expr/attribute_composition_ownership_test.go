package expr

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/eval"
)

func TestCompositionRecursiveConstraintSnapshots(t *testing.T) {
	for _, mutual := range []bool{false, true} {
		for _, nodeFirst := range []bool{false, true} {
			t.Run(map[bool]string{false: "self", true: "mutual"}[mutual]+map[bool]string{false: "/base first", true: "/node first"}[nodeFirst], func(t *testing.T) {
				eval.Reset()
				minimum := 0.0
				node := &UserTypeExpr{TypeName: "Node", AttributeExpr: &AttributeExpr{Type: &Object{
					{Name: "n", Attribute: &AttributeExpr{Type: Int, Validation: &ValidationExpr{Minimum: &minimum}, Meta: MetaExpr{"source": {"authored"}}}},
				}}}
				var target UserType = node
				if mutual {
					target = &UserTypeExpr{TypeName: "Other", AttributeExpr: &AttributeExpr{Type: &Object{
						{Name: "back", Attribute: &AttributeExpr{Type: node}},
					}}}
				}
				base := &UserTypeExpr{TypeName: "Base", AttributeExpr: &AttributeExpr{Type: &Object{
					{Name: "next", Attribute: &AttributeExpr{Type: target, Meta: MetaExpr{"source": {"base"}}}},
				}}}
				node.Bases = []DataType{base}
				root := &RootExpr{Types: []UserType{base, node}}
				if nodeFirst {
					root.Types = []UserType{node, base}
				}
				root.composeDesign()
				require.Empty(t, eval.Context.Errors)
				next := node.Find("next")
				require.NotSame(t, base.Find("next"), next)
				next.Meta["source"][0] = "derived"
				require.Equal(t, "base", base.Find("next").Meta["source"][0])
				canonical := next.Type
				if mutual {
					canonical = AsObject(canonical).Attribute("back").Type
				}
				require.Same(t, node, canonical)
				for _, snapshot := range []*AttributeExpr{node.constraintLocal, node.constraintParents[0]} {
					copied := snapshot.Find("next").Type
					if mutual {
						copied = AsObject(copied).Attribute("back").Type
					}
					require.NotSame(t, node, copied)
					field := AsObject(copied).Attribute("n")
					*node.Find("n").Validation.Minimum = 9
					node.Find("n").Meta["source"][0] = "changed"
					require.Equal(t, 0.0, *field.Validation.Minimum)
					require.Equal(t, "authored", field.Meta["source"][0])
				}
			})
		}
	}
}
