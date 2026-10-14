package expr_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

// TestHTTPErrorResponsesPreparedAtDeclaringScope checks that original responses
// and inherited copies satisfy the same preparation invariant before validation.
func TestHTTPErrorResponsesPreparedAtDeclaringScope(t *testing.T) {
	for _, scope := range []string{"api", "service", "method"} {
		for _, shape := range []string{"primitive", "named", "array", "object", "header"} {
			t.Run(scope+"/"+shape, func(t *testing.T) {
				root := expr.SetupTestDSL(t)
				require.True(t, eval.Execute(func() {
					var typ expr.DataType = String
					switch shape {
					case "named":
						typ = Type("ErrorText", String)
					case "array":
						typ = ArrayOf(String)
					case "object":
						typ = Type("ErrorObject", func() {
							Attribute("message", String)
						})
					}
					response := func() {
						Response("problem", StatusBadRequest, func() {
							if shape == "header" {
								Header("value:X-Problem")
							}
						})
					}
					API("prepared", func() {
						if scope == "api" {
							Error("problem", typ)
							HTTP(response)
						}
					})
					Service("prepared", func() {
						if scope == "service" {
							Error("problem", typ)
							HTTP(response)
						}
						Method("show", func() {
							Error("problem", typ)
							HTTP(func() {
								GET("/show")
								if scope == "method" {
									response()
								}
							})
						})
					})
				}, nil), "execute %s/%s", scope, shape)
				err := eval.RunDSL()
				assert.NoError(t, err)
				service := root.API.HTTP.Services[0]
				endpoint := service.HTTPEndpoints[0]
				var original *expr.HTTPErrorExpr
				switch scope {
				case "api":
					original = root.API.HTTP.Errors[0]
				case "service":
					original = service.HTTPErrors[0]
				case "method":
					original = endpoint.HTTPErrors[0]
				}
				require.NotNil(t, original.Response.Headers, "declaring response is prepared")
				if scope == "api" {
					headers := original.Response.Headers
					root.API.HTTP.Prepare()
					assert.Same(t, headers, original.Response.Headers, "repeated preparation preserves authored state")
				}
				require.Len(t, endpoint.HTTPErrors, 1)
				inherited := endpoint.HTTPErrors[0].Response
				require.NotNil(t, inherited.Headers)
				assert.Equal(t, expr.StatusBadRequest, inherited.StatusCode)
				if scope != "method" {
					assert.NotSame(t, original.Response, inherited)
				}
				if shape == "header" {
					assert.Equal(t, "X-Problem", original.Response.Headers.ElemName("value"))
					assert.Equal(t, "X-Problem", inherited.Headers.ElemName("value"))
				} else {
					assert.True(t, original.Response.Headers.IsEmpty())
					assert.True(t, inherited.Headers.IsEmpty())
				}
			})
		}
	}
}
