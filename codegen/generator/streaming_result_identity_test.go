package generator

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

// TestStreamingResultIdentity checks mixed-result accessors through generated
// typed clients and an HTTP server wrapper. JSON-RPC server wrappers are a
// separate protocol-shape contract.
func TestStreamingResultIdentity(t *testing.T) {
	for _, name := range []string{"shared", "optional", "server", "external"} {
		t.Run(name, func(t *testing.T) {
			harness, err := os.ReadFile("testdata/streaming_result_identity_" + name + ".go.txt")
			require.NoError(t, err)
			runDesignHarness(t, "example.com/streamidentity", streamingResultIdentityDSL(name), string(harness))
		})
	}
}

func streamingResultIdentityDSL(shape string) func() {
	return func() {
		dsl.API("streamidentity", func() {
			dsl.JSONRPC(func() {})
		})
		summary := dsl.Type("Summary", func() {
			field := "value"
			if shape == "optional" {
				field = "summary"
			}
			dsl.Attribute(field, dsl.String)
			dsl.Required(field)
		})
		chunk := dsl.Type("Chunk", func() {
			if shape == "external" {
				dsl.Meta("struct:pkg:path", "chunks")
			}
			dsl.Attribute("value", dsl.String)
			if shape != "optional" {
				dsl.Required("value")
			}
		})
		other := dsl.Type("Other", func() {
			dsl.Attribute("value", dsl.String)
			dsl.Required("value")
		})
		audit := dsl.Interceptor("audit", func() {
			dsl.ReadStreamingResult(func() {
				dsl.Attribute("value")
			})
			dsl.WriteStreamingResult(func() {
				dsl.Attribute("value")
			})
		})
		transports := []bool{false, true}
		if shape == "server" || shape == "external" {
			transports = []bool{false}
		}
		for _, rpc := range transports {
			service := "events"
			if rpc {
				service = "rpcevents"
			}
			dsl.Service(service, func() {
				if rpc {
					dsl.JSONRPC(func() {
						dsl.POST("/rpc")
					})
				}
				dsl.Method("watch", func() {
					dsl.Payload(func() {
						dsl.Attribute("id", dsl.String)
						if rpc {
							dsl.ID("id")
						}
					})
					if shape != "ordinary" {
						dsl.Result(summary)
					}
					dsl.StreamingResult(chunk)
					dsl.ClientInterceptor(audit)
					if shape == "server" {
						dsl.ServerInterceptor(audit)
					}
					if rpc {
						dsl.JSONRPC(func() {
							dsl.ServerSentEvents()
						})
					} else {
						dsl.HTTP(func() {
							dsl.POST("/watch")
							dsl.ServerSentEvents()
						})
					}
				})
				if shape == "external" {
					dsl.Method("follow", func() {
						dsl.StreamingResult(other)
						dsl.ClientInterceptor(audit)
						dsl.HTTP(func() {
							dsl.GET("/follow")
							dsl.ServerSentEvents()
						})
					})
				}
			})
		}
	}
}
