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
			Attribute("string_items", ArrayOf(String))
			Attribute("object_items", ArrayOf(ServerDetails))
			Attribute("any_items", ArrayOfRequired(Any))
			Attribute("nullable_items", ArrayOf(ServerDetails, func() {
				Nullable()
			}))
			Attribute("optional_value", String)
			Attribute("required_value", String)
			Required("string_items", "object_items", "nullable_items", "required_value")
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
		var RequiredDefaultSort = Type("RequiredDefaultSort", func() {
			Attribute("field", String, func() {
				Default("display_id")
				Enum("display_id", "created_at")
			})
			Required("field")
		})
		var RequiredDefaultRequest = Type("RequiredDefaultRequest", func() {
			Attribute("sort", RequiredDefaultSort)
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
			Method("RequiredDefaultSubmit", func() {
				Payload(RequiredDefaultRequest)
				HTTP(func() {
					POST("/required-default")
					Body(RequiredDefaultRequest)
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
	require.Contains(t, clientTypes.String(), "Labels loom.Optional[[]loom.Nullable[string]]")
	require.Contains(t, clientTypes.String(), "body.Labels.Value()")
	serverTypeFiles, err := filepath.Glob(filepath.Join(dir, "gen", "http", "optional_union_validation", "server", "types*.go"))
	require.NoError(t, err)
	var serverTypes strings.Builder
	for _, typeFile := range serverTypeFiles {
		contents, readErr := os.ReadFile(typeFile)
		require.NoError(t, readErr)
		serverTypes.Write(contents)
	}
	require.Contains(t, serverTypes.String(), "Values loom.Optional[[]loom.Nullable[string]]")
	require.Contains(t, serverTypes.String(), "[]loom.Nullable[string]")
	require.Contains(t, serverTypes.String(), "[]loom.Nullable[*ServerDetailsRequestBodyRequestBody]")
	require.Contains(t, serverTypes.String(), "[]loom.Nullable[ServerDetailsRequestBodyRequestBody]")
	require.Contains(t, serverTypes.String(), "AnyItems")
	require.Contains(t, serverTypes.String(), "loom.Optional[[]loom.Nullable[loom.JSONValue]]")
	require.Contains(t, serverTypes.String(), `loom.InvalidNullElementError("body.string_items", i)`)
	require.Contains(t, serverTypes.String(), `loom.InvalidNullElementError("body.object_items", i)`)
	require.Contains(t, serverTypes.String(), `loom.InvalidNullElementError("body.any_items", i)`)
	require.Contains(t, serverTypes.String(), "OptionalValue loom.Optional[string]")
	require.Contains(t, serverTypes.String(), "RequiredValue *string")
	require.Contains(t, serverTypes.String(), "Field loom.Optional[string]")
	require.Contains(t, serverTypes.String(), "if !actual.Field.Present()")
	require.NotContains(t, serverTypes.String(), "actual.Field == nil")
	require.Contains(t, serverTypes.String(), "func ValidateServerSharedNestedRequestBodyRequestBody")
	require.Contains(t, serverTypes.String(), "func ValidateResponseRootShared")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "presence_regression_test.go"), []byte(`package optionalunionvalidation_test

import (
	"context"
	json "encoding/json/v2"
	"strings"
	"testing"

	loomhttp "github.com/CaliLuke/loom/http"
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

func TestArrayItemNullability(t *testing.T) {
	tests := []struct {
		name string
		body string
		wantErrorPath string
	}{
		{
			name: "null scalar",
			body: `+"`"+`{"string_items":[null],"object_items":[{"code":"ok"}],"nullable_items":[null],"required_value":"ok"}`+"`"+`,
			wantErrorPath: "body.string_items[0]",
		},
		{
			name: "null object",
			body: `+"`"+`{"string_items":["ok"],"object_items":[null],"nullable_items":[null],"required_value":"ok"}`+"`"+`,
			wantErrorPath: "body.object_items[0]",
		},
		{
			name: "null required any",
			body: `+"`"+`{"string_items":["ok"],"object_items":[{"code":"ok"}],"any_items":[null],"nullable_items":[null],"required_value":"ok"}`+"`"+`,
			wantErrorPath: "body.any_items[0]",
		},
		{
			name: "nullable object",
			body: `+"`"+`{"string_items":["ok"],"object_items":[{"code":"ok"}],"nullable_items":[null],"required_value":"ok"}`+"`"+`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var body server.ServerSharedNestedRequestBodyRequestBody
			if err := json.Unmarshal([]byte(test.body), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			err := server.ValidateServerSharedNestedRequestBodyRequestBody(&body)
			if test.wantErrorPath == "" {
				if err != nil {
					t.Fatalf("validate nullable item: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantErrorPath) {
				t.Fatalf("validation error = %v, want path %s", err, test.wantErrorPath)
			}
			if status := loomhttp.NewErrorResponse(context.Background(), err).StatusCode(); status != 400 {
				t.Fatalf("status = %d, want 400", status)
			}
		})
	}
}

`), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}
