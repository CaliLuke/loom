package codegen

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	servicecodegen "github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestGRPCServiceLocalSymbolsGeneratedCompile(t *testing.T) {
	for _, name := range []string{"ordinary", "md", "ctx", "v", "result", "err", "req", "hdr", "trlr", "message", "p", "s", "stream"} {
		t.Run(name, func(t *testing.T) {
			dir := renderGRPCLocalSymbols(t, name, false)
			if os.Getenv("LOOM_691_GENERATE_ONLY") != "" {
				return
			}
			runGRPCGoCommand(t, dir, "mod", "tidy")
			runGRPCGoCommand(t, dir, "build", "./...")
			runGRPCGoCommand(t, dir, "vet", "./...")
		})
	}
}

func TestGRPCServiceLocalSymbolsScopeInventory(t *testing.T) {
	const module = "example.com/grpclocals"
	dir := renderGRPCLocalSymbols(t, "zzsvc", true)
	names := testingx.ServicePackageShadowNames(t, dir, module+"/gen", "zz")
	t.Logf("shadowing identifiers: %q", names)
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			root := RunGRPCDSL(t, func() {
				testdata.GRPCLocalSymbols(name)
			})
			require.NotEqual(t, name, CreateGRPCServices(root).Get(name).Service.PkgName,
				"service alias is shadowed by a generated identifier")
		})
	}
}

func TestGRPCServiceLocalSymbolsAliasContracts(t *testing.T) {
	for _, tc := range []struct {
		name          string
		serviceAlias  string
		protobufAlias string
	}{
		{"ordinary", "ordinary", "ordinarypb"},
		{"md", "mdsvc", "mdpb"},
		{"ctx", "ctxsvc", "ctxpb"},
		{"req", "req", "reqpb2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := RunGRPCDSL(t, func() {
				testdata.GRPCLocalSymbols(tc.name)
			})
			services := CreateGRPCServices(root)
			data := services.Get(tc.name)
			require.Equal(t, tc.name, services.ServicesData.Get(tc.name).PkgName)
			require.Equal(t, tc.name, data.Service.PathName)
			require.Equal(t, tc.name, data.ProtoPkg)
			require.Equal(t, tc.serviceAlias, data.Service.PkgName)
			require.Equal(t, tc.protobufAlias, data.PkgName)
		})
	}
}

func TestGRPCServiceLocalSymbolsDeclarationOrders(t *testing.T) {
	for _, names := range [][]string{
		{"md", "mdsvc", "mdsvc2", "ordinary"},
		{"mdsvc2", "mdsvc", "md", "ordinary"},
		{"req", "reqpb", "reqpb2", "ordinary"},
		{"reqpb2", "reqpb", "req", "ordinary"},
	} {
		root := RunGRPCDSL(t, func() {
			testdata.GRPCLocalSymbols(names...)
		})
		services := CreateGRPCServices(root)
		used := map[string]bool{"md": true, "reqpb": true}
		for _, name := range names {
			data := services.Get(name)
			for _, alias := range []string{data.Service.PkgName, data.PkgName} {
				require.False(t, used[alias], "duplicate or shadowed import %q for order %v", alias, names)
				used[alias] = true
			}
		}
	}
}

func TestGRPCServiceLocalSymbolsCustomCompile(t *testing.T) {
	const module = "example.com/customlocals"
	root := RunGRPCDSL(t, testdata.GRPCLocalSymbolsCustom)
	dir := t.TempDir()
	if base := os.Getenv("LOOM_691_ARTIFACTS"); base != "" {
		dir = filepath.Join(base, "custom")
		require.NoError(t, os.MkdirAll(dir, 0o750))
	}
	renderGRPCModule(t, dir, module, root, resolveGRPCLoomSource(t))
	if os.Getenv("LOOM_691_GENERATE_ONLY") != "" {
		return
	}
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "build", "./...")
	runGRPCGoCommand(t, dir, "vet", "./...")
	services := CreateGRPCServices(root)
	neutral := services.ServicesData.Get("md")
	require.NoError(t, servicecodegen.SetUserTypeImports(module+"/gen", neutral))
	data := services.Get("md")
	data, imports := services.fileData("md", grpcTypeImports(module+"/gen", root.API.GRPC.Service("md"), data))
	reserved := make(map[string]bool)
	for _, spec := range imports {
		if spec.Name != "" {
			reserved[spec.Name] = true
		}
	}
	endpoint := data.Endpoint("zzecho")
	for _, group := range [][]*MetadataData{
		endpoint.Request.Metadata,
		append(slices.Clone(endpoint.Response.Headers), endpoint.Response.Trailers...),
	} {
		used := make(map[string]bool)
		for _, item := range group {
			require.False(t, reserved[item.VarName], "metadata shadows actual import %q", item.VarName)
			require.False(t, used[item.VarName], "duplicate metadata name %q", item.VarName)
			used[item.VarName] = true
		}
	}
}

func renderGRPCLocalSymbols(t *testing.T, name string, examples bool) string {
	t.Helper()
	const module = "example.com/grpclocals"
	root := RunGRPCDSL(t, func() {
		testdata.GRPCLocalSymbols(name)
	})
	dir := t.TempDir()
	if base := os.Getenv("LOOM_691_ARTIFACTS"); base != "" {
		dir = filepath.Join(base, name)
		require.NoError(t, os.MkdirAll(dir, 0o750))
	}
	renderGRPCModule(t, dir, module, root, resolveGRPCLoomSource(t))
	services := CreateGRPCServices(root)
	var files []*codegen.File
	files = append(files, ClientCLIFiles(module+"/gen", services)...)
	files = append(files, servicecodegen.ExampleServiceFiles(module+"/gen", root, services.ServicesData)...)
	if examples {
		files = append(files, ExampleServerFiles(module+"/gen", services)...)
		files = append(files, ExampleCLIFiles(module+"/gen", services)...)
	}
	for _, file := range files {
		_, err := file.Render(dir)
		require.NoError(t, err, file.Path)
	}
	return dir
}
