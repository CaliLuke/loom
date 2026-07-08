package expr

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/eval"
)

func TestPrepareValidateFinalize(t *testing.T) {
	t.Run("finalizes temporary root with isolated globals", func(t *testing.T) {
		SetupTestDSL(t)

		originalRoot := Root
		originalGeneratedResultTypes := GeneratedResultTypes
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
		require.Same(t, originalGeneratedResultTypes, GeneratedResultTypes)
		require.Same(t, originalContext, eval.Context)
		require.Equal(t, "calc", root.API.Name)
		require.Equal(t, "0.0.1", root.API.Version)
		require.Len(t, root.API.Servers, 1)
	})

	t.Run("returns validation errors without mutating caller context", func(t *testing.T) {
		SetupTestDSL(t)

		originalRoot := Root
		originalGeneratedResultTypes := GeneratedResultTypes
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
		require.Same(t, originalGeneratedResultTypes, GeneratedResultTypes)
		require.Same(t, originalContext, eval.Context)
		require.Len(t, eval.Context.Errors, 1)
		require.Equal(t, "existing", eval.Context.Errors[0].GoError.Error())
	})

	t.Run("rejects nil root", func(t *testing.T) {
		SetupTestDSL(t)

		err := PrepareValidateFinalize(nil)

		require.EqualError(t, err, "root cannot be nil")
	})

	t.Run("does not share attribute validation state across temporary roots", func(t *testing.T) {
		SetupTestDSL(t)

		shared := &AttributeExpr{
			Type:       &Object{},
			Validation: &ValidationExpr{Required: []string{"missing"}},
		}
		firstSvc := &ServiceExpr{Name: "First"}
		firstMethod := &MethodExpr{Name: "read", Payload: shared, Service: firstSvc}
		firstSvc.Methods = []*MethodExpr{firstMethod}
		first := &RootExpr{
			Services: []*ServiceExpr{firstSvc},
			API: &APIExpr{
				Name:    "first",
				HTTP:    &HTTPExpr{},
				JSONRPC: &JSONRPCExpr{},
				GRPC:    &GRPCExpr{},
			},
		}
		secondSvc := &ServiceExpr{Name: "Second"}
		secondMethod := &MethodExpr{Name: "read", Payload: shared, Service: secondSvc}
		secondSvc.Methods = []*MethodExpr{secondMethod}
		second := &RootExpr{
			Services: []*ServiceExpr{secondSvc},
			API: &APIExpr{
				Name:    "second",
				HTTP:    &HTTPExpr{},
				JSONRPC: &JSONRPCExpr{},
				GRPC:    &GRPCExpr{},
			},
		}

		firstErr := PrepareValidateFinalize(first)
		secondErr := PrepareValidateFinalize(second)

		require.Error(t, firstErr)
		require.Contains(t, firstErr.Error(), `service "First" method "read": payload - required field "missing" does not exist`)
		require.Error(t, secondErr)
		require.Contains(t, secondErr.Error(), `service "Second" method "read": payload - required field "missing" does not exist`)
	})

	t.Run("register default roots resets attribute validation state", func(t *testing.T) {
		SetupTestDSL(t)

		shared := &AttributeExpr{
			Type:       &Object{},
			Validation: &ValidationExpr{Required: []string{"missing"}},
		}
		validated[shared] = true

		eval.Reset()
		require.NoError(t, RegisterDefaultRoots())

		if validated[shared] {
			t.Fatal("default root registration reused stale attribute validation state")
		}
	})
}
