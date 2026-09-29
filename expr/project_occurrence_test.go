package expr

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProjectPreservesFieldOccurrences(t *testing.T) {
	for _, named := range []bool{false, true} {
		for _, reversed := range []bool{false, true} {
			for _, defaulted := range []bool{false, true} {
				t.Run(fmt.Sprintf("named=%t/reversed=%t/default=%t", named, reversed, defaulted), func(t *testing.T) {
					var typ DataType = Bytes
					if named {
						typ = &UserTypeExpr{TypeName: "Blob", AttributeExpr: &AttributeExpr{Type: Bytes}}
					}
					names := []string{"plain", "nullable"}
					if reversed {
						names[0], names[1] = names[1], names[0]
					}
					rt := resultType(names[0], typ, names[1], typ,
						view(DefaultView, names[0], typ, names[1], typ))
					plain := AsObject(rt.Type).Attribute("plain")
					nullable := AsObject(rt.Type).Attribute("nullable")
					plain.Meta = MetaExpr{"occurrence": {"plain"}}
					nullable.Meta = MetaExpr{"occurrence": {"nullable"}}
					nullable.Nullable = true
					nullable.UserExamples = []*ExampleExpr{{Value: []byte("nullable")}}
					if defaulted {
						nullable.DefaultValue = []byte("default")
					}
					projected, err := Project(rt, DefaultView)
					require.NoError(t, err)
					actualPlain := AsObject(projected.Type).Attribute("plain")
					actualNullable := AsObject(projected.Type).Attribute("nullable")
					require.NotSame(t, actualPlain, actualNullable)
					require.False(t, actualPlain.Nullable)
					require.True(t, actualNullable.Nullable)
					require.Nil(t, actualPlain.DefaultValue)
					require.Equal(t, nullable.DefaultValue, actualNullable.DefaultValue)
					require.Equal(t, plain.Meta, actualPlain.Meta)
					require.Equal(t, nullable.Meta, actualNullable.Meta)
					require.Len(t, actualNullable.UserExamples, 1)
					require.Equal(t, nullable.UserExamples[0].Value, actualNullable.UserExamples[0].Value)
					require.Same(t, nullable.UserExamples[0], valueExampleOrigin(actualNullable.UserExamples[0]))
					actualNullable.Meta["occurrence"][0] = "changed"
					require.Equal(t, "nullable", nullable.Meta["occurrence"][0])
				})
			}
		}
	}
}

func TestProjectResolvesChildViewBeforeReuse(t *testing.T) {
	for _, reversed := range []bool{false, true} {
		t.Run(fmt.Sprintf("reversed=%t", reversed), func(t *testing.T) {
			child := resultType("a", String, "b", Int,
				view(DefaultView, "a", String, "b", Int), view("tiny", "a", String))
			fields := make([]any, 0, 5)
			fields = append(fields, "full", child, "small", child)
			selection := []any{"full", child, "small:tiny", child}
			if reversed {
				fields = append(fields[:0], "small", child, "full", child)
				selection = []any{"small:tiny", child, "full", child}
			}
			parent := resultType(append(fields, view(DefaultView, selection...))...)
			projected, err := Project(parent, DefaultView)
			require.NoError(t, err)
			full := AsObject(AsObject(projected.Type).Attribute("full").Type)
			small := AsObject(AsObject(projected.Type).Attribute("small").Type)
			require.Len(t, *full, 2)
			require.Len(t, *small, 1)
			require.NotNil(t, small.Attribute("a"))
		})
	}
}

func TestProjectCollectionOwnsDerivedNamingProvenance(t *testing.T) {
	for _, selected := range []string{DefaultView, "tiny"} {
		for _, canonical := range []bool{false, true} {
			t.Run(fmt.Sprintf("view=%s/canonical=%t", selected, canonical), func(t *testing.T) {
				element := resultType("a", String, "b", Int,
					view(DefaultView, "a", String, "b", Int), view("tiny", "a", String))
				element.TypeName = "RecipeResponse"
				source := collection(element)
				source.Identifier = "application/vnd.recipe.collection"
				source.TypeName = "ListRecipesResponseBody"
				source.Meta = MetaExpr{"name:original": {"RecipeCollection"}, "occurrence": {"body"}}
				if canonical {
					source.Meta["openapi:typename"] = []string{"PublicRecipes"}
					source.Meta["openapi:typename:canonical"] = []string{"true"}
				}
				projected, err := Project(source, selected)
				require.NoError(t, err)
				child := AsArray(projected.Type).ElemType.Type.(*ResultTypeExpr)
				require.Equal(t, child.TypeName+"Collection", projected.TypeName)
				require.NotContains(t, projected.Meta, "name:original", "a new projected declaration cannot inherit its unprojected body's generated name")
				require.Equal(t, []string{"RecipeCollection"}, source.Meta["name:original"])
				require.Equal(t, source.Meta["openapi:typename"], projected.Meta["openapi:typename"])
				require.Equal(t, source.Meta["openapi:typename:canonical"], projected.Meta["openapi:typename:canonical"])
				projected.Meta["occurrence"][0] = "changed"
				require.Equal(t, []string{"body"}, source.Meta["occurrence"])
			})
		}
	}
}
