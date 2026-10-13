package expr_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

func TestGRPCFieldNumberMessageRoles(t *testing.T) {
	for _, role := range []string{"request", "response", "stream", "stream-response", "array", "map", "error", "recursive"} {
		t.Run(role, func(t *testing.T) {
			for _, tag := range []int{19500, 20000} {
				t.Run(strconv.Itoa(tag), func(t *testing.T) {
					design := func() {
						nested := Type("Nested", func() {
							Field(tag, "value", String)
							if role == "recursive" {
								Field(1, "next", "Nested")
							}
						})
						Service("svc", func() {
							Method("call", func() {
								switch role {
								case "response":
									Result(nested)
								case "stream":
									StreamingPayload(nested)
								case "stream-response":
									StreamingResult(nested)
								case "array":
									Payload(ArrayOf(nested))
								case "map":
									Payload(MapOf(String, nested))
								case "error":
									Error("broken", nested)
								default:
									Payload(nested)
								}
								GRPC(func() {
									if role == "error" {
										Response("broken", CodeInternal)
									}
								})
							})
						})
					}
					expr.SetupTestDSL(t)
					require.True(t, eval.Execute(design, nil), eval.Context.Error())
					err := eval.RunDSL()
					if tag == 19500 {
						require.Error(t, err, "selected emitted field must reject reserved number")
						require.Contains(t, err.Error(), "value")
						require.Contains(t, err.Error(), "19500")
					} else {
						require.NoError(t, err)
					}
				})
			}
		})
	}
}

func TestGRPCFieldNumberSelectionControls(t *testing.T) {
	for _, role := range []string{"request-metadata", "response-header", "response-trailer", "http-only", "unused", "separate-messages", "opaque"} {
		t.Run(role, func(t *testing.T) {
			expr.RunDSL(t, func() {
				tagged := Type("Tagged", func() {
					Field(19500, "value", String)
					if role == "opaque" {
						Meta("struct:field:proto", "google.protobuf.StringValue", "google/protobuf/wrappers.proto")
					}
				})
				Service("svc", func() {
					Method("call", func() {
						switch role {
						case "response-header", "response-trailer":
							Result(tagged)
						case "unused":
							Payload(String)
						case "separate-messages":
							Payload(func() {
								Field(1, "left", String)
							})
							Result(func() {
								Field("01", "right", String)
							})
						default:
							Payload(tagged)
						}
						if role == "http-only" {
							HTTP(func() {
								POST("/call")
							})
						} else {
							GRPC(func() {
								switch role {
								case "request-metadata":
									Metadata(func() {
										Attribute("value")
									})
								case "response-header":
									Response(CodeOK, func() {
										Headers(func() {
											Attribute("value")
										})
									})
								case "response-trailer":
									Response(CodeOK, func() {
										Trailers(func() {
											Attribute("value")
										})
									})
								}
							})
						}
					})
				})
			})
		})
	}
}
