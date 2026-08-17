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
	require.Contains(t, serverTypes.String(), "func ValidateResponseRootShared")
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}
