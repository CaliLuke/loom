package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestSharedMessageConverters checks designs where different service types
// map to the same protocol buffer message in the same converter direction.
// Each service type gets its own converter, so the generated modules compile
// and vet.
func TestSharedMessageConverters(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"customized payloads", customizedPayloadsDSL},
		{"redundant customized payload next to an unmodified payload", redundantCustomizedPayloadDSL},
		{"session auth payloads", sessionPayloadsDSL},
		{"customized result next to an unmodified result", customizedResultNextToUnmodifiedDSL},
		{"customized results", customizedResultsDSL},
		{"customized streaming results", customizedStreamingResultsDSL},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			code := protoFileCode(t, c.DSL)
			fpath := codegen.CreateTempFile(t, code)
			require.NoError(t, protoc(defaultProtocCmd, fpath, nil), "compile proto file %q", fpath)
			dir := t.TempDir()
			renderGRPCModule(t, dir, "example.com/sharedmessage", RunGRPCDSL(t, c.DSL), resolveGRPCLoomSource(t))
			runGRPCGoCommand(t, dir, "mod", "tidy")
			runGRPCGoCommand(t, dir, "vet", "./...")
		})
	}
}

// menuResultType declares the result type Menu with the struct:name:proto
// name MenuProto and the optional fields x and y.
func menuResultType() expr.UserType {
	return ResultType("application/vnd.menu", "Menu", func() {
		Meta("struct:name:proto", "MenuProto")
		Attributes(xyFields)
	})
}

func customizedPayloadsDSL() {
	rt := menuResultType()
	Service("svc", func() {
		Method("m1", func() {
			Payload(rt, func() {
				Required("x")
			})
			Result(String)
			GRPC(func() {})
		})
		Method("m2", func() {
			Payload(rt, func() {
				Required("x")
			})
			Result(String)
			GRPC(func() {})
		})
	})
}

func redundantCustomizedPayloadDSL() {
	a := metaType("A", "AProto", func() {
		xyFields()
		Required("x")
	})
	Service("svc", func() {
		Method("m1", func() {
			Payload(a)
			Result(String)
			GRPC(func() {})
		})
		Method("m2", func() {
			Payload(a, func() {
				Required("x")
			})
			Result(String)
			GRPC(func() {})
		})
	})
}

func sessionPayloadsDSL() {
	jwt := JWTSecurity("jwt")
	session := SessionAuth("session", func() {
		BearerTransport(jwt, "auth")
	})
	a := metaType("A", "AProto", xyFields)
	Service("svc", func() {
		SessionSecurity(session)
		Method("m1", func() {
			Payload(a)
			Result(String)
			GRPC(func() {})
		})
		Method("m2", func() {
			Payload(a)
			Result(String)
			GRPC(func() {})
		})
	})
}

func customizedResultNextToUnmodifiedDSL() {
	a := metaType("A", "AProto", func() {
		xyFields()
		Required("x")
	})
	Service("svc", func() {
		Method("m1", func() {
			Result(a, func() {
				Required("x")
			})
			GRPC(func() {})
		})
		Method("m2", func() {
			Result(a)
			GRPC(func() {})
		})
	})
}

func customizedResultsDSL() {
	a := metaType("A", "AProto", xyFields)
	Service("svc", func() {
		Method("m1", func() {
			Result(a, func() {
				Required("x")
			})
			GRPC(func() {})
		})
		Method("m2", func() {
			Result(a, func() {
				Required("x")
			})
			GRPC(func() {})
		})
	})
}

func customizedStreamingResultsDSL() {
	a := metaType("A", "AProto", func() {
		xyFields()
		Required("x")
	})
	Service("svc", func() {
		Method("m1", func() {
			Result(a, func() {
				Required("x")
			})
			GRPC(func() {})
		})
		Method("m2", func() {
			StreamingResult(a, func() {
				Required("x")
			})
			GRPC(func() {})
		})
		Method("m3", func() {
			StreamingResult(a)
			GRPC(func() {})
		})
		Method("m4", func() {
			Result(a)
			GRPC(func() {})
		})
	})
}

// TestSharedMessageShapes checks that generation fails with an error naming
// the methods when two messages with different fields take the same
// struct:name:proto name, since only one of them can be generated, and that
// messages with the same name and fields are shared.
func TestSharedMessageShapes(t *testing.T) {
	cases := []struct {
		Name     string
		DSL      func()
		Expected string
	}{
		{"customized payload and result", func() {
			a := metaType("A", "AProto", xyFields)
			Service("svc", func() {
				Method("m1", func() {
					Payload(a, func() {
						Required("y")
					})
					Result(a, func() {
						Required("x")
					})
					GRPC(func() {})
				})
			})
		}, `method "m1" of service "svc" maps two messages with different fields to protocol buffer message "AProto"`},
		{"request metadata and response trailers", func() {
			a := metaType("A", "AProto", xyFields)
			Service("svc", func() {
				Method("m1", func() {
					Payload(a)
					Result(a)
					GRPC(func() {
						Metadata(func() {
							Attribute("x")
						})
						Response(func() {
							Trailers(func() {
								Attribute("y")
							})
						})
					})
				})
			})
		}, `method "m1" of service "svc" maps two messages with different fields to protocol buffer message "AProto"`},
		{"customized payloads of two methods", func() {
			a := metaType("A", "AProto", xyFields)
			Service("svc", func() {
				Method("m1", func() {
					Payload(a)
					Result(String)
					GRPC(func() {})
				})
				Method("m2", func() {
					Payload(a, func() {
						Required("x")
					})
					Result(String)
					GRPC(func() {})
				})
			})
		}, `methods "m1" and "m2" of service "svc" both map to protocol buffer message "AProto" with different fields`},
		{"payload metadata in one method", func() {
			a := metaType("A", "AProto", xyFields)
			Service("svc", func() {
				Method("m1", func() {
					Payload(a)
					Result(String)
					GRPC(func() {
						Metadata(func() {
							Attribute("x")
						})
					})
				})
				Method("m2", func() {
					Payload(a)
					Result(String)
					GRPC(func() {})
				})
			})
		}, `methods "m1" and "m2" of service "svc" both map to protocol buffer message "AProto" with different fields`},
		{"name of a generated message", func() {
			a := metaType("A", "M2Request", xyFields)
			Service("svc", func() {
				Method("m1", func() {
					Payload(a)
					Result(String)
					GRPC(func() {})
				})
				Method("m2", func() {
					Payload(Int)
					Result(String)
					GRPC(func() {})
				})
			})
		}, `methods "m1" and "m2" of service "svc" both map to protocol buffer message "M2Request" with different fields`},
		{"same fields", customizedPayloadsDSL, ""},
		{"same fields in payload and result", func() {
			a := metaType("A", "AProto", xyFields)
			Service("svc", func() {
				Method("m1", func() {
					Payload(a, func() {
						Required("x")
					})
					Result(a, func() {
						Required("x")
					})
					GRPC(func() {})
				})
			})
		}, ""},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunGRPCDSL(t, c.DSL)
			err := generationError(func() { ProtoFiles("", CreateGRPCServices(root)) })
			if c.Expected == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), c.Expected)
		})
	}
}
