package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestHTTPErrorNameFromStatusAdmission(t *testing.T) {
	for _, test := range []struct{ name, want string }{
		{"shared status", "unique"},
		{"success status", "unique"},
		{"header", "cannot also"},
		{"reserved header", "cannot also"},
		{"cookie", "cannot also"},
		{"body", "cannot also"},
		{"selected name", "cannot also"},
		{"missing name", "requires a custom"},
		{"built in", "requires a custom"},
		{"jsonrpc", "HTTP only"},
		{"enum", "does not satisfy"},
		{"pattern", "does not satisfy"},
		{"length", "does not satisfy"},
		{"inherited collision", "unique"},
		{"missing other", "message"},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, statusErrorNameDesign(test.name))
			require.ErrorContains(t, err, test.want)
		})
	}
}

func TestHTTPErrorNameFromStatusBodies(t *testing.T) {
	for _, mode := range []string{"implicit", "listed", "selected", "empty", "inherited", "constrained"} {
		t.Run(mode, func(t *testing.T) {
			root := expr.RunDSL(t, statusErrorNameDesign(mode))
			response := root.API.HTTP.Services[0].HTTPEndpoints[0].HTTPErrors[0].Response
			require.Equal(t, "kind", response.StatusErrorName)
			if object := expr.AsObject(response.Body.Type); object != nil {
				require.Nil(t, object.Attribute("kind"))
			}
			require.True(t, response.Headers.IsEmpty())
		})
	}
}

func TestHTTPErrorNameFromStatusContext(t *testing.T) {
	for _, mode := range []string{"top", "success", "grpc"} {
		t.Run(mode, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				if mode == "top" {
					ErrorNameFromStatus()
					return
				}
				Service("records", func() {
					Method("read", func() {
						if mode == "grpc" {
							Error("invalid")
							GRPC(func() {
								Response("invalid", func() {
									ErrorNameFromStatus()
								})
							})
							return
						}
						HTTP(func() {
							GET("/records")
							Response(StatusOK, func() {
								ErrorNameFromStatus()
							})
						})
					})
				})
			})
			require.ErrorContains(t, err, "ErrorNameFromStatus")
		})
	}
}

func statusErrorNameDesign(mode string) func() {
	return func() {
		failure := Type("Failure", func() {
			if mode != "missing name" {
				ErrorName("kind", String, func() {
					if mode == "enum" {
						Enum("different")
					}
					if mode == "pattern" {
						Pattern("^other")
					}
					if mode == "length" {
						MaxLength(2)
					}
					if mode == "constrained" {
						Enum("invalid")
						Pattern("^invalid$")
						MinLength(7)
						MaxLength(7)
					}
				})
				Required("kind")
			}
			Attribute("message", String)
			if mode != "empty" && mode != "selected name" {
				Required("message")
			}
		})
		response := func() {
			Response("invalid", StatusBadRequest, func() {
				ErrorNameFromStatus()
				switch mode {
				case "header":
					Header("kind:loom-error")
				case "reserved header":
					Header("message:Loom-Error")
				case "cookie":
					Cookie("kind:identity")
				case "body":
					Body(func() {
						Attribute("kind")
						Attribute("message")
					})
				case "selected name":
					Body("kind")
				case "listed":
					Body(func() {
						Attribute("message")
					})
				case "selected":
					Body("message")
				case "empty", "missing other":
					Body(Empty)
				}
			})
		}
		Service("records", func() {
			inherited := mode == "inherited" || mode == "inherited collision"
			if inherited {
				Error("invalid", failure)
				HTTP(response)
			}
			if mode == "jsonrpc" {
				JSONRPC(func() {
					POST("/records")
				})
			}
			Method("read", func() {
				if !inherited {
					if mode == "built in" {
						Error("invalid")
					} else {
						Error("invalid", failure)
					}
				}
				if mode == "shared status" || mode == "inherited collision" {
					Error("other", failure)
				}
				mapping := func() {
					if mode != "jsonrpc" {
						GET("/records")
					}
					if !inherited {
						response()
					}
					if mode == "shared status" || mode == "inherited collision" {
						Response("other", StatusBadRequest)
					}
					if mode == "success status" {
						Response(StatusBadRequest)
					}
				}
				if mode == "jsonrpc" {
					JSONRPC(mapping)
				} else {
					HTTP(mapping)
				}
			})
		})
	}
}
