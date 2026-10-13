package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

func TestElementAdmissionArtifacts(t *testing.T) {
	output := os.Getenv("LOOM_ELEMENT_ADMISSION_OUTPUT")
	if output == "" {
		t.Skip("set LOOM_ELEMENT_ADMISSION_OUTPUT for isolated output comparison")
	}
	for _, tc := range []struct {
		name string
		dsl  func()
	}{
		{"nil-items", nilCollectionDSL},
		{"physical-carriers", nilElementWireDSL},
		{"inline", testdata.InlineObjectFieldsDSL},
		{"recursive-map", recursiveNestedCollectionDSL(mapOfMapsIndex, "", nil)},
		{"Any", testdata.AnyErrorDSL},
		{"transport", elementAdmissionTransportDSL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(output, tc.name)
			require.NoError(t, os.MkdirAll(dir, 0o700))
			root := RunGRPCDSL(t, tc.dsl)
			renderGRPCModule(t, dir, "example.com/elementadmission", root, resolveGRPCLoomSource(t))
			for _, file := range ClientCLIFiles("example.com/elementadmission/gen", CreateGRPCServices(root)) {
				_, err := file.Render(dir)
				require.NoError(t, err, file.Path)
			}
		})
	}
}

func TestElementAdmissionArtifactBuilds(t *testing.T) {
	output := os.Getenv("LOOM_ELEMENT_ADMISSION_OUTPUT")
	if output == "" {
		t.Skip("set LOOM_ELEMENT_ADMISSION_OUTPUT for isolated output compilation")
	}
	for _, name := range []string{"nil-items", "physical-carriers", "inline", "recursive-map", "Any", "transport"} {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(output, name)
			runGRPCGoCommand(t, dir, "mod", "tidy")
			runGRPCGoCommand(t, dir, "build", "./...")
			runGRPCGoCommand(t, dir, "vet", "./...")
		})
	}
}

func TestElementAdmissionErrorImportBinding(t *testing.T) {
	root := RunGRPCDSL(t, func() {
		leaf := Type("Leaf", func() {
			Field(1, "value", String)
		})
		Service("errors", func() {
			Method("echo", func() {
				Payload(ArrayOf(leaf))
				Result(ArrayOf(leaf))
				GRPC(func() {})
			})
		})
	})
	const module = "example.com/admissionimports"
	dir := t.TempDir()
	renderGRPCModule(t, dir, module, root, resolveGRPCLoomSource(t))
	services := CreateGRPCServices(root)
	require.Equal(t, "errors_", services.Get("errors").Service.PkgName)
	for _, file := range ClientCLIFiles(module+"/gen", services) {
		_, err := file.Render(dir)
		require.NoError(t, err)
	}
	for _, side := range []string{"client", "server"} {
		content, err := os.ReadFile(filepath.Join(dir, "gen", "grpc", services.Get("errors").Service.PathName, side, "types.go"))
		require.NoError(t, err)
		require.Contains(t, string(content), `"errors"`)
		require.Contains(t, string(content), `errors_ "example.com/admissionimports/gen/errors_"`)
	}
	cli, err := os.ReadFile(filepath.Join(dir, "gen", "grpc", services.Get("errors").Service.PathName, "client", "cli.go"))
	require.NoError(t, err)
	require.Contains(t, string(cli), `errors.New("gRPC conversion: nil message collection element")`)
	require.Contains(t, string(cli), `errors_ "example.com/admissionimports/gen/errors_"`)
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "build", "./...")
	runGRPCGoCommand(t, dir, "vet", "./...")
}
