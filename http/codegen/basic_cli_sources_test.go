package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	servicedata "github.com/CaliLuke/loom/codegen/service/testdata"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
)

func TestBasicCLISourceOwnership(t *testing.T) {
	for _, mapping := range []string{"basic", "query-header", "optional", "defaulted", "path-cookie", "repeated", "ordinary-repeated", "scalar-body", "object-body"} {
		t.Run(mapping, func(t *testing.T) {
			data := CreateHTTPServices(RunHTTPDSL(t, servicedata.BasicCLISourcesDSL("http", mapping)))
			service := data.Get("sources")
			endpoint := service.Endpoint("call")
			init := endpoint.Payload.Request.cliPayloadInit()
			original := append(append([]*InitArgData(nil), init.ClientArgs...), init.CLIArgs...)
			first := make(map[string]*InitArgData)
			for _, arg := range original {
				if first[arg.Name] == nil {
					first[arg.Name] = arg
				}
			}
			args := cliPayloadArgs(init)
			counts := make(map[string]int)
			for _, arg := range args {
				require.Same(t, first[arg.Name], arg, "retain complete first record for %s", arg.Name)
				if arg.FieldName != "" {
					counts[arg.Name]++
				}
			}
			for source, count := range counts {
				require.Equal(t, 1, count, "direct CLI inputs for payload source %s", source)
			}
			require.Equal(t, len(first), len(args))
			second := cliPayloadArgs(init)
			require.Equal(t, args, second)
			second[0] = nil
			require.NotNil(t, args[0], "CLI selections must have independent slice storage")
			require.Equal(t, original, append(append([]*InitArgData(nil), init.ClientArgs...), init.CLIArgs...))
			flags, builder := buildFlags(service, endpoint)
			require.NotNil(t, builder)
			seen := make(map[string]bool)
			for _, flag := range flags {
				require.False(t, seen[flag.FullName], "duplicate flag %s", flag.FullName)
				seen[flag.FullName] = true
			}
		})
	}
}

func TestBasicCLISourcesGenerated(t *testing.T) {
	for _, mapping := range []string{"basic", "query-header", "optional", "defaulted", "path-cookie", "repeated", "ordinary-repeated", "scalar-body", "object-body"} {
		t.Run(mapping, func(t *testing.T) {
			dir := t.TempDir()
			renderBasicCLISources(t, dir, mapping)
			harness := strings.NewReplacer("TRANSPORT", "http", "MAPPING", mapping).Replace(servicedata.BasicCLISourcesHarness)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "sources_test.go"), []byte(harness), 0o600))
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "build", "./...")
			runGoCommand(t, dir, "vet", "./...")
			runGoCommand(t, dir, "test", "-race", "./...")
		})
	}
}

func TestBasicCLISourceArtifacts(t *testing.T) {
	output := os.Getenv("LOOM_BASIC_SOURCES_OUTPUT")
	if output == "" {
		t.Skip("set LOOM_BASIC_SOURCES_OUTPUT for isolated artifact comparison")
	}
	for _, mapping := range []string{"basic", "query-header", "optional", "defaulted", "path-cookie", "repeated", "ordinary-repeated", "scalar-body", "object-body"} {
		t.Run(mapping, func(t *testing.T) {
			dir := filepath.Join(output, "http", mapping)
			require.NoError(t, os.MkdirAll(dir, 0o700))
			renderBasicCLISources(t, dir, mapping)
		})
	}
}

func renderBasicCLISources(t *testing.T, dir, mapping string) {
	t.Helper()
	root := RunHTTPDSL(t, servicedata.BasicCLISourcesDSL("http", mapping))
	files, err := openapiv3.Files(root)
	require.NoError(t, err)
	renderGeneratedFiles(t, dir, files)
	renderHTTPModule(t, dir, "example.com/sources", root)
	path := filepath.Join(dir, "go.mod")
	module, err := os.ReadFile(path)
	require.NoError(t, err)
	module = []byte(strings.Replace(string(module), "github.com/CaliLuke/loom v1.0.0", "github.com/CaliLuke/loom v1.10.0-alpha.9.0.20261013020843-063198a2f427", 1))
	require.NoError(t, os.WriteFile(path, module, 0o600))
}
