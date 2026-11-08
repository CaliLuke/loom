package expr_test

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestHTTPErrorMediaIndependentSuccessFraming(t *testing.T) {
	for _, framing := range []string{"raw", "file", "sse", "ordinary"} {
		t.Run(framing, func(t *testing.T) {
			root := expr.RunDSL(t, func() {
				carrier := Type("Failure", func() {
					OneOf("body", func() {
						Attribute("json", String)
						Attribute("binary", Bytes)
					})
					Required("body")
				})
				Service("media", func() {
					Method("read", func() {
						if framing == "sse" {
							StreamingResult(String)
						}
						Error("failed", carrier)
						HTTP(func() {
							GET("/")
							switch framing {
							case "raw":
								SkipResponseBodyEncodeDecode()
							case "file":
								FileResponse()
							case "sse":
								ServerSentEvents()
							}
							Response(StatusOK)
							Response("failed", StatusBadRequest, func() {
								Body("body")
								MediaType("application/json", "json")
								MediaType("application/octet-stream", "binary")
							})
						})
					})
				})
			})
			e := root.API.HTTP.Service("media").Endpoint("read")
			require.Len(t, e.HTTPErrors[0].Response.MediaTypes, 2)
			require.NotNil(t, expr.AsUnion(e.HTTPErrors[0].Response.Body.Type))
		})
	}
}

func TestHTTPMediaSuccessAndRPCFramingRemainRejected(t *testing.T) {
	for _, framing := range []string{"raw", "file", "sse", "rpc"} {
		t.Run(framing, func(t *testing.T) {
			design := func() {
				carrier := Type("Carrier", func() {
					OneOf("body", func() {
						Attribute("json", String)
						Attribute("binary", Bytes)
					})
					Required("body")
				})
				Service("media", func() {
					if framing == "rpc" {
						JSONRPC(func() {
							POST("/rpc")
						})
					}
					Method("read", func() {
						Result(carrier)
						if framing == "sse" {
							StreamingResult(String)
						}
						if framing == "rpc" {
							Error("failed", carrier)
							JSONRPC(func() {
								Response("failed", -32603, func() {
									Body("body")
									MediaType("application/json", "json")
									MediaType("application/octet-stream", "binary")
								})
							})
						} else {
							HTTP(func() {
								GET("/")
								switch framing {
								case "raw":
									SkipResponseBodyEncodeDecode()
								case "file":
									FileResponse()
								case "sse":
									ServerSentEvents()
								}
								Response(StatusOK, func() {
									Body("body")
									MediaType("application/json", "json")
									MediaType("application/octet-stream", "binary")
								})
							})
						}
					})
				})
			}
			require.ErrorContains(t, expr.RunInvalidDSL(t, design), "MediaType requires an ordinary typed HTTP body")
		})
	}
}
