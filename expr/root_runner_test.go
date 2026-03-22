package expr

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/v3/eval"
)

func TestPrepareValidateFinalize(t *testing.T) {
	t.Run("finalizes temporary root with isolated globals", func(t *testing.T) {
		ResetDSL(t)

		originalRoot := Root
		originalContext := eval.Context

		root := &RootExpr{
			Services: []*ServiceExpr{{Name: "calc"}},
			API: &APIExpr{
				HTTP:    &HTTPExpr{},
				JSONRPC: &JSONRPCExpr{},
				GRPC:    &GRPCExpr{},
			},
		}

		err := PrepareValidateFinalize(root)

		require.NoError(t, err)
		require.Same(t, originalRoot, Root)
		require.Same(t, originalContext, eval.Context)
		require.Equal(t, "calc", root.API.Name)
		require.Equal(t, "0.0.1", root.API.Version)
		require.Len(t, root.API.Servers, 1)
	})

	t.Run("returns validation errors without mutating caller context", func(t *testing.T) {
		ResetDSL(t)

		originalRoot := Root
		originalContext := eval.Context
		originalContext.Record(&eval.Error{GoError: errors.New("existing")})

		dup1 := &UserTypeExpr{
			TypeName:      "Conflict",
			AttributeExpr: &AttributeExpr{Type: String},
		}
		dup2 := &UserTypeExpr{
			TypeName:      "Conflict",
			AttributeExpr: &AttributeExpr{Type: Int},
		}
		root := &RootExpr{
			Types: []UserType{dup1, dup2},
			API: &APIExpr{
				Name:    "dup",
				HTTP:    &HTTPExpr{},
				JSONRPC: &JSONRPCExpr{},
				GRPC:    &GRPCExpr{},
			},
		}

		err := PrepareValidateFinalize(root)

		require.Error(t, err)
		require.Contains(t, err.Error(), `type "Conflict" defined twice`)
		require.Same(t, originalRoot, Root)
		require.Same(t, originalContext, eval.Context)
		require.Len(t, eval.Context.Errors, 1)
		require.Equal(t, "existing", eval.Context.Errors[0].GoError.Error())
	})

	t.Run("rejects nil root", func(t *testing.T) {
		ResetDSL(t)

		err := PrepareValidateFinalize(nil)

		require.EqualError(t, err, "root cannot be nil")
	})
}
