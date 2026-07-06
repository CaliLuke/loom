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

func TestNewErrorInfersServiceFromExpression(t *testing.T) {
	service := &expr.ServiceExpr{Name: "Service"}

	err := NewError(nil, service, errors.New("boom"))

	require.Error(t, err)
	require.Contains(t, err.Error(), "boom")
	require.Contains(t, err.Error(), "service Service")
}

func TestNewErrorEnrichesIncompleteExistingCodegenError(t *testing.T) {
	service := &expr.ServiceExpr{Name: "Service"}
	method := &expr.MethodExpr{Name: "Method"}
	inner := NewError(nil, method, errors.New("boom"))

	outer := NewError(NewSilentContext().WithService(service).WithMethod(method), nil, inner)

	require.Same(t, inner, outer)
	require.Contains(t, outer.Error(), "service Service, method Method")
}
