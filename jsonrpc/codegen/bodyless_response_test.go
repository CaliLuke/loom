package codegen

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dave/jennifer/jen"
	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

//go:embed testdata/bodyless_response_harness.go.txt
var bodylessResponseHarness string

func TestJSONRPCBodylessResultBinding(t *testing.T) {
	root := RunJSONRPCDSL(t, bodylessResponseDSL)
	data := CreateJSONRPCServices(root).Get("bodyless")
	for _, endpoint := range data.Endpoints {
		t.Run(endpoint.Method.Name, func(t *testing.T) {
			wantBody := endpoint.Method.Name == "value"
			require.Equal(t, wantBody, endpoint.Result.Responses[0].EncodePlan.HasBody)
			wantResult := wantBody || endpoint.Method.Name == "metadata"
			source := fmt.Sprintf("%#v", jen.BlockFunc(func(group *jen.Group) {
				writeJSONRPCEndpointInvoke(group, endpoint)
			}))
			require.Equal(t, wantResult, strings.Contains(source, "res, err := endpoint"), source)
		})
	}
}

func TestJSONRPCBodylessResultGenerated(t *testing.T) {
	root := RunJSONRPCDSL(t, bodylessResponseDSL)
	dir := t.TempDir()
	if output := os.Getenv("LOOM_BODYLESS_OUTPUT"); output != "" {
		dir = output
		require.NoError(t, os.MkdirAll(dir, 0o750))
	}
	renderJSONRPCModule(t, dir, "example.com/bodyless", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "response_test.go"), []byte(bodylessResponseHarness), 0o600))
	if os.Getenv("LOOM_BODYLESS_GENERATE_ONLY") != "" {
		return
	}
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "build", "./...")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
}

func bodylessResponseDSL() {
	API("bodyless", func() {
		JSONRPC(func() {})
	})
	local := Type("LocalEmpty", func() {})
	moved := Type("MovedEmpty", func() {
		Meta("struct:pkg:path", "common")
	})
	Service("bodyless", func() {
		JSONRPC(func() {
			POST("/rpc")
		})
		for _, tc := range []struct {
			name    string
			result  any
			payload bool
		}{
			{"local", local, true}, {"local_bare", local, false},
			{"moved", moved, true}, {"moved_bare", moved, false},
			{"nothing", nil, false}, {"value", String, false},
		} {
			Method(tc.name, func() {
				if tc.payload {
					Payload(func() {
						ID("id", String)
					})
				}
				if tc.result != nil {
					Result(tc.result)
				}
				JSONRPC(func() {})
			})
		}
		Method("metadata", func() {
			Result(func() {
				Attribute("header", String)
				Attribute("cookie", String)
				Required("header", "cookie")
			})
			JSONRPC(func() {
				Response(func() {
					Header("header:X-Result")
					Cookie("cookie:result")
				})
			})
		})
	})
}
