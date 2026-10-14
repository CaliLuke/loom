package codegen

import (
	"fmt"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/stretchr/testify/require"
)

func TestCustomResponseRepresentationCompile(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		Service("uuidresponse", func() {
			Method("read", func() {
				Result(func() {
					Attribute("id", String, func() {
						Meta("struct:field:type", "uuid.UUID", "github.com/google/uuid")
					})
				})
				HTTP(func() {
					GET("/read")
					Response(StatusOK, func() {
						Header("id:X-ID")
					})
				})
			})
		})
	})
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/uuidresponse", root)
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

func TestCustomResponseRepresentationOwner(t *testing.T) {
	for _, location := range []string{"header", "cookie"} {
		for _, shape := range []string{"inline", "named", "projected", "error"} {
			for _, required := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/required=%t", location, shape, required), func(t *testing.T) {
					root := RunHTTPDSL(t, func() {
						fields := func() {
							Attribute("id", String, func() {
								Meta("struct:field:type", "uuid.UUID", "github.com/google/uuid")
							})
							if required {
								Required("id")
							}
						}
						var value any = fields
						if shape == "named" || shape == "error" {
							value = Type("Value", fields)
						}
						if shape == "projected" {
							value = ResultType("application/vnd.custom", func() {
								fields()
								View("default", func() {
									Attribute("id")
								})
							})
						}
						Service("customresponse", func() {
							Method("read", func() {
								if shape == "error" {
									Error("invalid", value)
								} else {
									Result(value)
								}
								HTTP(func() {
									GET("/read")
									mapping := func() {
										if location == "header" {
											Header("id:X-ID")
										} else {
											Cookie("id:ID")
										}
									}
									if shape == "error" {
										Response("invalid", StatusBadRequest, mapping)
									} else {
										Response(StatusOK, mapping)
									}
								})
							})
						})
					})
					services := CreateHTTPServices(root)
					functions := renderTransportFuncs(t, append(ServerFiles("", services), ClientFiles("", services)...))
					encoder := "EncodeReadResponse"
					if shape == "error" {
						encoder = "EncodeReadError"
					}
					require.Contains(t, functions["DecodeReadResponse"], "UnmarshalText", "response must share custom text decoder")
					require.Contains(t, functions[encoder], "fmt.Sprint(", "custom String output must use its text representation")
				})
			}
		}
	}
}
