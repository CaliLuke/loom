package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestGeneratedOptionalUnionObjectValidationCompiles(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		var Details = Type("Details", func() {
			Attribute("code", String, func() {
				MinLength(1)
			})
			Required("code")
		})
		var ServerDetails = Type("ServerDetails", func() {
			Attribute("code", String, func() {
				MinLength(1)
			})
			Required("code")
		})
		var Envelope = Type("Envelope", func() {
			OneOf("choice", func() {
				Attribute("object", func() {
					Attribute("default_code", String, func() {
						Default("fallback")
						MinLength(1)
					})
					Attribute("optional_code", String, func() {
						MinLength(1)
					})
					Attribute("required_code", String, func() {
						MinLength(1)
					})
					Required("required_code")
				})
				Attribute("text", String)
				Attribute("nullable_items", ArrayOf(Details, func() {
					Nullable()
				}))
			})
			Attribute("details", Details, func() {
				Nullable()
			})
		})
		var Submission = Type("Submission", func() {
			OneOf("server_choice", func() {
				Attribute("server_items", ArrayOf(ServerDetails, func() {
					Nullable()
				}))
				Attribute("server_text", String)
			})
			Attribute("details", ServerDetails)
		})
		var SharedBody = Type("SharedBody", func() {
			Attribute("default_code", String, func() {
				Default("fallback")
				MinLength(1)
			})
			Attribute("optional_code", String, func() {
				MinLength(1)
			})
			Attribute("required_code", String, func() {
				MinLength(1)
			})
			Required("required_code")
		})
		var CrossNested = Type("CrossNested", func() {
			Attribute("labels", ArrayOf(String), func() {
				MinLength(1)
			})
		})
		var CrossRequest = Type("CrossRequest", func() {
			Attribute("nested", CrossNested)
		})
		var ServerSharedNested = Type("ServerSharedNested", func() {
			Attribute("values", ArrayOf(String), func() {
				MinLength(1)
			})
			Attribute("optional_value", String)
			Attribute("required_value", String)
			Required("required_value")
		})
		var ServerOrderRequest = Type("ServerOrderRequest", func() {
			Attribute("nested", ServerSharedNested)
		})
		var ServerOrderResponse = Type("ServerOrderResponse", func() {
			Attribute("nested", ServerSharedNested)
		})
		var ResponseRootShared = Type("ResponseRootShared", func() {
			Attribute("values", ArrayOf(String), func() {
				MinLength(1)
			})
		})
		var ResponseRootRequest = Type("ResponseRootRequest", func() {
			Attribute("nested", ResponseRootShared)
		})
		var HeaderBody = Type("HeaderBody", func() {
			Attribute("nickname", String)
			Attribute("labels", ArrayOf(String))
			Attribute("metadata", MapOf(String, String))
		})
		var HeaderResult = Type("HeaderResult", func() {
			Attribute("body", HeaderBody)
			Attribute("trace", String)
		})
		Service("OptionalUnionValidation", func() {
			Method("Show", func() {
				Result(Envelope)
				HTTP(func() {
					GET("/choice")
					Response(StatusOK)
				})
			})
			Method("Submit", func() {
				Payload(Submission)
				HTTP(func() {
					POST("/choices/submission")
					Body(Submission)
				})
			})
			Method("Echo", func() {
				Payload(SharedBody)
				Result(SharedBody)
				HTTP(func() {
					POST("/shared/echo")
					Body(SharedBody)
					Response(StatusOK)
				})
			})
			Method("CrossSubmit", func() {
				Payload(CrossRequest)
				HTTP(func() {
					POST("/cross")
					Body(CrossRequest)
				})
			})
			Method("CrossShow", func() {
				Result(CrossNested)
				HTTP(func() {
					GET("/cross")
					Response(StatusOK)
				})
			})
			Method("ServerOrderSubmit", func() {
				Payload(ServerOrderRequest)
				HTTP(func() {
					POST("/server-order")
					Body(ServerOrderRequest)
				})
			})
			Method("ServerOrderShow", func() {
				Result(ServerOrderResponse)
				HTTP(func() {
					GET("/server-order")
					Response(StatusOK)
				})
			})
			Method("ResponseRootShow", func() {
				Result(ResponseRootShared)
				HTTP(func() {
					GET("/response-root")
					Response(StatusOK)
				})
			})
			Method("ResponseRootSubmit", func() {
				Payload(ResponseRootRequest)
				HTTP(func() {
					POST("/response-root/request")
					Body(ResponseRootRequest)
				})
			})
			Method("HeaderShow", func() {
				Result(HeaderResult)
				HTTP(func() {
					GET("/header")
					Response(StatusOK, func() {
						Body("body")
						Header("trace")
					})
				})
			})
		})
	})
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/optionalunionvalidation", root)
	typeFiles, err := filepath.Glob(filepath.Join(dir, "gen", "http", "optional_union_validation", "client", "types*.go"))
	require.NoError(t, err)
	var clientTypes strings.Builder
	for _, typeFile := range typeFiles {
		contents, readErr := os.ReadFile(typeFile)
		require.NoError(t, readErr)
		clientTypes.Write(contents)
	}
	require.Contains(t, clientTypes.String(), "Labels loom.Optional[[]string]")
	require.Contains(t, clientTypes.String(), "body.Labels.Value()")
	serverTypeFiles, err := filepath.Glob(filepath.Join(dir, "gen", "http", "optional_union_validation", "server", "types*.go"))
	require.NoError(t, err)
	var serverTypes strings.Builder
	for _, typeFile := range serverTypeFiles {
		contents, readErr := os.ReadFile(typeFile)
		require.NoError(t, readErr)
		serverTypes.Write(contents)
	}
	require.Contains(t, serverTypes.String(), "Values loom.Optional[[]string]")
	require.Contains(t, serverTypes.String(), "OptionalValue loom.Optional[string]")
	require.Contains(t, serverTypes.String(), "RequiredValue *string")
	require.Contains(t, serverTypes.String(), "func ValidateServerSharedNestedRequestBodyRequestBody")
	require.Contains(t, serverTypes.String(), "func ValidateResponseRootShared")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "presence_regression_test.go"), []byte(`package optionalunionvalidation_test

import (
	json "encoding/json/v2"
	"testing"

	server "example.com/optionalunionvalidation/gen/http/optional_union_validation/server"
)

func TestSharedRequestPresence(t *testing.T) {
	var nullBody server.ServerSharedNestedRequestBodyRequestBody
	if err := json.Unmarshal([]byte("{\"optional_value\":null,\"required_value\":\"ok\"}"), &nullBody); err == nil {
		t.Error("expected explicit null to be rejected")
	}

	var missingRequired server.ServerSharedNestedRequestBodyRequestBody
	if err := json.Unmarshal([]byte("{}"), &missingRequired); err != nil {
		t.Fatalf("decode empty object: %v", err)
	}
	if err := server.ValidateServerSharedNestedRequestBodyRequestBody(&missingRequired); err == nil {
		t.Error("expected missing required value to be rejected")
	}
}
`), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}
