package expr

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnionBranchNamesIgnoreDeclarationOrder(t *testing.T) {
	orders := [][]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}}
	for _, definitions := range orders {
		for _, roots := range orders {
			t.Run(fmt.Sprintf("definitions%v/roots%v", definitions, roots), func(t *testing.T) {
				root := SetupTestDSL(t)
				branches := make([]*UserTypeExpr, 3)
				for _, index := range definitions {
					name := "rS"
					if index == 2 {
						name = "rS3"
					}
					branches[index] = root.NewUnionBranch(name, &AttributeExpr{Type: []DataType{String, Int, Boolean}[index]})
				}
				for _, index := range roots {
					root.Types = append(root.Types, &UserTypeExpr{
						TypeName:      []string{"A", "B", "C"}[index],
						AttributeExpr: &AttributeExpr{Type: &Object{{Name: "field", Attribute: &AttributeExpr{Type: branches[index]}}}},
					})
				}
				root.Types = append(root.Types, &UserTypeExpr{TypeName: "rS2", AttributeExpr: &AttributeExpr{Type: String}})
				copy := Dup(&Object{
					{Name: "first", Attribute: &AttributeExpr{Type: branches[0]}},
					{Name: "second", Attribute: &AttributeExpr{Type: branches[1]}},
				})
				copyFirst := AsObject(copy).Attribute("first").Type.(*UserTypeExpr)
				copySecond := AsObject(copy).Attribute("second").Type.(*UserTypeExpr)
				require.Equal(t, String, copyFirst.Type)
				require.Equal(t, Int, copySecond.Type)
				root.Types = append(root.Types, &UserTypeExpr{TypeName: "Z", AttributeExpr: &AttributeExpr{Type: copy}})
				root.Prepare()
				for index, expected := range []string{"rS", "rS4", "rS3"} {
					require.Equal(t, expected, branches[index].Name())
					require.Equal(t, expected, branches[index].ID())
				}
				require.Equal(t, branches[0].Name(), copyFirst.Name())
				require.Equal(t, branches[1].Name(), copySecond.Name())
				root.Prepare()
				require.Equal(t, "rS4", branches[1].Name(), "preparation must be idempotent")
			})
		}
	}
}

func TestUnionBranchCopyKeepsOnlyRequestedDefinition(t *testing.T) {
	for _, keepBranch := range []bool{false, true} {
		t.Run(fmt.Sprintf("keep-branch-%t", keepBranch), func(t *testing.T) {
			root := SetupTestDSL(t)
			branch := root.NewUnionBranch("rS", &AttributeExpr{Type: String})
			authored := &UserTypeExpr{TypeName: "rS", AttributeExpr: &AttributeExpr{Type: Int}}
			keep := authored
			if keepBranch {
				keep = branch
			}
			copy, kept := DupKeeping(&Object{
				{Name: "branch", Attribute: &AttributeExpr{Type: branch}},
				{Name: "authored", Attribute: &AttributeExpr{Type: authored}},
			}, []UserType{keep})
			require.Len(t, kept, 1)
			require.Same(t, keep, kept[0].Type)
			require.Equal(t, String, AsObject(copy).Attribute("branch").Type.(UserType).Attribute().Type)
			require.Equal(t, Int, AsObject(copy).Attribute("authored").Type.(UserType).Attribute().Type)
		})
	}
}
