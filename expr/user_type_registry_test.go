package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserTypeRegistry(t *testing.T) {
	t.Run("initializes empty", func(t *testing.T) {
		registry := NewUserTypeRegistry()

		require.NotNil(t, registry)
		require.Empty(t, registry.Collect())
	})

	t.Run("creates once and reuses existing type", func(t *testing.T) {
		registry := NewUserTypeRegistry()
		calls := 0

		first := registry.GetOrCreate("Action", func() *AttributeExpr {
			calls++
			return &AttributeExpr{Type: String}
		})
		second := registry.GetOrCreate("Action", func() *AttributeExpr {
			calls++
			return &AttributeExpr{Type: Int}
		})

		require.Same(t, first, second)
		require.Equal(t, 1, calls)
		require.Equal(t, "Action", first.TypeName)
		require.Equal(t, String, first.Type)
	})

	t.Run("attribute references named user type", func(t *testing.T) {
		registry := NewUserTypeRegistry()

		att := registry.Attribute("Action", func() *AttributeExpr {
			return &AttributeExpr{Type: String}
		})

		ut, ok := att.Type.(*UserTypeExpr)
		require.True(t, ok)
		require.Equal(t, "Action", ut.TypeName)
	})

	t.Run("collects types in deterministic order", func(t *testing.T) {
		registry := NewUserTypeRegistry()
		for _, name := range []string{"Gamma", "Alpha", "Beta"} {
			registry.GetOrCreate(name, func() *AttributeExpr {
				return &AttributeExpr{Type: String}
			})
		}

		collected := registry.Collect()

		require.Len(t, collected, 3)
		require.Equal(t, "Alpha", collected[0].(*UserTypeExpr).TypeName)
		require.Equal(t, "Beta", collected[1].(*UserTypeExpr).TypeName)
		require.Equal(t, "Gamma", collected[2].(*UserTypeExpr).TypeName)
	})

	t.Run("works in temporary root lifecycle", func(t *testing.T) {
		registry := NewUserTypeRegistry()
		payload := registry.Attribute("Action", func() *AttributeExpr {
			return &AttributeExpr{
				Type: &Object{
					{Name: "name", Attribute: &AttributeExpr{Type: String}},
				},
			}
		})
		service := &ServiceExpr{
			Name: "calc",
			Methods: []*MethodExpr{{
				Name:    "add",
				Payload: payload,
				Result:  &AttributeExpr{Type: String},
			}},
		}
		root := &RootExpr{
			Services: []*ServiceExpr{service},
			Types:    registry.Collect(),
			API: &APIExpr{
				Name: "calc",
				HTTP: &HTTPExpr{},
				JSONRPC: &JSONRPCExpr{
					HTTPExpr: HTTPExpr{
						Services: []*HTTPServiceExpr{NewJSONRPCHTTPService(service, "/rpc")},
					},
				},
				GRPC: &GRPCExpr{},
				ExampleGenerator: &ExampleGenerator{
					Randomizer: NewFakerRandomizer("calc"),
				},
			},
		}
		root.API.JSONRPC.Services[0].Root = &root.API.JSONRPC.HTTPExpr

		err := PrepareValidateFinalize(root)

		require.NoError(t, err)
		require.Len(t, root.Types, 1)
		require.Equal(t, "Action", root.Types[0].(*UserTypeExpr).TypeName)
	})
}
