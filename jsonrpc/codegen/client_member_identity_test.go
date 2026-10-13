package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestJSONRPCClientMemberIdentity(t *testing.T) {
	for _, kind := range []string{"rpc-ordinary", "rpc-unary", "rpc-sse", "rpc-websocket"} {
		t.Run(kind, func(t *testing.T) {
			root := RunJSONRPCDSL(t, testdata.ClientMemberIdentityDSL(kind))
			services := CreateJSONRPCServices(root)
			canonical := services.ServicesData.Get("members")
			names := make([]string, len(canonical.Methods))
			for index, method := range canonical.Methods {
				names[index] = method.VarName
			}
			data := services.Get("members")
			for index, endpoint := range data.Endpoints {
				require.Equal(t, names[index], endpoint.Method.VarName)
			}
			files := ClientFiles("example.com/clientmembers/gen", services)
			sources := make([]string, 0, len(files))
			for _, file := range files {
				var source strings.Builder
				for _, section := range file.AllSections() {
					require.NoError(t, section.Write(&source))
				}
				sources = append(sources, source.String())
			}
			members, err := testdata.ClientMemberDeclarations(sources)
			require.NoError(t, err)
			for receiver, names := range members {
				unique := make(map[string]bool)
				for _, name := range names {
					require.False(t, unique[name], "%s.%s must have one declaration", receiver, name)
					unique[name] = true
				}
			}
			expected := []string{"RestoreResponseBody", "scheme", "host", "encoder", "decoder"}
			expected = append(expected, "Doer")
			if kind == "rpc-websocket" {
				expected = append(expected, "dialer", "webSocketConfig", "connMu", "conn", "closed", "streamConfig", "getConn", "Close", "IsClosed")
			}
			for _, endpoint := range data.Endpoints {
				expected = append(expected, endpoint.EndpointInit, endpoint.RequestInit.Name)
				if endpoint.ClientDoer != "" {
					expected = append(expected, endpoint.ClientDoer)
				}
				if endpoint.StreamEndpointInit != "" {
					expected = append(expected, endpoint.StreamEndpointInit)
				}
			}
			require.ElementsMatch(t, expected, members["Client"], "all emitted receiver roles must belong to the analysis plan")
		})
	}
}

func TestJSONRPCClientMemberIdentityGenerated(t *testing.T) {
	for _, kind := range []string{"rpc-ordinary", "rpc-unary", "rpc-sse", "rpc-websocket"} {
		t.Run(kind, func(t *testing.T) {
			root := RunJSONRPCDSL(t, testdata.ClientMemberIdentityDSL(kind))
			dir := t.TempDir()
			if output := os.Getenv("LOOM_CLIENT_MEMBER_OUTPUT"); output != "" {
				dir = filepath.Join(output, kind)
				require.NoError(t, os.MkdirAll(dir, 0o700))
			}
			renderJSONRPCModule(t, dir, "example.com/clientmembers", root)
			renderCodegenFiles(t, dir, ClientCLIFiles("example.com/clientmembers/gen", CreateJSONRPCServices(root)))
			if os.Getenv("LOOM_CLIENT_MEMBER_GENERATE_ONLY") != "" {
				return
			}
			runtime, err := os.ReadFile(filepath.Join("..", "..", "http", "codegen", "testdata", "client_members", strings.ReplaceAll(kind, "-", "_")+".go.txt"))
			if err == nil {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "client_roles_test.go"), runtime, 0o600))
			} else {
				require.True(t, os.IsNotExist(err), "read mounted harness: %v", err)
			}
			runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
			runGoJSONRPCTestCommand(t, dir, "build", "./...")
			runGoJSONRPCTestCommand(t, dir, "vet", "./...")
			if runtime != nil {
				runGoJSONRPCTestCommand(t, dir, "test", "-race", "-timeout=30s", ".")
			}
		})
	}
}
