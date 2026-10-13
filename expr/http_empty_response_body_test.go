package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestExplicitEmptyResponseBody(t *testing.T) {
	cases := []struct {
		name              string
		mode              string
		wantBody          bool
		wantDocumentation bool
	}{
		{"selected empty object", "selected", true, true},
		{"explicit empty type", "type", true, true},
		{"documentation only", "documentation", false, true},
		{"explicit Empty sentinel", "sentinel", false, false},
		{"inferred empty type", "implicit", false, false},
		{"all fields mapped to headers", "metadata", false, false},
		{"ordinary selected object", "ordinary", true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := expr.RunDSL(t, func() {
				shape := Type("Record", func() {
					Meta("openapi:additionalProperties", "false")
					if tc.mode == "ordinary" {
						Attribute("value", String)
					}
				})
				Service("bodies", func() {
					Method("show", func() {
						switch tc.mode {
						case "selected", "ordinary":
							Result(func() {
								Attribute("body", shape)
								Attribute("label", String)
								Required("body", "label")
							})
						case "metadata":
							Result(func() {
								Attribute("label", String)
							})
						default:
							Result(shape)
						}
						HTTP(func() {
							GET("/show")
							Response(StatusOK, func() {
								switch tc.mode {
								case "selected", "ordinary":
									Header("label:X-Label")
									Body("body")
								case "type":
									Body(shape)
								case "sentinel":
									Body(Empty)
									OpenAPIBody(Empty)
								case "metadata":
									Header("label:X-Label")
								}
								if tc.wantDocumentation {
									OpenAPIBody(shape)
								}
							})
						})
					})
				})
			})
			response := root.API.HTTP.Services[0].HTTPEndpoints[0].Responses[0]
			require.Equal(t, tc.wantBody, response.Body.Type != expr.Empty, "runtime body")
			if response.OpenAPIBody != nil {
				require.Equal(t, tc.wantDocumentation, response.OpenAPIBody.Type != expr.Empty, "documented body")
			}
			if tc.wantBody {
				require.True(t, expr.IsObject(response.Body.Type))
				if tc.mode != "ordinary" {
					require.Empty(t, *expr.AsObject(response.Body.Type))
				}
			}
		})
	}
}

// An empty Body callback does not declare a type; this repair preserves the
// existing rejection instead of introducing a new DSL constructor.
func TestUntypedEmptyBodyRemainsRejected(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		shape := Type("EmptyRecord", func() {
		})
		Service("bodies", func() {
			Method("show", func() {
				Result(shape)
				HTTP(func() {
					GET("/show")
					Response(StatusOK, func() {
						Body(func() {
						})
					})
				})
			})
		})
	})
	require.ErrorContains(t, err, "attribute type is nil")
}
