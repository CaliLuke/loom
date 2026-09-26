package expr_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestHTTPBodyOmittedOptionalFieldRejected checks that an optional attribute
// of an HTTP or JSON-RPC body whose JSON name is "-" is rejected with a message
// that names the attribute and the body. The generated decoders track the
// presence of optional fields, which a field omitted from JSON cannot have.
func TestHTTPBodyOmittedOptionalFieldRejected(t *testing.T) {
	cases := []struct {
		name string
		dsl  func()
		want string
	}{
		{
			name: "request body",
			dsl: omittedFieldDSL(func() {
				Payload(func() {
					Attribute("name", String)
					Attribute("secret", String, omitFromJSON)
				})
			}),
			want: `attribute "secret" in the HTTP request body is optional, but its JSON name "-" omits it from JSON`,
		},
		{
			name: "response body",
			dsl: omittedFieldDSL(func() {
				Result(func() {
					Attribute("name", String)
					Attribute("secret", String, omitFromJSON)
				})
			}),
			want: `attribute "secret" in the HTTP response body is optional, but its JSON name "-" omits it from JSON`,
		},
		{
			name: "nested type",
			dsl: func() {
				item := Type("Item", func() {
					Attribute("x", String)
					Attribute("secret", String, omitFromJSON)
				})
				omittedFieldDSL(func() {
					Payload(func() {
						Attribute("item", item)
					})
				})()
			},
			want: `attribute "secret" of type "Item" in the HTTP request body is optional, but its JSON name "-" omits it from JSON`,
		},
		{
			name: "required with default",
			dsl: omittedFieldDSL(func() {
				Payload(func() {
					Attribute("name", String)
					Attribute("secret", String, func() {
						omitFromJSON()
						Default("x")
					})
					Required("secret")
				})
			}),
			want: `attribute "secret" in the HTTP request body is optional, but its JSON name "-" omits it from JSON`,
		},
		{
			name: "json name metadata",
			dsl: omittedFieldDSL(func() {
				Payload(func() {
					Attribute("name", String)
					Attribute("secret", String, func() {
						Meta("struct:tag:json:name", "-")
					})
				})
			}),
			want: `attribute "secret" in the HTTP request body is optional, but its JSON name "-" omits it from JSON`,
		},
		{
			name: "optional suffixed attribute",
			dsl: omittedFieldDSL(func() {
				Payload(func() {
					Attribute("name", String)
					Attribute("secret:s", String, omitFromJSON)
				})
			}),
			want: `attribute "secret:s" in the HTTP request body is optional, but its JSON name "-" omits it from JSON`,
		},
		{
			name: "error body",
			dsl: func() {
				problem := Type("Problem", func() {
					ErrorName("name", String)
					Attribute("detail", String, omitFromJSON)
					Required("name")
				})
				Service("svc", func() {
					Method("m", func() {
						Error("bad", problem)
						HTTP(func() {
							POST("/m")
							Response("bad", StatusBadRequest)
						})
					})
				})
			},
			want: `attribute "detail" of type "Problem" in the HTTP "bad" error response body is optional, but its JSON name "-" omits it from JSON`,
		},
		{
			name: "json-rpc",
			dsl: func() {
				Service("svc", func() {
					JSONRPC(func() {
						POST("/rpc")
					})
					Method("m", func() {
						Payload(func() {
							Attribute("name", String)
							Attribute("secret", String, omitFromJSON)
						})
						JSONRPC(func() {})
					})
				})
			},
			want: `attribute "secret" in the JSON-RPC request body is optional, but its JSON name "-" omits it from JSON`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, c.dsl)
			got := stripValidationLocations(err.Error())
			if !strings.Contains(got, c.want) {
				t.Errorf("got error %q\nwant it to contain %q", got, c.want)
			}
			if !strings.Contains(got, `remove the "-" JSON tag or leave the attribute out of the body with an explicit Body`) {
				t.Errorf("got error %q\nwant it to say how to fix the design", got)
			}
		})
	}
}

// TestHTTPBodyOmittedFieldAccepted checks that a JSON name "-" is accepted on
// required body attributes, including one required by its "n:m" key, and on
// attributes that no HTTP or JSON-RPC body holds: an attribute mapped to a
// header, an attribute left out of an explicit body, a gRPC field and the
// field of a type that no body uses.
func TestHTTPBodyOmittedFieldAccepted(t *testing.T) {
	cases := []struct {
		name string
		dsl  func()
	}{
		{
			name: "mapped to a header",
			dsl: omittedFieldDSL(func() {
				Payload(func() {
					Attribute("name", String)
					Attribute("secret", String, omitFromJSON)
				})
			}, func() {
				Header("secret")
			}),
		},
		{
			name: "left out of an explicit body",
			dsl: omittedFieldDSL(func() {
				Payload(func() {
					Attribute("name", String)
					Attribute("secret", String, omitFromJSON)
				})
			}, func() {
				Body(func() {
					Attribute("name")
				})
			}),
		},
		{
			name: "required attribute",
			dsl: omittedFieldDSL(func() {
				Payload(func() {
					Attribute("name", String)
					Attribute("secret", String, omitFromJSON)
					Required("secret")
				})
			}),
		},
		{
			name: "required suffixed attribute",
			dsl: omittedFieldDSL(func() {
				Payload(func() {
					Attribute("name", String)
					Attribute("secret:s", String, omitFromJSON)
					Required("secret:s")
				})
			}),
		},
		{
			name: "grpc",
			dsl: func() {
				Service("svc", func() {
					Method("m", func() {
						Payload(func() {
							Field(1, "name", String)
							Field(2, "secret", String, omitFromJSON)
						})
						GRPC(func() {})
					})
				})
			},
		},
		{
			name: "type without body",
			dsl: func() {
				Type("T", func() {
					Attribute("secret", String, omitFromJSON)
				})
				omittedFieldDSL(func() {
					Payload(String)
				})()
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.NotNil(t, expr.RunDSL(t, c.dsl))
		})
	}
}

// omittedFieldDSL returns a design with a method svc.m whose DSL is method
// and whose HTTP endpoint is POST /m, followed by the DSL of each of http.
func omittedFieldDSL(method func(), http ...func()) func() {
	return func() {
		Service("svc", func() {
			Method("m", func() {
				method()
				HTTP(func() {
					POST("/m")
					for _, fn := range http {
						fn()
					}
				})
			})
		})
	}
}

// omitFromJSON omits the attribute from JSON.
func omitFromJSON() {
	Meta("struct:tag:json", "-")
}
