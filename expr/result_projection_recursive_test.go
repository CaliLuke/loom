package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRecursiveResultProjection(t *testing.T) {
	shapes := []struct {
		name string
		wrap func(DataType) DataType
		leaf func(DataType) DataType
	}{
		{"array", func(dt DataType) DataType {
			return &Array{ElemType: &AttributeExpr{Type: dt}}
		}, func(dt DataType) DataType {
			return AsArray(dt).ElemType.Type
		}},
		{"map", func(dt DataType) DataType {
			return &Map{KeyType: &AttributeExpr{Type: String}, ElemType: &AttributeExpr{Type: dt}}
		}, func(dt DataType) DataType {
			return AsMap(dt).ElemType.Type
		}},
		{"union", func(dt DataType) DataType {
			return &Union{TypeName: "Choice", Values: []*NamedAttributeExpr{{Name: "owner", Attribute: &AttributeExpr{Type: dt}}}}
		}, func(dt DataType) DataType {
			return AsUnion(dt).Values[0].Attribute.Type
		}},
		{"object", func(dt DataType) DataType {
			return &Object{{Name: "owner", Attribute: &AttributeExpr{Type: dt}}}
		}, func(dt DataType) DataType {
			return AsObject(dt).Attribute("owner").Type
		}},
	}
	for _, shape := range shapes {
		t.Run(shape.name, func(t *testing.T) {
			owner := resultType("id", String, "secret", String, view(DefaultView, "id", String), view("full", "id", String, "secret", String))
			wrapped := shape.wrap(owner)
			parent := resultType("nested", wrapped, view(DefaultView, "nested", wrapped), view("full", "nested", wrapped))
			for _, parentView := range []string{DefaultView, "full"} {
				projected, err := Project(parent, parentView)
				require.NoError(t, err)
				child := shape.leaf(AsObject(projected.Type).Attribute("nested").Type)
				require.Nil(t, AsObject(child).Attribute("secret"), "nested result selects its default independently of parent view")
				require.NotNil(t, AsObject(child).Attribute("id"))
				require.NotNil(t, AsObject(owner).Attribute("secret"), "projection cannot mutate the authored type")
			}
		})
	}
}
