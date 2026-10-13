package generator

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

func TestMixedClientInterception(t *testing.T) {
	harness, err := os.ReadFile("testdata/mixed_client_interception.go.txt")
	require.NoError(t, err)
	for _, mode := range []string{"mixed", "collision"} {
		t.Run(mode, func(t *testing.T) {
			runDesignHarness(t, "example.com/dispatch", mixedClientInterceptionDSL(mode), string(harness))
		})
	}
}

func mixedClientInterceptionDSL(mode string) func() {
	return func() {
		dsl.API("dispatch", func() {
			dsl.JSONRPC(func() {})
		})
		summary := dsl.Type("Summary", func() {
			dsl.Attribute("value", dsl.String)
			dsl.Required("value")
		})
		chunk := dsl.Type("Chunk", func() {
			dsl.Attribute("value", dsl.String)
			dsl.Required("value")
		})
		clash := dsl.Type("WrapCombinedStreamClientEndpoint", func() {
			dsl.Attribute("note", dsl.String)
		})
		request := dsl.Type("Request", func() {
			dsl.ID("id", dsl.String)
			dsl.Attribute("value", dsl.String)
			dsl.Required("value")
			if mode == "collision" {
				dsl.Attribute("collision", clash)
			}
		})
		audit := dsl.Interceptor("audit", mixedClientMessageAccess)
		both := dsl.Interceptor("both", func() {
			mixedClientPayloadAccess()
			mixedClientMessageAccess()
		})
		payload := dsl.Interceptor("payload", mixedClientPayloadAccess)
		generic := dsl.Interceptor("generic", func() {})
		for _, rpc := range []bool{false, true} {
			name := "events"
			if rpc {
				name = "rpcevents"
			}
			dsl.Service(name, func() {
				if rpc {
					dsl.JSONRPC(func() {
						dsl.POST("/rpc")
					})
				}
				methods := []string{"stream_only", "combined", "payload_only", "generic", "ordered", "plain"}
				if mode == "ordinary" {
					methods = []string{"stream_only", "combined", "payload_only", "generic", "plain"}
				}
				for _, method := range methods {
					dsl.Method(method, func() {
						dsl.Payload(request)
						stream := mode != "ordinary" || method == "stream_only" || method == "combined"
						if mode != "ordinary" || !stream {
							dsl.Result(summary)
						}
						if stream {
							dsl.StreamingResult(chunk)
						}
						switch method {
						case "stream_only":
							dsl.ClientInterceptor(audit)
						case "combined":
							dsl.ClientInterceptor(both)
						case "payload_only":
							dsl.ClientInterceptor(payload)
						case "generic":
							dsl.ClientInterceptor(generic)
						case "ordered":
							dsl.ClientInterceptor(audit, both, payload, generic)
						}
						if rpc {
							dsl.JSONRPC(func() {
								if stream {
									dsl.ServerSentEvents()
								}
							})
						} else {
							dsl.HTTP(func() {
								dsl.POST("/" + method)
								if stream {
									dsl.ServerSentEvents()
								}
							})
						}
					})
				}
			})
		}
	}
}

func mixedClientPayloadAccess() {
	dsl.ReadPayload(func() {
		dsl.Attribute("value")
	})
	dsl.WritePayload(func() {
		dsl.Attribute("value")
	})
}

func mixedClientMessageAccess() {
	dsl.ReadStreamingResult(func() {
		dsl.Attribute("value")
	})
	dsl.WriteStreamingResult(func() {
		dsl.Attribute("value")
	})
}
