package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
)

// TestEndpointMessageNames checks that a message generated for a method, such
// as its request, response, streaming request, stream envelope or error
// message, does not take the name of a design type that a message of the
// service refers to when the two messages differ, and that it keeps the name
// when the design type is not a message of the service or has the same
// message, so that the designs that did not collide keep their output.
func TestEndpointMessageNames(t *testing.T) {
	cases := []struct {
		Name        string
		DSL         func()
		Contains    []string
		NotContains []string
	}{
		{"request field", endpointMessageFieldDSL("MRequest", Payload), []string{
			"rpc M (MRequest2) returns (MResponse);",
			"message MRequest2 {\n\tMRequest t = 1;\n\toptional string other = 2;\n}",
			"message MRequest {\n\toptional string name = 1;\n}",
		}, []string{"MRequest t = 1;\n\toptional string other = 2;\n}\n\nmessage MResponse"}},
		{"response field", endpointMessageFieldDSL("MResponse", Result), []string{
			"rpc M (MRequest) returns (MResponse2);",
			"message MResponse2 {\n\tMResponse t = 1;\n\toptional string other = 2;\n}",
			"message MResponse {\n\toptional string name = 1;\n}",
		}, nil},
		{"streaming request field", endpointMessageFieldDSL("MStreamingRequest", StreamingPayload), []string{
			"rpc M (stream MStreamingRequest2) returns (MResponse);",
			"message MStreamingRequest2 {\n\tMStreamingRequest t = 1;\n\toptional string other = 2;\n}",
		}, nil},
		{"error field", endpointMessageErrorDSL, []string{
			"message MFailError2 {\n\toptional string msg = 1;\n\tMFailError detail = 2;\n}",
			"message MFailError {\n\toptional string name = 1;\n}",
		}, nil},
		{"wrapped result", endpointMessageWrapperDSL, []string{
			"rpc M (MRequest) returns (MResponse2);",
			"message MResponse2 {\n\trepeated string field = 1;\n}",
			"message NRequest {\n\tMResponse t = 1;\n}",
			"message MResponse {\n\toptional string name = 1;\n}",
		}, nil},
		{"numbered design type", endpointMessageNumberedDSL, []string{
			"rpc M (MRequest3) returns (MResponse);",
			"message MRequest3 {\n\tMRequest t = 1;\n\tMRequest2 u = 2;\n}",
			"message MRequest2 {\n\toptional sint64 count = 1;\n}",
			"message MResponse {\n\tMRequest t = 1;\n}",
		}, nil},
		{"stream envelope", endpointMessageEnvelopeDSL, []string{
			"rpc M (stream MStreamingRequest2) returns (MResponse);",
			"message MStreamingRequest2 {\n\toneof body {\n\t\tMRequest initial_payload = 1;\n\t\tMStreamItem stream_item = 2;\n\t}\n}",
			"message MStreamItem {\n\tMStreamingRequest t = 1;\n}",
			"message MStreamingRequest {\n\toptional string name = 1;\n}",
		}, nil},
		{"direct payload", endpointMessageDirectDSL, []string{
			"rpc M (MRequest) returns (MResponse);",
			"message MRequest {\n\toptional string name = 1;\n}",
		}, []string{"MRequest2"}},
		{"recursive direct payload", endpointMessageRecursiveDSL, []string{
			"rpc M (MRequest) returns (MResponse);",
			"message MRequest {\n\toptional string name = 1;\n\tMRequest child = 2;\n}",
			"message MResponse {\n\toptional string name = 1;\n\tMRequest child = 2;\n}",
		}, []string{"MRequest2"}},
		{"same message", endpointMessageSameShapeDSL, []string{
			"rpc M (MRequest) returns (MResponse);",
			"message MRequest {\n\toptional string a = 1;\n}",
			"message NRequest {\n\tMRequest t = 1;\n}",
		}, []string{"MRequest2"}},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			var code string
			require.NoError(t, generationError(func() {
				code = protoFileCode(t, c.DSL)
			}))
			for _, want := range c.Contains {
				assert.Contains(t, code, want)
			}
			for _, unwanted := range c.NotContains {
				assert.NotContains(t, code, unwanted)
			}
			fpath := codegen.CreateTempFile(t, code)
			assert.NoError(t, protoc(defaultProtocCmd, fpath, nil), "compile proto file %q", fpath)
		})
	}
}

// TestGeneratedEndpointMessageNames compiles the modules generated for
// designs whose types are named like generated request, response, streaming
// request, stream envelope and error messages, and round-trips the request
// and response of the method whose payload and result hold such types.
func TestGeneratedEndpointMessageNames(t *testing.T) {
	for name, dsl := range map[string]func(){
		"streaming request": endpointMessageFieldDSL("MStreamingRequest", StreamingPayload),
		"error":             endpointMessageErrorDSL,
		"wrapped result":    endpointMessageWrapperDSL,
		"stream envelope":   endpointMessageEnvelopeDSL,
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			renderGRPCModule(t, dir, "example.com/endpointmessages", RunGRPCDSL(t, dsl), resolveGRPCLoomSource(t))
			runGRPCGoCommand(t, dir, "mod", "tidy")
			runGRPCGoCommand(t, dir, "vet", "./...")
		})
	}
	t.Run("request and response", func(t *testing.T) {
		runGeneratedRoundTrip(t, "example.com/endpointmessages", endpointMessageNumberedDSL, endpointMessageNamesRoundTripHarness)
	})
}

// endpointMessageFieldDSL declares the type tname and passes an object whose
// t field holds it to the method m with dslFunc, such as Payload.
func endpointMessageFieldDSL(tname string, dslFunc func(any, ...any)) func() {
	return func() {
		ut := Type(tname, func() {
			Field(1, "name", String)
		})
		Service("svc", func() {
			Method("m", func() {
				dslFunc(func() {
					Field(1, "t", ut)
					Field(2, "other", String)
				})
				GRPC(func() {})
			})
		})
	}
}

// endpointMessageErrorDSL declares the error Fail of the method m, whose
// detail field holds the type MFailError, named like the message of the
// error.
func endpointMessageErrorDSL() {
	detail := Type("MFailError", func() {
		Field(1, "name", String)
	})
	fail := Type("Fail", func() {
		Field(1, "msg", String)
		Field(2, "detail", detail)
	})
	Service("svc", func() {
		Method("m", func() {
			Payload(String)
			Error("fail", fail)
			GRPC(func() {
				Response("fail", CodeNotFound)
			})
		})
	})
}

// endpointMessageWrapperDSL declares the type MResponse, used as the payload
// of m and as a field of the payload of n, while the array result of m is
// wrapped in a message named like the response message of m.
func endpointMessageWrapperDSL() {
	ut := Type("MResponse", func() {
		Field(1, "name", String)
	})
	Service("svc", func() {
		Method("m", func() {
			Payload(ut)
			Result(ArrayOf(String))
			GRPC(func() {})
		})
		Method("n", func() {
			Payload(func() {
				Field(1, "t", ut)
			})
			GRPC(func() {})
		})
	})
}

// endpointMessageNumberedDSL declares the types MRequest and MRequest2 held
// by the payload of m, and MRequest held by its result.
func endpointMessageNumberedDSL() {
	req := Type("MRequest", func() {
		Field(1, "name", String)
	})
	req2 := Type("MRequest2", func() {
		Field(1, "count", Int)
	})
	Service("svc", func() {
		Method("m", func() {
			Payload(func() {
				Field(1, "t", req)
				Field(2, "u", req2)
			})
			Result(func() {
				Field(1, "t", req)
			})
			GRPC(func() {})
		})
	})
}

// endpointMessageEnvelopeDSL declares the type MStreamingRequest held by the
// streaming payload of m, whose stream envelope takes the name of the
// streaming request message.
func endpointMessageEnvelopeDSL() {
	ut := Type("MStreamingRequest", func() {
		Field(1, "name", String)
	})
	Service("svc", func() {
		Method("m", func() {
			Payload(func() {
				Field(1, "id", String)
			})
			StreamingPayload(func() {
				Field(1, "t", ut)
			})
			Result(String)
			GRPC(func() {})
		})
	})
}

// endpointMessageDirectDSL uses the type MRequest as the payload and result
// of m, whose request message has the fields of the type.
func endpointMessageDirectDSL() {
	ut := Type("MRequest", func() {
		Field(1, "name", String)
	})
	Service("svc", func() {
		Method("m", func() {
			Payload(ut)
			Result(ut)
			GRPC(func() {})
		})
	})
}

// endpointMessageRecursiveDSL uses the recursive type MRequest as the
// payload and result of m, whose request message is the message of the type.
func endpointMessageRecursiveDSL() {
	ut := Type("MRequest", func() {
		Field(1, "name", String)
		Field(2, "child", "MRequest")
	})
	Service("svc", func() {
		Method("m", func() {
			Payload(ut)
			Result(ut)
			GRPC(func() {})
		})
	})
}

// endpointMessageSameShapeDSL declares the type MRequest, a field of the
// payload of n, with the fields of the payload of m.
func endpointMessageSameShapeDSL() {
	ut := Type("MRequest", func() {
		Field(1, "a", String)
	})
	Service("svc", func() {
		Method("m", func() {
			Payload(func() {
				Field(1, "a", String)
			})
			GRPC(func() {})
		})
		Method("n", func() {
			Payload(func() {
				Field(1, "t", ut)
			})
			GRPC(func() {})
		})
	})
}

const endpointMessageNamesRoundTripHarness = `package roundtrip

import (
	"testing"

	"github.com/stretchr/testify/require"

	"%[1]s/gen/grpc/svc/client"
	pb "%[1]s/gen/grpc/svc/pb"
	"%[1]s/gen/grpc/svc/server"
	svc "%[1]s/gen/svc"
)

func TestRoundTrip(t *testing.T) {
	name, count := "n", 3
	payload := &svc.MPayload{T: &svc.MRequest{Name: &name}, U: &svc.MRequest2{Count: &count}}
	var message *pb.MRequest3
	require.NotPanics(t, func() {
		message = client.NewProtoMRequest3(payload)
	})
	require.Equal(t, "n", message.GetT().GetName())
	require.Equal(t, int64(3), message.GetU().GetCount())
	require.Equal(t, payload, server.NewMPayload(message))

	result := &svc.MResult{T: &svc.MRequest{Name: &name}}
	require.Equal(t, result, client.NewMResult(server.NewProtoMResponse(result)))
	require.Equal(t, &svc.MResult{}, client.NewMResult(server.NewProtoMResponse(&svc.MResult{})))
}
`
