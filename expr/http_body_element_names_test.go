package expr_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestHTTPBodyElementNamesRejected checks that the fields of an HTTP or
// JSON-RPC body whose element name suffix, such as "x" in "a:x", gives them a
// JSON name that the body cannot use are rejected, and that the message names
// both attributes and the JSON name.
func TestHTTPBodyElementNamesRejected(t *testing.T) {
	cases := []struct {
		name string
		dsl  func()
		want string
	}{
		{
			name: "two suffixes",
			dsl: httpElementNamesDSL(func() {
				Attribute("a:x", String)
				Attribute("b:x", Int)
			}),
			want: `attributes "a:x" and "b:x" of type "T" both use the JSON name "x" in the HTTP request body`,
		},
		{
			name: "suffix and attribute name",
			dsl: httpElementNamesDSL(func() {
				Attribute("a:x", String)
				Attribute("x", Int)
			}),
			want: `attributes "a:x" and "x" of type "T" both use the JSON name "x" in the HTTP request body`,
		},
		{
			name: "suffix and json tag",
			dsl: httpElementNamesDSL(func() {
				Attribute("a:x", String)
				Attribute("b", Int, func() {
					Meta("struct:tag:json", "x")
				})
			}),
			want: `attributes "a:x" and "b" of type "T" both use the JSON name "x" in the HTTP request body`,
		},
		{
			name: "response body",
			dsl: httpElementNamesDSL(func() {
				Attribute("a:x", String)
				Attribute("b:x", Int)
			}),
			want: `attributes "a:x" and "b:x" of type "T" both use the JSON name "x" in the HTTP response body`,
		},
		{
			name: "empty element name",
			dsl: httpElementNamesDSL(func() {
				Attribute("a:", String)
			}),
			want: `attribute "a:" of type "T" has an empty element name, so the HTTP request body cannot name its JSON field`,
		},
		{
			name: "dash element name",
			dsl: httpElementNamesDSL(func() {
				Attribute("a:-", String)
			}),
			want: `attribute "a:-" of type "T" has the element name "-", which would omit the field from the HTTP request body`,
		},
		{
			name: "invalid element name",
			dsl: httpElementNamesDSL(func() {
				Attribute("a:x,y", String)
			}),
			want: `attribute "a:x,y" of type "T" in the HTTP request body: JSON name "x,y" cannot contain a comma`,
		},
		{
			name: "nested type",
			dsl: func() {
				leaf := Type("Leaf", func() {
					Attribute("a:x", String)
					Attribute("b:x", Int)
				})
				Service("svc", func() {
					Method("m", func() {
						Result(func() {
							Attribute("leaf", leaf)
						})
						HTTP(func() {
							GET("/m")
						})
					})
				})
			},
			want: `attributes "a:x" and "b:x" of type "Leaf" both use the JSON name "x" in the HTTP response body`,
		},
		{
			name: "inline payload",
			dsl: func() {
				Service("svc", func() {
					Method("m", func() {
						Payload(func() {
							Attribute("a:x", String)
							Attribute("b:x", String)
						})
						HTTP(func() {
							POST("/m")
						})
					})
				})
			},
			want: `attributes "a:x" and "b:x" both use the JSON name "x" in the HTTP request body`,
		},
		{
			name: "explicit body",
			dsl: func() {
				leaf := Type("Leaf", func() {
					Attribute("a:x", String)
					Attribute("b:x", Int)
				})
				Service("svc", func() {
					Method("m", func() {
						Payload(func() {
							Attribute("leaf", leaf)
						})
						HTTP(func() {
							POST("/m")
							Body("leaf")
						})
					})
				})
			},
			want: `attributes "a:x" and "b:x" of type "Leaf" both use the JSON name "x" in the HTTP request body`,
		},
		{
			name: "error body",
			dsl: func() {
				failure := Type("Failure", func() {
					Attribute("a:x", String)
					Attribute("b:x", Int)
				})
				Service("svc", func() {
					Method("m", func() {
						Error("bad", failure)
						HTTP(func() {
							GET("/m")
							Response("bad", StatusBadRequest)
						})
					})
				})
			},
			want: `attributes "a:x" and "b:x" of type "Failure" both use the JSON name "x" in the HTTP "bad" error response body`,
		},
		{
			name: "streaming payload",
			dsl: func() {
				message := Type("Message", func() {
					Attribute("a:x", String)
					Attribute("b:x", Int)
				})
				Service("svc", func() {
					Method("m", func() {
						StreamingPayload(message)
						HTTP(func() {
							GET("/m")
						})
					})
				})
			},
			want: `attributes "a:x" and "b:x" of type "Message" both use the JSON name "x" in the HTTP streaming body`,
		},
		{
			name: "untagged union branch",
			dsl: func() {
				first := Type("First", func() {
					Attribute("a:x", String)
					Attribute("b:x", Int)
				})
				second := Type("Second", func() {
					Attribute("c", String)
					Required("c")
				})
				Service("svc", func() {
					Method("m", func() {
						Payload(OneOf(first, second), func() {
							Untagged()
						})
						HTTP(func() {
							POST("/m")
						})
					})
				})
			},
			want: `attributes "a:x" and "b:x" of type "First" both use the JSON name "x" in the HTTP request body`,
		},
		{
			name: "json-rpc",
			dsl: func() {
				payload := Type("T", func() {
					Attribute("a:x", String)
					Attribute("b:x", Int)
				})
				Service("svc", func() {
					JSONRPC(func() {
						POST("/rpc")
					})
					Method("m", func() {
						Payload(payload)
						Result(String)
						JSONRPC(func() {})
					})
				})
			},
			want: `attributes "a:x" and "b:x" of type "T" both use the JSON name "x" in the JSON-RPC request body`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, c.dsl)
			got := stripValidationLocations(err.Error())
			if !strings.Contains(got, c.want) {
				t.Errorf("got error %q\nwant it to contain %q", got, c.want)
			}
		})
	}
}

// TestHTTPBodyElementNamesAccepted checks that element name suffixes that give
// every field of each HTTP and JSON-RPC body its own JSON name are accepted,
// as are suffixes that collide only outside the bodies: in a gRPC service,
// which ignores them, in a type that no body uses, and with an attribute that
// the endpoint maps to a header.
func TestHTTPBodyElementNamesAccepted(t *testing.T) {
	cases := []struct {
		name string
		dsl  func()
	}{
		{
			name: "distinct element names",
			dsl: httpElementNamesDSL(func() {
				Attribute("a:x", String)
				Attribute("b:y", Int)
				Attribute("c", Int)
			}),
		},
		{
			name: "element name of its own attribute",
			dsl: httpElementNamesDSL(func() {
				Attribute("a:a", String)
				Attribute("b", Int)
			}),
		},
		{
			name: "json tag overrides the element name",
			dsl: httpElementNamesDSL(func() {
				Attribute("a:x", String, func() {
					Meta("struct:tag:json", "z")
				})
				Attribute("b:x", Int)
			}),
		},
		{
			name: "colliding attribute mapped to a header",
			dsl: func() {
				Service("svc", func() {
					Method("m", func() {
						Payload(func() {
							Attribute("a:x", String)
							Attribute("x", String)
						})
						HTTP(func() {
							POST("/m")
							Header("x")
						})
					})
				})
			},
		},
		{
			name: "type without body",
			dsl: func() {
				Type("T", func() {
					Attribute("a:x", String)
					Attribute("b:x", Int)
				})
				Service("svc", func() {
					Method("m", func() {
						Payload(String)
						HTTP(func() {
							POST("/m")
						})
					})
				})
			},
		},
		{
			name: "grpc",
			dsl: func() {
				payload := Type("T", func() {
					Field(1, "a:x", String)
					Field(2, "b:x", Int)
				})
				Service("svc", func() {
					Method("m", func() {
						Payload(payload)
						Result(payload)
						GRPC(func() {})
					})
				})
			},
		},
		{
			name: "grpc untagged union branch",
			dsl: func() {
				first := Type("First", func() {
					Field(1, "a:x", String)
					Field(2, "b:x", Int)
				})
				second := Type("Second", func() {
					Field(1, "c", String)
					Required("c")
				})
				Service("svc", func() {
					Method("m", func() {
						Payload(func() {
							Field(1, "u", OneOf(first, second), func() {
								Untagged()
							})
						})
						GRPC(func() {})
					})
				})
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.NotNil(t, expr.RunDSL(t, c.dsl))
		})
	}
}

// httpElementNamesDSL returns a design with a type T of the given attributes
// that an HTTP endpoint uses as its payload and result.
func httpElementNamesDSL(attributes func()) func() {
	return func() {
		payload := Type("T", attributes)
		Service("svc", func() {
			Method("m", func() {
				Payload(payload)
				Result(payload)
				HTTP(func() {
					POST("/m")
				})
			})
		})
	}
}
