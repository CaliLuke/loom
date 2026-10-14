package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestRequiredReconstructionSourceControls(t *testing.T) {
	for _, kind := range []string{"basic", "jwt", "jsonrpc id", "response tag", "fixed view", "dynamic view", "raw request", "mapped default"} {
		t.Run(kind, func(t *testing.T) {
			expr.RunDSL(t, func() {
				basic := BasicAuthSecurity("basic")
				jwt := JWTSecurity("jwt")
				viewed := ResultType("application/vnd.coverage", func() {
					Attributes(func() {
						Attribute("sent", String)
						Attribute("hidden", String)
						Required("sent", "hidden")
					})
					View("default", func() {
						Attribute("sent")
					})
					View("full", func() {
						Attribute("sent")
						Attribute("hidden")
					})
				})
				Service("coverage", func() {
					if kind == "jsonrpc id" {
						JSONRPC(func() {
							POST("/coverage")
						})
					}
					Method("send", func() {
						if kind == "basic" {
							Security(basic)
						}
						if kind == "jwt" {
							Security(jwt)
						}
						Payload(func() {
							Attribute("sent", String)
							switch kind {
							case "basic":
								Username("user", String)
								Password("password", String)
								Required("user", "password")
							case "jwt":
								Token("token", String)
								Required("token")
							case "jsonrpc id":
								ID("id", String)
								Required("id")
							case "raw request", "mapped default":
								Attribute("mapped", String, func() {
									Default("fallback")
								})
								Required("mapped")
							}
						})
						switch kind {
						case "response tag":
							Result(func() {
								Attribute("kind", String)
								Attribute("sent", String)
								Required("kind", "sent")
							})
						case "fixed view":
							Result(viewed, func() {
								View("default")
							})
						case "dynamic view":
							Result(viewed)
						}
						mapping := func() {
							if kind != "jsonrpc id" {
								POST("/coverage")
							}
							if kind == "raw request" {
								Header("mapped:x-mapped")
								Header("sent:x-sent")
								SkipRequestBodyEncodeDecode()
							} else {
								Body("sent")
							}
							if kind == "mapped default" {
								Header("mapped:x-mapped")
							}
							if kind == "response tag" {
								Response(StatusOK)
								Response(StatusAccepted, func() {
									Tag("kind", "accepted")
									Body("sent")
								})
							}
						}
						if kind == "jsonrpc id" {
							JSONRPC(mapping)
						} else {
							HTTP(mapping)
						}
					})
				})
			})
		})
	}
}

func TestRequiredUnmappedDefaultReadiness(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		Service("coverage", func() {
			Method("send", func() {
				Payload(func() {
					Attribute("sent", String)
					Attribute("lost", String, func() {
						Default("fallback")
					})
					Required("lost")
				})
				HTTP(func() {
					POST("/coverage")
					Body("sent")
				})
			})
		})
	})
	require.ErrorContains(t, err, `required payload attribute "lost" has no reconstruction source`)
}

func TestRequiredErrorMappingControls(t *testing.T) {
	for _, shape := range []string{"body", "header", "cookie", "problem"} {
		t.Run(shape, func(t *testing.T) {
			expr.RunDSL(t, func() {
				failure := Type("Failure", func() {
					ErrorName("kind", String)
					Attribute("detail", String)
					Required("kind", "detail")
				})
				Service("coverage", func() {
					Method("send", func() {
						if shape == "problem" {
							Error("busy")
						} else {
							Error("busy", failure)
						}
						HTTP(func() {
							POST("/coverage")
							Response("busy", StatusBadRequest, func() {
								switch shape {
								case "header":
									Header("kind:X-Kind")
									Body("detail")
								case "cookie":
									Cookie("kind:error-kind")
									Body("detail")
								case "problem":
									Body("detail")
								}
							})
						})
					})
				})
			})
		})
	}
}

func TestRequiredInheritedMappingControls(t *testing.T) {
	for _, location := range []string{"inherited header", "path", "map query", "suffix"} {
		t.Run(location, func(t *testing.T) {
			expr.RunDSL(t, func() {
				Service("coverage", func() {
					if location == "inherited header" {
						HTTP(func() {
							Header("mapped:X-Mapped")
						})
					}
					Method("send", func() {
						Payload(func() {
							Attribute("sent", String)
							switch location {
							case "map query":
								Attribute("mapped", MapOf(String, String))
							case "suffix":
								Attribute("mapped:source", String)
							default:
								Attribute("mapped", String)
							}
							if location == "suffix" {
								Required("mapped:source")
							} else {
								Required("mapped")
							}
						})
						HTTP(func() {
							if location == "path" {
								POST("/coverage/{mapped}")
							} else {
								POST("/coverage")
							}
							if location == "suffix" {
								Body(func() {
									Attribute("mapped:wire")
									Attribute("sent")
								})
							} else {
								Body("sent")
							}
							if location == "map query" {
								MapParams("mapped")
							}
						})
					})
				})
			})
		})
	}
}

func TestRequiredResponseReconstruction(t *testing.T) {
	for _, rpc := range []bool{false, true} {
		for _, firstMissing := range []bool{false, true} {
			t.Run(map[bool]string{false: "http", true: "jsonrpc"}[rpc]+map[bool]string{false: "/later missing", true: "/first missing"}[firstMissing], func(t *testing.T) {
				design := func() {
					Service("coverage", func() {
						if rpc {
							JSONRPC(func() {
								POST("/coverage")
							})
						}
						Method("send", func() {
							Result(func() {
								Attribute("kind", String)
								Attribute("sent", String)
								Attribute("lost", String)
								Required("sent", "lost")
							})
							mapping := func() {
								if !rpc {
									POST("/coverage")
								}
								Response(StatusOK, func() {
									if !firstMissing {
										Body("sent")
									}
								})
								Response(StatusAccepted, func() {
									Tag("kind", "accepted")
									if firstMissing {
										Body("sent")
									}
								})
							}
							if rpc {
								JSONRPC(mapping)
							} else {
								HTTP(mapping)
							}
						})
					})
				}
				if rpc && !firstMissing {
					expr.RunDSL(t, design)
				} else {
					err := expr.RunInvalidDSL(t, design)
					require.ErrorContains(t, err, "required result attribute \"lost\"")
					require.ErrorContains(t, err, "no reconstruction source")
				}
			})
		}
	}
}
