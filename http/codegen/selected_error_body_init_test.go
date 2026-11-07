package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestSelectedProblemBodyInitTarget(t *testing.T) {
	for _, field := range []string{"type", "title", "status", "detail", "instance", "code", "retry_hint", "timeout", "temporary", "fault"} {
		t.Run(field, func(t *testing.T) {
			root := RunHTTPDSL(t, func() {
				Service("problems", func() {
					Method("selected", func() {
						Error("oops")
						HTTP(func() {
							GET("/selected")
							Response("oops", StatusBadRequest, func() {
								Body(field)
							})
						})
					})
				})
			})
			data := CreateHTTPServices(root).Get("problems")
			response := data.Endpoint("selected").Errors[0].Errors[0].Response
			init := response.ResultInit
			require.NotEmpty(t, response.ResultAttr, "the server retains the selected public problem field")
			require.Empty(t, init.ReturnTypeAttribute, "the client adapter returns the whole native error")
			require.False(t, init.ReturnIsPrimitivePointer)
			require.False(t, init.ReturnIsUnionValue)
			require.True(t, init.SkipFieldInit)
			require.Contains(t, init.ClientCode, "v := loomhttp.ProblemErrorFromBody(")
		})
	}
}
