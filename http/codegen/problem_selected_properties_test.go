package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

func TestProblemSelectedPropertyConstruction(t *testing.T) {
	endpoint := firstEndpointData(t, func() {
		dsl.Service("properties", func() {
			dsl.Method("selected", func() {
				dsl.Error("oops")
				dsl.HTTP(func() {
					dsl.GET("/selected")
					dsl.Response("oops", dsl.StatusBadRequest, func() {
						dsl.Body("detail")
					})
				})
			})
		})
	})
	code := endpoint.Errors[0].Errors[0].Response.ResultInit.ClientCode
	for _, assignment := range []string{"problem.Timeout = timeout", "problem.Temporary = temporary", "problem.Fault = fault"} {
		require.Contains(t, code, assignment)
	}
	require.Contains(t, code, "loomhttp.ProblemErrorFromBody(*problem)")
}
