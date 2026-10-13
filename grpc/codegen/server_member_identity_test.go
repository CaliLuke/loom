package codegen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	servicecodegen "github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/dsl"
)

type grpcMemberCase struct {
	name    string
	methods []string
	stream  bool
	secure  bool
	located bool
}

func TestGRPCServerMemberIdentityGenerated(t *testing.T) {
	for _, tc := range grpcMemberCases() {
		t.Run(tc.name, func(t *testing.T) {
			root := RunGRPCDSL(t, grpcMemberDesign(tc.methods, tc.stream, tc.secure, tc.located))
			dir := t.TempDir()
			if output := os.Getenv("LOOM_GRPC_MEMBER_OUTPUT"); output != "" {
				dir = filepath.Join(output, tc.name)
				require.NoError(t, os.MkdirAll(dir, 0o700))
			}
			renderGRPCModule(t, dir, "example.com/grpcmembers", root, resolveGRPCLoomSource(t))
			services := CreateGRPCServices(root)
			require.NoError(t, servicecodegen.SetUserTypeImports("example.com/grpcmembers/gen", services.ServicesData.Get("members")))
			data := services.Get("members")
			assertServerPackageInventory(t, dir, append(serverPackageNames(data, root.API.GRPC.Services[0]), data.serverEmbeddingAlias))
			mounted := false
			if harness, err := os.ReadFile(filepath.Join("testdata", "server_members", tc.name+".go.txt")); err == nil {
				common, err := os.ReadFile(filepath.Join("testdata", "server_members", "common.go.txt"))
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(dir, "common_test.go"), common, 0o600))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "members_test.go"), harness, 0o600))
				mounted = true
			} else {
				require.ErrorIs(t, err, os.ErrNotExist)
			}
			runGRPCGoCommand(t, dir, "mod", "tidy")
			runGRPCGoCommand(t, dir, "build", "./...")
			runGRPCGoCommand(t, dir, "vet", "./...")
			if mounted {
				runGRPCGoCommand(t, dir, "test", "-race", "-count=1", ".")
			}
		})
	}
}

func TestGRPCServerMemberDeclarations(t *testing.T) {
	for _, tc := range grpcMemberCases() {
		t.Run(tc.name, func(t *testing.T) {
			root := RunGRPCDSL(t, grpcMemberDesign(tc.methods, tc.stream, tc.secure, tc.located))
			services := CreateGRPCServices(root)
			data := services.Get("members")
			neutral := services.ServicesData.Get("members")
			source := "package server\n" + sectionCode(t, grpcServerStructSection(data))
			for _, e := range data.Endpoints {
				source += "\n" + sectionCode(t, grpcServerInterfaceSection(e))
				require.Equal(t, neutral.Method(e.Method.Name).VarName, e.Method.VarName)
				require.Equal(t, protoServiceName(e.Method.VarName), e.RPCName)
			}
			file, err := parser.ParseFile(token.NewFileSet(), "server.go", source, 0)
			require.NoError(t, err)
			members := make(map[string]string)
			add := func(name, kind string) {
				if previous, exists := members[name]; exists {
					t.Errorf("receiver member %q collides: %s and %s", name, previous, kind)
				}
				members[name] = kind
			}
			for _, decl := range file.Decls {
				switch decl := decl.(type) {
				case *ast.GenDecl:
					for _, spec := range decl.Specs {
						typ, ok := spec.(*ast.TypeSpec)
						if !ok || typ.Name.Name != data.ServerStruct {
							continue
						}
						fields := typ.Type.(*ast.StructType).Fields.List
						for _, field := range fields {
							if len(field.Names) > 0 {
								for _, name := range field.Names {
									add(name.Name, "field")
								}
							} else {
								switch typ := field.Type.(type) {
								case *ast.SelectorExpr:
									add(typ.Sel.Name, "embedding")
								case *ast.Ident:
									add(typ.Name, "embedding")
								default:
									t.Errorf("unexpected embedding %T", field.Type)
								}
							}
						}
					}
				case *ast.FuncDecl:
					if decl.Recv != nil {
						add(decl.Name.Name, "RPC")
					}
				}
			}
		})
	}
}

func grpcMemberCases() []grpcMemberCase {
	return []grpcMemberCase{
		{name: "ordinary", methods: []string{"mount", "service", "use"}},
		{name: "handler-suffix", methods: []string{"mount", "mountH"}},
		{name: "fallback-suffix", methods: []string{"mount", "mountH", "mountHHandler", "mountHHandler2"}},
		{name: "embedded-name", methods: []string{"UnimplementedMembersServer"}},
		{name: "stream-handler", methods: []string{"mount", "mountH"}, stream: true},
		{name: "admission-control", methods: []string{"upload"}, secure: true},
		{name: "admission-handler", methods: []string{"mount", "mountH"}, secure: true},
		{name: "embedding-import", methods: []string{"UnimplementedMembersServer"}, stream: true, located: true},
	}
}

func grpcMemberDesign(names []string, stream, secure, located bool) func() {
	return func() {
		var result any = dsl.String
		var payload any = dsl.String
		if located {
			payload = dsl.Type("Nested", func() {
				dsl.Meta("struct:pkg:path", "unimplementedMembersServerEmbedded")
				dsl.Field(1, "value", dsl.String)
			})
			result = dsl.Type("Item", func() {
				dsl.Meta("struct:pkg:path", "unimplementedMembersServer")
				dsl.Field(1, "value", dsl.String)
			})
		}
		key := dsl.APIKeySecurity("member_key", func() {
		})
		dsl.Service("members", func() {
			for _, method := range names {
				dsl.Method(method, func() {
					if secure {
						dsl.Security(key)
						dsl.Payload(func() {
							dsl.APIKey("member_key", "key", dsl.String)
							dsl.Field(1, "name", dsl.String)
							dsl.Required("key", "name")
						})
						dsl.StreamingPayload(dsl.String)
					} else {
						dsl.Payload(payload)
					}
					if stream {
						dsl.StreamingResult(result)
					} else {
						dsl.Result(dsl.String)
					}
					dsl.GRPC(func() {
					})
				})
			}
		})
	}
}

// assertServerPackageInventory checks the alias crossing against actual emitted
// declarations, independently from the analysis inventory used to allocate it.
func assertServerPackageInventory(t *testing.T, dir string, names []string) {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, "gen", "grpc", "members", "server", "*.go"))
	require.NoError(t, err)
	require.NotEmpty(t, paths)
	for _, path := range paths {
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		require.NoError(t, err)
		for _, decl := range file.Decls {
			switch decl := decl.(type) {
			case *ast.FuncDecl:
				if decl.Recv == nil {
					require.Contains(t, names, decl.Name.Name, path)
				}
			case *ast.GenDecl:
				for _, spec := range decl.Specs {
					switch spec := spec.(type) {
					case *ast.TypeSpec:
						require.Contains(t, names, spec.Name.Name, path)
					case *ast.ValueSpec:
						for _, name := range spec.Names {
							require.Contains(t, names, name.Name, path)
						}
					}
				}
			}
		}
	}
}
