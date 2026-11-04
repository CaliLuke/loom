package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestHTTPClientMemberIdentityGenerated(t *testing.T) {
	batch := testingx.NewModuleBatch(t, checkoutPinnedLoomModule(t), os.Getenv("LOOM_CLIENT_MEMBER_OUTPUT"))
	for _, kind := range []string{"http-ordinary", "http-fixed", "http-mixed", "http-mixed-stream-collision", "http-grouped", "http-websocket"} {
		t.Run(kind, func(t *testing.T) {
			root := RunHTTPDSL(t, testdata.ClientMemberIdentityDSL(kind))
			assertHTTPClientMemberIdentity(t, root, kind)
			c := batch.Case(t, kind, "example.com/clientmembers")
			renderHTTPModuleFiles(t, c.Dir, c.ModulePath+"/gen", root)
			if os.Getenv("LOOM_CLIENT_MEMBER_GENERATE_ONLY") != "" {
				return
			}
			runtime, err := os.ReadFile(filepath.Join("testdata", "client_members", strings.ReplaceAll(kind, "-", "_")+".go.txt"))
			if err == nil {
				c.WriteFile(t, "client_roles_test.go", string(runtime))
			} else {
				require.True(t, os.IsNotExist(err), "read mounted harness: %v", err)
			}
		})
	}
	if os.Getenv("LOOM_CLIENT_MEMBER_GENERATE_ONLY") == "" {
		batch.Check(t)
	}
}

func assertHTTPClientMemberIdentity(t *testing.T, root *expr.RootExpr, kind string) {
	t.Helper()
	services := CreateHTTPServices(root)
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
	if kind == "http-websocket" {
		expected = append(expected, "dialer", "configurer")
	}
	for _, group := range data.clientGroups {
		expected = append(expected, group.FieldName)
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
}
