package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

func TestGRPCMetadataImplicitCredentialCollision(t *testing.T) {
	for _, collide := range []bool{false, true} {
		t.Run(map[bool]string{false: "separate", true: "collision"}[collide], func(t *testing.T) {
			expr.SetupTestDSL(t)
			require.True(t, eval.Execute(func() {
				key := APIKeySecurity("key")
				Service("binding", func() {
					Method("call", func() {
						Security(key)
						Payload(func() {
							APIKey("key", "token", String)
							Field(1, "other", String)
							Required("token")
						})
						GRPC(func() {
							Metadata(func() {
								name := "x-other"
								if collide {
									name = "AUTHORIZATION"
								}
								Attribute("other:" + name)
							})
						})
					})
				})
			}, nil))
			err := eval.RunDSL()
			if collide {
				require.Error(t, err)
				require.Contains(t, err.Error(), "authorization")
				require.Contains(t, err.Error(), "token")
				require.Contains(t, err.Error(), "other")
				return
			}
			require.NoError(t, err)
			e := expr.Root.API.GRPC.Service("binding").GRPCEndpoints[0]
			require.Equal(t, "authorization", e.Metadata.ElemName("token"))
			require.True(t, e.Metadata.IsRequired("token"))
			require.Equal(t, "authorization", e.Requirements[0].Schemes[0].Name)
		})
	}
}

func TestGRPCMetadataChannelsRemainIndependent(t *testing.T) {
	root := expr.RunDSL(t, func() {
		Service("binding", func() {
			Method("call", func() {
				Payload(func() {
					Field(1, "input", Bytes)
				})
				Result(func() {
					Field(1, "first", Bytes)
					Field(2, "last", Bytes)
				})
				GRPC(func() {
					Metadata(func() {
						Attribute("input:x-raw-bin")
					})
					Response(CodeOK, func() {
						Headers(func() {
							Attribute("first:x-raw-bin")
						})
						Trailers(func() {
							Attribute("last:x-raw-bin")
						})
					})
				})
			})
		})
	})
	e := root.API.GRPC.Service("binding").GRPCEndpoints[0]
	require.Equal(t, "x-raw-bin", e.Metadata.ElemName("input"))
	require.Equal(t, "x-raw-bin", e.Response.Headers.ElemName("first"))
	require.Equal(t, "x-raw-bin", e.Response.Trailers.ElemName("last"))
}

func TestGRPCMetadataInheritedErrorUsesMappingSource(t *testing.T) {
	root := expr.RunDSL(t, func() {
		Service("binding", func() {
			Error("broken", func() {
				Field(1, "raw", Bytes)
				Required("raw")
			})
			GRPC(func() {
				Response("broken", CodeInternal, func() {
					Headers(func() {
						Attribute("raw:x-raw")
					})
				})
			})
			Method("call", func() {
				Error("broken", func() {
					Field(1, "raw", String)
				})
				Result(String)
				GRPC(func() {
				})
			})
		})
	})
	e := root.API.GRPC.Service("binding").GRPCEndpoints[0]
	require.Len(t, e.GRPCErrors, 1)
	mapped := e.GRPCErrors[0].Response.Headers
	require.Equal(t, expr.Bytes, mapped.Find("raw").Type)
	require.Equal(t, "x-raw-bin", mapped.ElemName("raw"))
	require.True(t, mapped.IsRequired("raw"))
}

func TestGRPCMetadataBindingRetainsSourceAndScope(t *testing.T) {
	for _, placement := range []string{"implicit", "explicit", "message"} {
		t.Run(placement, func(t *testing.T) {
			root := expr.RunDSL(t, func() {
				key := APIKeySecurity("key")
				Service("binding", func() {
					Security(key)
					Method("call", func() {
						Payload(func() {
							APIKeyField(2, "key", "token", String)
							Field(1, "raw:data", Bytes, func() {
								Meta("struct:field:name", "Binary")
							})
							Required("token", "raw")
						})
						GRPC(func() {
							if placement == "message" {
								Message(func() {
									Field(2, "token")
								})
							}
							Metadata(func() {
								Attribute("raw:X-RAW")
								if placement == "explicit" {
									Attribute("token:X-TOKEN")
								}
							})
						})
					})
				})
			})
			e := root.API.GRPC.Service("binding").GRPCEndpoints[0]
			mapped := e.Metadata
			require.Equal(t, "x-raw-bin", mapped.ElemName("raw"))
			key, source := e.MethodExpr.Payload.FindAttribute("raw")
			require.Equal(t, "raw:data", key)
			require.Equal(t, expr.Bytes, source.Type)
			require.Equal(t, []string{"Binary"}, source.Meta["struct:field:name"])
			require.True(t, e.MethodExpr.Payload.IsRequired(key))
			for range 2 {
				var errors *eval.ValidationErrors
				require.ErrorAs(t, e.Validate(), &errors)
				require.Empty(t, errors.Errors)
			}
			require.Same(t, mapped, e.Metadata, "repeated admission must reuse canonical bindings")
			scheme := e.Requirements[0].Schemes[0]
			switch placement {
			case "implicit":
				require.Equal(t, "authorization", scheme.Name)
				require.Equal(t, "metadata", scheme.In)
			case "explicit":
				require.Equal(t, "x-token", scheme.Name)
				require.Equal(t, "metadata", scheme.In)
			default:
				require.Equal(t, "token", scheme.Name)
				require.Equal(t, "message", scheme.In)
				require.Nil(t, e.Metadata.Find("token"))
			}
		})
	}
}

func TestGRPCMetadataRejectsImplicitSessionCollision(t *testing.T) {
	expr.SetupTestDSL(t)
	require.True(t, eval.Execute(func() {
		jwt := JWTSecurity("jwt")
		key := APIKeySecurity("key")
		session := SessionAuth("session", func() {
			BearerTransport(jwt, "auth")
			CookieTransport(key, "cookie")
		})
		Service("binding", func() {
			SessionSecurity(session)
			Method("call", func() {
				Payload(func() {
					Field(1, "message", String)
				})
				GRPC(func() {
				})
			})
		})
	}, nil))
	err := eval.RunDSL()
	require.ErrorContains(t, err, `attributes "auth" and "cookie" share canonical key "authorization"`)
}
