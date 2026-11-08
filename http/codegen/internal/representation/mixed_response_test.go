package representation

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestMixedResponseWholeResultAuthority(t *testing.T) {
	for _, kind := range []string{"object", "primitive", "collection"} {
		t.Run(kind, func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				opening := dsl.Type("Opening", func() {
					dsl.Attribute("status", dsl.Int)
					dsl.Attribute("session", dsl.String)
					dsl.Required("status", "session")
				})
				dsl.Service("mixed", func() {
					dsl.Method("read", func() {
						switch kind {
						case "object":
							dsl.Result(func() {
								dsl.Attribute("status", dsl.String)
								dsl.Attribute("session", dsl.String)
								dsl.Required("status", "session")
							})
						case "primitive":
							dsl.Result(dsl.Int)
						case "collection":
							dsl.Result(dsl.ArrayOf(dsl.String))
						}
						dsl.StreamingResult(dsl.String)
						dsl.HTTP(func() {
							dsl.GET("/")
							dsl.ServerSentEvents()
							dsl.Response(dsl.StatusCreated, opening, func() {
								dsl.StatusField("status")
								dsl.Header("session:X-Session")
							})
						})
					})
				})
			})
			owner := service.NewSemanticData(root)
			method := owner.Method(root.Services[0].Methods[0])
			prepared, err := PrepareService(root.API.HTTP.Services[0], nil, owner)
			require.NoError(t, err)
			endpoint := prepared.Endpoints[0]
			response := endpoint.Response.Responses[0]
			require.NotNil(t, response.Unary)
			require.NotNil(t, response.Occurrence.UnaryBody)
			require.Equal(t, expr.Empty, response.Body.Type, "metadata has no payload")
			require.Equal(t, []string{"application/json"}, response.Unary.ContentTypes)
			require.NotNil(t, response.Unary.BodyValue)
			require.NoError(t, response.Unary.BodyValue.Error)
			require.Same(t, method.ResultValue.Context, response.Unary.BodyValue.Source.Context)
			require.Equal(t, method.ResultValue.Occurrence, response.Unary.BodyValue.Source.Occurrence)
			require.Equal(t, method.StreamingResultValue.Occurrence, endpoint.Stream.ResponseValue.Source.Occurrence)
			require.NotEqual(t, response.Headers[0].Value.Source.Occurrence, response.Unary.BodyValue.Source.Occurrence)
			require.True(t, response.Unary.BodyValue.Plan.Root().Valid())
			require.Len(t, response.Unary.BodyValues, 1)
			switch kind {
			case "object":
				require.Equal(t, expr.String, response.Unary.Body.Find("status").Type)
				require.Equal(t, expr.String, response.Unary.Body.Find("session").Type)
			case "primitive":
				require.True(t, expr.IsPrimitive(response.Unary.Body.Type))
			default:
				require.True(t, expr.IsArray(response.Unary.Body.Type))
			}
		})
	}
}
