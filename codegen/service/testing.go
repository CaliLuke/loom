package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

// initDSL initializes the DSL environment and returns the root.
func initDSL(t *testing.T) *expr.RootExpr {
	t.Helper()
	return expr.SetupTestDSL(t)
}

// runDSL returns the DSL root resulting from running the given DSL.
func runDSL(t *testing.T, dsl func()) *expr.RootExpr {
	root := initDSL(t)
	require.True(t, eval.Execute(dsl, nil))
	require.NoError(t, eval.RunDSL())
	return root
}

// runDSLWithError returns the DSL root and error from running the given DSL.
func runDSLWithError(t *testing.T, dsl func()) (*expr.RootExpr, error) {
	root := initDSL(t)
	require.True(t, eval.Execute(dsl, nil))
	err := eval.RunDSL()
	require.Error(t, err)
	return root, err
}
