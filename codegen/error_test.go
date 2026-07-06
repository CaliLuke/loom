package codegen

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestNewErrorPreservesExistingCodegenError(t *testing.T) {
	service := &expr.ServiceExpr{Name: "Service"}
	method := &expr.MethodExpr{Name: "Method"}
	inner := NewError(NewSilentContext().WithService(service).WithMethod(method), nil, errors.New("boom"))

	outer := NewError(NewSilentContext().WithService(&expr.ServiceExpr{Name: "Other"}), nil, inner)

	require.Same(t, inner, outer)
	require.Contains(t, outer.Error(), "service Service, method Method")
}
