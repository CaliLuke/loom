package dsl

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

func TestGRPCPresenceAdmissionUsesServiceContract(t *testing.T) {
	for _, location := range []string{"payload", "result", "streaming payload", "streaming result"} {
		for _, shape := range []string{"named object", "anonymous object", "array element", "map value", "scalar"} {
			t.Run(location+"/"+shape, func(t *testing.T) {
				expr.SetupTestDSL(t)
				require.True(t, eval.Execute(func() {
					leaf := Type("Leaf", func() {
						Field(1, "x", Int)
					})
					attribute := func() {
						switch shape {
						case "named object":
							Field(1, "maybe", leaf, func() {
								Nullable()
							})
						case "anonymous object":
							Field(1, "maybe", func() {
								Field(1, "x", Int)
								Nullable()
							})
						case "array element":
							Field(1, "maybe", ArrayOf(leaf, func() {
								Nullable()
							}))
						case "map value":
							Field(1, "maybe", MapOf(String, leaf, func() {
								Elem(func() {
									Nullable()
								})
							}))
						case "scalar":
							Field(1, "maybe", String, func() {
								Nullable()
							})
						}
					}
					Service("nullable", func() {
						Method("send", func() {
							switch location {
							case "payload":
								Payload(attribute)
							case "result":
								Result(attribute)
							case "streaming payload":
								StreamingPayload(attribute)
							case "streaming result":
								StreamingResult(attribute)
							}
							GRPC(func() {})
						})
					})
				}, nil), eval.Context.Error())
				require.ErrorContains(t, eval.RunDSL(), "do not support nullable attributes")
			})
		}
	}
}

func TestGRPCPresenceAdmissionPreservesSupportedShapes(t *testing.T) {
	for _, shape := range []string{"native", "Any root", "Any field"} {
		t.Run(shape, func(t *testing.T) {
			expr.RunDSL(t, func() {
				Service("supported", func() {
					Method("send", func() {
						switch shape {
						case "native":
							Payload(func() {
								Field(1, "object", func() {
									Field(1, "x", Int)
								})
								Field(2, "array", ArrayOf(String))
								Field(3, "map", MapOf(String, Int))
							})
						case "Any root":
							Payload(Any, func() {
								Nullable()
							})
						case "Any field":
							Payload(func() {
								Field(1, "value", Any, func() {
									Nullable()
								})
							})
						}
						GRPC(func() {})
					})
				})
			})
		})
	}
}

func TestGRPCPresenceAdmissionUsesMappedSourceContract(t *testing.T) {
	for _, location := range []string{"request message", "request metadata", "response message", "response header", "response trailer", "error message"} {
		t.Run(location, func(t *testing.T) {
			expr.SetupTestDSL(t)
			require.True(t, eval.Execute(func() {
				value := Type("Value", func() {
					Field(1, "maybe", String, func() {
						Nullable()
					})
				})
				Service("mapped", func() {
					Method("send", func() {
						if strings.HasPrefix(location, "request") {
							Payload(value)
						}
						if strings.HasPrefix(location, "response") {
							Result(value)
						}
						if location == "error message" {
							Error("bad", value)
						}
						GRPC(func() {
							mapping := func() {
								Attribute("maybe")
							}
							switch location {
							case "request message":
								Message(mapping)
							case "request metadata":
								Metadata(mapping)
							case "response message":
								Response(CodeOK, func() {
									Message(mapping)
								})
							case "response header":
								Response(CodeOK, func() {
									Headers(mapping)
								})
							case "response trailer":
								Response(CodeOK, func() {
									Trailers(mapping)
								})
							case "error message":
								Response("bad", CodeInvalidArgument, func() {
									Message(mapping)
								})
							}
						})
					})
				})
			}, nil), eval.Context.Error())
			require.ErrorContains(t, eval.RunDSL(), "do not support nullable attributes")
		})
	}
}

func TestGRPCPresenceAdmissionChecksLocalDeclarations(t *testing.T) {
	for _, anySource := range []bool{false, true} {
		expr.SetupTestDSL(t)
		require.True(t, eval.Execute(func() {
			Service("local", func() {
				Method("send", func() {
					Payload(func() {
						if anySource {
							Field(1, "value", Any)
						} else {
							Field(1, "value", String)
						}
					})
					GRPC(func() {
						Message(func() {
							Attribute("value", func() {
								Nullable()
							})
						})
					})
				})
			})
		}, nil), eval.Context.Error())
		if anySource {
			require.NoError(t, eval.RunDSL())
		} else {
			require.ErrorContains(t, eval.RunDSL(), "request messages do not support nullable attributes")
		}
	}
}

func TestGRPCPresenceAdmissionRejectsAnyMetadata(t *testing.T) {
	for _, location := range []string{"request", "header", "trailer"} {
		t.Run(location, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				value := Type("Value", func() {
					Field(1, "value", Any)
				})
				Service("metadata", func() {
					Method("send", func() {
						Payload(value)
						Result(value)
						GRPC(func() {
							mapping := func() {
								Attribute("value:x-value")
							}
							if location == "request" {
								Metadata(mapping)
							} else {
								Response(CodeOK, func() {
									if location == "header" {
										Headers(mapping)
									} else {
										Trailers(mapping)
									}
								})
							}
						})
					})
				})
			})
			require.ErrorContains(t, err, "metadata do not support nullable attributes")
		})
	}
}

func TestGRPCPresenceAdmissionUsesScopedErrorSource(t *testing.T) {
	for _, inheritedNullable := range []bool{false, true} {
		expr.SetupTestDSL(t)
		require.True(t, eval.Execute(func() {
			Service("errors", func() {
				Error("bad", func() {
					Field(1, "detail", String, func() {
						if inheritedNullable {
							Nullable()
						}
					})
				})
				GRPC(func() {
					Response("bad", CodeInvalidArgument)
				})
				Method("send", func() {
					Error("bad", func() {
						Field(1, "detail", String, func() {
							if !inheritedNullable {
								Nullable()
							}
						})
					})
					GRPC(func() {})
				})
			})
		}, nil), eval.Context.Error())
		if inheritedNullable {
			require.ErrorContains(t, eval.RunDSL(), "error response results do not support nullable attributes")
		} else {
			require.NoError(t, eval.RunDSL())
		}
	}
}

func TestGRPCPresenceAdmissionRetainsInvalidMappingDiagnostics(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		Service("invalid", func() {
			Method("send", func() {
				Payload(func() {
					Field(1, "known", String)
				})
				GRPC(func() {
					Message(func() {
						Attribute("missing", func() {
							Nullable()
						})
					})
				})
			})
		})
	})
	require.ErrorContains(t, err, `message attribute "missing" is not found in Payload`)
	require.ErrorContains(t, err, "request messages do not support nullable attributes")
}

func TestGRPCPresenceAdmissionChecksNestedLocalDeclarations(t *testing.T) {
	for _, shape := range []string{"nested object", "array element", "map value", "scalar wrapper"} {
		for _, anySource := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/any=%t", shape, anySource), func(t *testing.T) {
				expr.SetupTestDSL(t)
				require.True(t, eval.Execute(func() {
					var primitive expr.DataType = String
					if anySource {
						primitive = Any
					}
					Service("local", func() {
						Method("send", func() {
							switch shape {
							case "nested object":
								Payload(func() {
									Field(1, "value", func() {
										Field(1, "child", primitive)
									})
								})
							case "array element":
								Payload(func() {
									Field(1, "value", ArrayOf(primitive))
								})
							case "map value":
								Payload(func() {
									Field(1, "value", MapOf(String, primitive))
								})
							case "scalar wrapper":
								Payload(primitive)
							}
							GRPC(func() {
								Message(func() {
									switch shape {
									case "nested object":
										Attribute("value", func() {
											Attribute("child", func() {
												Nullable()
											})
										})
									case "array element":
										Attribute("value", ArrayOf(String, func() {
											Nullable()
										}))
									case "map value":
										Attribute("value", MapOf(String, String, func() {
											Elem(func() {
												Nullable()
											})
										}))
									case "scalar wrapper":
										Field(1, "value", primitive, func() {
											Nullable()
										})
									}
								})
							})
						})
					})
				}, nil), eval.Context.Error())
				if anySource && (shape == "nested object" || shape == "scalar wrapper") {
					require.NoError(t, eval.RunDSL())
				} else {
					require.ErrorContains(t, eval.RunDSL(), "do not support nullable attributes")
				}
			})
		}
	}
}
