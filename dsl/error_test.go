package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestErrorStringArgumentDiagnostic(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		Service("songs", func() {
			Method("get", func() {
				Error("bad_request", "The request or song ID is invalid.")
			})
		})
	})

	require.Contains(t, err.Error(), `error descriptions are set with a DSL function: Error("bad_request", func() { Description("The request or song ID is invalid.") })`)
}

func TestErrorStringArgumentNamesUserType(t *testing.T) {
	root := expr.RunDSL(t, func() {
		customError := Type("CustomError", func() {
			Attribute("message", String)
		})
		Service("songs", func() {
			Error("bad_request", "CustomError")
		})
		_ = customError
	})

	require.Equal(t, "CustomError", root.Service("songs").Errors[0].AttributeExpr.Type.Name())
}
