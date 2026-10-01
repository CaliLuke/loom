package codegen

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	servicecodegen "github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

func TestGRPCServiceImportAliases(t *testing.T) {
	root := RunGRPCDSL(t, testdata.GRPCServiceImportAliasesDSL)
	services := CreateGRPCServices(root)
	for _, tc := range []struct {
		name  string
		alias string
	}{
		{"ordinary", "ordinary"},
		{"protojson", "protojsonsvc"},
		{"protojsonsvc", "protojsonsvcsvc"},
		{"strconv", "strconvsvc"},
		{"metadata", "metadatasvc"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			neutral := services.ServicesData.Get(tc.name)
			data := services.Get(tc.name)
			require.Equal(t, tc.name, neutral.PkgName)
			require.Equal(t, tc.alias, data.Service.PkgName)
			require.Equal(t, tc.alias, data.Endpoint("send").ServicePkgName)
			require.Contains(t, data.Endpoint("send").PayloadRef, tc.alias+".")
			require.Contains(t, data.Endpoint("watch").ServerStream.SendRef, tc.alias+".")
		})
	}
}

func TestGRPCServiceImportAliasAvoidsClientReceiver(t *testing.T) {
	root := RunGRPCDSL(t, func() {
		dsl.Service("c", func() {
			dsl.Method("show", func() {
				dsl.Payload(func() {
					dsl.Field(1, "id", dsl.String)
				})
				dsl.GRPC(func() {})
			})
		})
	})

	data := CreateGRPCServices(root).Get("c")
	require.Equal(t, "csvc", data.Service.PkgName)
	require.Contains(t, data.Endpoint("show").PayloadRef, "csvc.")
}

func TestGRPCProtobufImportAvoidsFrameworkPackage(t *testing.T) {
	root := RunGRPCDSL(t, testdata.GRPCServiceImportAliasesDSL)
	data := CreateGRPCServices(root).Get("loom")
	require.Equal(t, "loompb2", data.PkgName)
	require.Equal(t, "loom", data.ProtoPkg)
	require.Equal(t, "loom", data.Service.PathName)
}

func TestGRPCServiceImportAliasDeclarationOrders(t *testing.T) {
	for _, names := range [][]string{
		{"protojson", "protojsonsvc", "protojsonsvc2", "ordinary"},
		{"loom", "loompb", "loompb2", "ordinary"},
	} {
		var check func(int)
		check = func(index int) {
			if index < len(names) {
				for other := index; other < len(names); other++ {
					names[index], names[other] = names[other], names[index]
					check(index + 1)
					names[index], names[other] = names[other], names[index]
				}
				return
			}
			t.Run(strings.Join(names, "-"), func(t *testing.T) {
				root := RunGRPCDSL(t, func() {
					testdata.GRPCServiceImportAliases(names...)
				})
				services := CreateGRPCServices(root)
				used := map[string]bool{"protojson": true, "loompb": true}
				for _, name := range names {
					alias := services.Get(name).PkgName
					require.False(t, used[alias], "duplicate protobuf alias %q", alias)
					used[alias] = true
				}
				for _, name := range names {
					alias := services.Get(name).Service.PkgName
					require.False(t, used[alias], "duplicate alias %q", alias)
					used[alias] = true
					require.Equal(t, name, services.ServicesData.Get(name).PkgName)
				}
			})
		}
		check(0)
	}
}

func TestGRPCServiceImportReservationsCoverGeneratedImports(t *testing.T) {
	const genpkg = "example.com/importcoverage/gen"
	for _, tc := range []struct {
		name   string
		design func()
	}{
		{"aliases", testdata.GRPCServiceImportAliasesDSL},
		{"protobuf-json", testdata.CLIProtoJSONDSL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := RunGRPCDSL(t, tc.design)
			services := CreateGRPCServices(root)
			files := append(ClientFiles(genpkg, services), ServerFiles(genpkg, services)...)
			files = append(files, ClientTypeFiles(genpkg, services)...)
			files = append(files, ServerTypeFiles(genpkg, services)...)
			files = append(files, ClientCLIFiles(genpkg, services)...)
			files = append(files, ExampleServerFiles(genpkg, services)...)
			files = append(files, ExampleCLIFiles(genpkg, services)...)
			files = append(files, ResponseContractTestFiles(genpkg, services)...)
			for _, file := range files {
				header := codegen.HeaderDataForSection(file.HeaderSection())
				require.NotNil(t, header, file.Path)
				for _, spec := range header.Imports {
					if strings.HasPrefix(spec.Path, "example.com/importcoverage") {
						continue
					}
					name := spec.Name
					if name == "" {
						name = filepath.Base(spec.Path)
					}
					require.Contains(t, transportGeneratedImportNames, name, "%s imports %s", file.Path, spec.Path)
				}
			}
			for _, service := range root.Services {
				require.False(t, slices.Contains(transportGeneratedImportNames, services.Get(service.Name).Service.PkgName))
			}
		})
	}
}

func TestGRPCServiceImportAliasGeneratedModule(t *testing.T) {
	const module = "example.com/importaliases"
	root := RunGRPCDSL(t, testdata.GRPCServiceImportAliasesDSL)
	dir := t.TempDir()
	renderGRPCModule(t, dir, module, root, resolveGRPCLoomSource(t))
	services := CreateGRPCServices(root)
	for _, file := range ClientCLIFiles(module+"/gen", services) {
		_, err := file.Render(dir)
		require.NoError(t, err)
	}
	for _, file := range servicecodegen.ExampleServiceFiles(module+"/gen", root, services.ServicesData) {
		_, err := file.Render(dir)
		require.NoError(t, err)
	}
	for _, file := range ExampleServerFiles(module+"/gen", services) {
		header := codegen.HeaderDataForSection(file.HeaderSection())
		used := make(map[string]bool)
		for _, spec := range header.Imports {
			name := spec.Name
			if name == "" {
				name = filepath.Base(spec.Path)
			}
			require.False(t, used[name], "duplicate example import %q", name)
			used[name] = true
		}
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cli_test.go"), []byte(grpcImportAliasHarness), 0o600))
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "vet", "./...")
	runGRPCGoCommand(t, dir, "test", "-race", "-run", "^TestCLIUnion$", "./...")
}

const grpcImportAliasHarness = `package importaliases_test

import (
 "testing"
 "github.com/stretchr/testify/require"
 protojsonclient "example.com/importaliases/gen/grpc/protojson/client"
 strconvclient "example.com/importaliases/gen/grpc/strconv/client"
)

func TestCLIUnion(t *testing.T) {
 a, err := protojsonclient.BuildSendPayload("{\"leaf\":{\"name\":\"accepted\"}}")
 require.NoError(t, err)
 require.NotNil(t, a)
 leaf, ok := a.AsLeaf()
 require.True(t, ok)
 require.Equal(t, "accepted", *leaf.Name)
 b, err := strconvclient.BuildSendPayload("{\"other\":{\"count\":17}}")
 require.NoError(t, err)
 require.NotNil(t, b)
 other, ok := b.AsOther()
 require.True(t, ok)
 require.Equal(t, 17, *other.Count)
 _, err = protojsonclient.BuildSendPayload("{")
 require.Error(t, err)
}
`

func TestGRPCServiceImportAliasCustomTypesCompile(t *testing.T) {
	const module = "example.com/customaliases"
	root := RunGRPCDSL(t, testdata.GRPCServiceImportAliasCustomTypeDSL)
	dir := t.TempDir()
	renderGRPCModule(t, dir, module, root, resolveGRPCLoomSource(t))
	services := CreateGRPCServices(root)
	for _, service := range root.Services {
		require.NoError(t, servicecodegen.SetUserTypeImports(module+"/gen", services.ServicesData.Get(service.Name)))
	}
	for _, file := range ClientCLIFiles(module+"/gen", services) {
		_, err := file.Render(dir)
		require.NoError(t, err)
	}
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "vet", "./...")
}

func TestGRPCExampleServerAPIImportAvoidsFrameworkNames(t *testing.T) {
	for _, name := range []string{"context", "sync", "net"} {
		t.Run(name, func(t *testing.T) {
			root := RunGRPCDSL(t, func() {
				dsl.API(name, func() {})
				dsl.Service("ordinary", func() {
					dsl.Method("ping", func() {
						dsl.Result(dsl.String)
						dsl.GRPC(func() {})
					})
				})
			})
			files := ExampleServerFiles("example.com/serveraliases/gen", CreateGRPCServices(root))
			require.Len(t, files, 1)
			header := codegen.HeaderDataForSection(files[0].HeaderSection())
			var alias string
			for _, spec := range header.Imports {
				if spec.Path == "example.com/serveraliases" {
					alias = spec.Name
				}
			}
			require.Equal(t, name+"api", alias)
		})
	}
}

func TestGRPCMetadataImportAliases(t *testing.T) {
	root := RunGRPCDSL(t, testdata.GRPCMetadataImportAliasesDSL)
	services := CreateGRPCServices(root)
	for _, name := range []string{"protojson", "loom", "ordinary"} {
		t.Run(name, func(t *testing.T) {
			data := services.Get(name)
			endpoint := data.Endpoint("send")
			for _, group := range [][]*MetadataData{endpoint.Request.Metadata, append(slices.Clone(endpoint.Response.Headers), endpoint.Response.Trailers...)} {
				used := map[string]bool{data.Service.PkgName: true, data.PkgName: true}
				for _, item := range group {
					require.False(t, used[item.VarName], "metadata variable shadows import or another local: %s", item.VarName)
					used[item.VarName] = true
				}
			}
			require.Equal(t, name, services.ServicesData.Get(name).PkgName)
		})
	}
}

func TestGRPCMetadataImportAliasesGeneratedModule(t *testing.T) {
	const module = "example.com/metadataaliases"
	root := RunGRPCDSL(t, testdata.GRPCMetadataImportAliasesDSL)
	dir := t.TempDir()
	renderGRPCModule(t, dir, module, root, resolveGRPCLoomSource(t))
	services := CreateGRPCServices(root)
	for _, service := range root.Services {
		require.NoError(t, servicecodegen.SetUserTypeImports(module+"/gen", services.ServicesData.Get(service.Name)))
	}
	for _, file := range ClientCLIFiles(module+"/gen", services) {
		_, err := file.Render(dir)
		require.NoError(t, err)
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "metadata_test.go"), []byte(grpcMetadataAliasHarness), 0o600))
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "vet", "./...")
	runGRPCGoCommand(t, dir, "test", "-race", "-run", "^TestMetadataAliases$", "./...")
}

const grpcMetadataAliasHarness = `package metadataaliases_test

import (
 "context"
 "reflect"
 "testing"
 "github.com/stretchr/testify/require"
 "google.golang.org/grpc/metadata"
 protojsonclient "example.com/metadataaliases/gen/grpc/protojson/client"
 protojsonserver "example.com/metadataaliases/gen/grpc/protojson/server"
 loomclient "example.com/metadataaliases/gen/grpc/loom/client"
 loomserver "example.com/metadataaliases/gen/grpc/loom/server"
 ordinaryclient "example.com/metadataaliases/gen/grpc/ordinary/client"
 ordinaryserver "example.com/metadataaliases/gen/grpc/ordinary/server"
 )

func TestMetadataAliases(t *testing.T) {
 for _, tc := range []struct {
  name string
  build func() (any, error)
  encodeRequest func(context.Context, any, *metadata.MD) (any, error)
  decodeRequest func(context.Context, any, metadata.MD) (any, error)
  encodeResponse func(context.Context, any, *metadata.MD, *metadata.MD) (any, error)
  decodeResponse func(context.Context, any, metadata.MD, metadata.MD) (any, error)
 }{
  {"protojson", func() (any, error) { return protojsonclient.BuildSendPayload("{\"body\":{\"value\":\"body value\"}}", "a", "b", "c", "d", "e", "f") },
   protojsonclient.EncodeSendRequest, protojsonserver.DecodeSendRequest,
   protojsonserver.EncodeSendResponse, protojsonclient.DecodeSendResponse},
  {"loom", func() (any, error) { return loomclient.BuildSendPayload("{\"body\":{\"value\":\"body value\"}}", "a", "b", "c", "d", "e", "f") },
   loomclient.EncodeSendRequest, loomserver.DecodeSendRequest,
   loomserver.EncodeSendResponse, loomclient.DecodeSendResponse},
  {"ordinary", func() (any, error) { return ordinaryclient.BuildSendPayload("{\"body\":{\"value\":\"body value\"}}", "a", "b", "c", "d", "e", "f") },
   ordinaryclient.EncodeSendRequest, ordinaryserver.DecodeSendRequest,
   ordinaryserver.EncodeSendResponse, ordinaryclient.DecodeSendResponse},
 } {
  t.Run(tc.name, func(t *testing.T) {
   ctx := context.Background()
   payload, err := tc.build()
   require.NoError(t, err)
   body := reflect.ValueOf(payload).Elem().FieldByName("Body").Elem()
   require.Equal(t, "body value", body.FieldByName("Value").Elem().String())
   requestMetadata := metadata.MD{}
   message, err := tc.encodeRequest(ctx, payload, &requestMetadata)
   require.NoError(t, err)
   expected := metadata.Pairs("protojsonsvc", "a", "protojsonsvc2", "b", "protojsonsvc3", "c", "loompb2", "d", "loompb22", "e", "loompb3", "f")
   require.Equal(t, expected, requestMetadata)
   decoded, err := tc.decodeRequest(ctx, message, requestMetadata)
   require.NoError(t, err)
   require.Equal(t, payload, decoded)
   headers, trailers := metadata.MD{}, metadata.MD{}
   response, err := tc.encodeResponse(ctx, decoded, &headers, &trailers)
   require.NoError(t, err)
   require.Equal(t, metadata.Pairs("protojsonsvc", "a", "protojsonsvc2", "b", "protojsonsvc3", "c"), headers)
   require.Equal(t, metadata.Pairs("loompb2", "d", "loompb22", "e", "loompb3", "f"), trailers)
   result, err := tc.decodeResponse(ctx, response, headers, trailers)
   require.NoError(t, err)
   require.Equal(t, payload, result)
  })
 }
}
`
