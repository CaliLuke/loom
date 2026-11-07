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
	require.NotContains(t, err.Error(), "panic:")
}

// Descriptive and validation modifiers do not declare an object's shape.
func TestBodyTemplateWithoutDeclaredShapeRejected(t *testing.T) {
	for _, context := range []string{"request", "response", "documentation", "request documentation"} {
		for _, modifier := range []struct {
			name string
			dsl  func()
		}{
			{name: "empty", dsl: func() {
			}},
			{name: "description", dsl: func() {
				Description("Body details")
			}},
			{name: "metadata", dsl: func() {
				Meta("test:body", "descriptive")
			}},
			{name: "required", dsl: func() {
				Required("value")
			}},
		} {
			t.Run(context+"/"+modifier.name, func(t *testing.T) {
				err := expr.RunInvalidDSL(t, func() {
					shape := Type("Record", func() {
						Attribute("value", String)
					})
					Service("bodies", func() {
						Method("show", func() {
							Payload(shape)
							Result(shape)
							HTTP(func() {
								POST("/show")
								if context == "request" {
									Body(modifier.dsl)
								}
								if context == "request documentation" {
									SkipRequestBodyEncodeDecode()
									Header("value")
									OpenAPIRequestBody(modifier.dsl, "application/json", true)
								}
								Response(StatusOK, func() {
									switch context {
									case "response":
										Body(modifier.dsl)
									case "documentation":
										OpenAPIBody(modifier.dsl)
									}
								})
							})
						})
					})
				})
				require.ErrorContains(t, err, "attribute type is nil")
				require.NotContains(t, err.Error(), "panic:")
			})
		}
	}
}

func TestBodyTemplateShapeDeclarations(t *testing.T) {
	for _, context := range []string{"request", "response", "documentation"} {
		for _, kind := range []string{"attribute", "extend", "reference"} {
			t.Run(context+"/"+kind, func(t *testing.T) {
				design := func() {
					shape := Type("Record", func() {
						Attribute("value", String)
					})
					callback := func() {
						switch kind {
						case "attribute":
							Attribute("value")
						case "extend":
							Extend(shape)
						case "reference":
							Reference(shape)
						}
					}
					Service("bodies", func() {
						Method("show", func() {
							Payload(shape)
							Result(shape)
							HTTP(func() {
								POST("/show")
								if context == "request" {
									Body(callback)
								}
								Response(StatusOK, func() {
									switch context {
									case "response":
										Body(callback)
									case "documentation":
										OpenAPIBody(callback)
									}
								})
							})
						})
					})
				}
				if kind == "reference" {
					err := expr.RunInvalidDSL(t, design)
					require.ErrorContains(t, err, "attribute type is nil")
					require.NotContains(t, err.Error(), "panic:")
					return
				}
				root := expr.RunDSL(t, design)
				endpoint := root.API.HTTP.Services[0].HTTPEndpoints[0]
				body := endpoint.Body
				switch context {
				case "response":
					body = endpoint.Responses[0].Body
				case "documentation":
					body = endpoint.Responses[0].OpenAPIBody
				}
				require.NotNil(t, body.Find("value"))
			})
		}
	}
}

func TestExplicitEmptyRequestBody(t *testing.T) {
	for _, tc := range []struct {
		name     string
		selected bool
	}{
		{name: "typed"},
		{name: "selected", selected: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := expr.RunDSL(t, func() {
				empty := Type("EmptyRecord", func() {
				})
				Service("bodies", func() {
					Method("show", func() {
						if tc.selected {
							Payload(func() {
								Attribute("body", empty)
							})
						} else {
							Payload(empty)
						}
						HTTP(func() {
							POST("/show")
							if tc.selected {
								Body("body")
							} else {
								Body(empty)
							}
						})
					})
				})
			})
			body := root.API.HTTP.Services[0].HTTPEndpoints[0].Body
			require.NotNil(t, body)
			require.True(t, expr.IsObject(body.Type))
			require.Empty(t, *expr.AsObject(body.Type))
		})
	}
}
