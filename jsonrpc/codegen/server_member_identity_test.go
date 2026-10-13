package codegen

import (
	_ "embed"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

//go:embed testdata/server_member_identity_harness.go.txt
var jsonrpcServerMemberHarness string

func TestJSONRPCServerMemberIdentityGenerated(t *testing.T) {
	for _, kind := range []string{"unary", "server", "bidi", "sse", "mixed", "keywords"} {
		for _, collisions := range []bool{false, true} {
			if kind == "keywords" && !collisions {
				continue
			}
			name := kind + "/ordinary"
			if collisions {
				name = kind + "/fixed-members"
			}
			t.Run(name, func(t *testing.T) {
				root := RunJSONRPCDSL(t, jsonrpcServerMemberIdentityDSL(kind, collisions))
				dir := t.TempDir()
				if output := os.Getenv("LOOM_SERVER_MEMBER_OUTPUT"); output != "" {
					dir = filepath.Join(output, "jsonrpc", name)
					require.NoError(t, os.MkdirAll(dir, 0o700))
				}
				renderJSONRPCModule(t, dir, "example.com/rpcmembers", root)
				if collisions {
					constructor := "srv := server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil)"
					if kind == "server" || kind == "bidi" || kind == "keywords" {
						constructor = `srv := server.New(func(ctx context.Context, stream members.Stream) error {
for {
if err := stream.Recv(ctx); err != nil {
return err
}
}
}, endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, &websocket.Upgrader{}, loomhttp.WebSocketConfig{})`
					}
					harness := strings.NewReplacer("@@NEW@@", constructor, "@@KIND@@", kind).Replace(jsonrpcServerMemberHarness)
					require.NoError(t, os.WriteFile(filepath.Join(dir, "members_test.go"), []byte(harness), 0o600))
				}
				if os.Getenv("LOOM_SERVER_MEMBER_GENERATE_ONLY") != "" {
					return
				}
				runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
				runGoJSONRPCTestCommand(t, dir, "build", "./...")
				runGoJSONRPCTestCommand(t, dir, "vet", "./...")
				if collisions {
					runGoJSONRPCTestCommand(t, dir, "test", "-race", "./...")
				}
			})
		}
	}
}

func TestJSONRPCServerMemberIdentity(t *testing.T) {
	for _, kind := range []string{"unary", "server", "bidi", "sse", "mixed", "keywords"} {
		for _, collisions := range []bool{false, true} {
			name := kind + "/ordinary"
			if collisions {
				name = kind + "/fixed-members"
			}
			t.Run(name, func(t *testing.T) {
				root := RunJSONRPCDSL(t, jsonrpcServerMemberIdentityDSL(kind, collisions))
				services := CreateJSONRPCServices(root)
				canonical := services.ServicesData.Get("members")
				names := make(map[string]string, len(canonical.Methods))
				for _, method := range canonical.Methods {
					names[method.Name] = method.VarName
				}
				data := services.Get("members")
				for _, endpoint := range data.Endpoints {
					require.Equal(t, names[endpoint.Method.Name], endpoint.Method.VarName, "transport member allocation must preserve canonical methods")
				}
				for _, endpoint := range data.Endpoints {
					require.True(t, token.IsIdentifier(endpoint.ServerHandler))
					if !collisions {
						preferred := endpoint.Method.VarName
						if kind == "server" || kind == "bidi" || kind == "keywords" {
							preferred = lowerInitial(preferred)
						}
						require.Equal(t, preferred, endpoint.ServerHandler)
					}
				}
				if collisions && (kind == "server" || kind == "bidi") {
					require.NotEqual(t, data.Endpoint("read").ServerEndpoint, data.Endpoint("readEndpoint").ServerHandler)
				}
				source := rpcServerMemberSource(t, data, kind)
				assertRPCServerMemberIdentity(t, source)
			})
		}
	}
}

func TestJSONRPCServerMemberReservationsFollowRenderedReceivers(t *testing.T) {
	for _, kind := range []string{"unary", "server", "bidi", "sse", "mixed"} {
		t.Run(kind, func(t *testing.T) {
			root := RunJSONRPCDSL(t, jsonrpcServerMemberIdentityDSL(kind, false))
			data := CreateJSONRPCServices(root).Get("members")
			members := assertRPCServerMemberIdentity(t, rpcServerMemberSource(t, data, kind))
			fixed := make(map[string]bool)
			for receiver, names := range members {
				if receiver != data.ServerStruct && receiver != lowerInitial(data.Service.StructName)+"Stream" {
					continue
				}
				for _, endpoint := range data.Endpoints {
					delete(names, endpoint.ServerHandler)
					delete(names, endpoint.ServerEndpoint)
				}
				for name := range names {
					fixed[name] = true
				}
			}
			var competitors []string
			for name := range fixed {
				if kind == "server" || kind == "bidi" {
					if name != lowerInitial(name) {
						continue
					}
				} else if !ast.IsExported(name) {
					continue
				}
				competitors = append(competitors, name)
			}
			slices.Sort(competitors)
			root = RunJSONRPCDSL(t, jsonrpcServerMemberIdentityDSL(kind, false, competitors...))
			data = CreateJSONRPCServices(root).Get("members")
			assertRPCServerMemberIdentity(t, rpcServerMemberSource(t, data, kind))
		})
	}
}

func rpcServerMemberSource(t *testing.T, data *httpcodegen.ServiceData, kind string) string {
	t.Helper()
	sections := jsonrpcServerBaseSections(data, kind == "sse", kind == "mixed")
	sections = append(sections, jsonrpcServerTransportSections(data, kind == "sse", kind == "mixed")...)
	sections = append(sections, jsonrpcServerMountSection(data, kind == "sse", kind == "mixed"))
	if kind == "server" || kind == "bidi" || kind == "keywords" {
		sections = append(sections, jsonrpcWebSocketServerSections(data)...)
	} else {
		sections = append(sections, jsonrpcServerEncodeErrorSection(data.ServerStruct))
	}
	var source strings.Builder
	source.WriteString("package server\n")
	for _, section := range sections {
		require.NoError(t, section.Write(&source))
	}
	return source.String()
}

func assertRPCServerMemberIdentity(t *testing.T, source string) map[string]map[string]bool {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "server.go", source, 0)
	require.NoError(t, err)
	members := make(map[string]map[string]bool)
	add := func(receiver, name string) {
		if members[receiver] == nil {
			members[receiver] = make(map[string]bool)
		}
		if members[receiver][name] {
			t.Errorf("receiver %q duplicates member %q", receiver, name)
		}
		members[receiver][name] = true
	}
	for _, declaration := range file.Decls {
		switch decl := declaration.(type) {
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				typ, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				structure, ok := typ.Type.(*ast.StructType)
				if !ok {
					continue
				}
				for _, field := range structure.Fields.List {
					if len(field.Names) == 0 {
						add(typ.Name.Name, field.Type.(*ast.SelectorExpr).Sel.Name)
					}
					for _, name := range field.Names {
						add(typ.Name.Name, name.Name)
					}
				}
			}
		case *ast.FuncDecl:
			if decl.Recv != nil {
				receiver := decl.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
				add(receiver, decl.Name.Name)
			}
		}
	}
	return members
}

func jsonrpcServerMemberIdentityDSL(kind string, collisions bool, extraNames ...string) func() {
	return func() {
		dsl.API("members", func() {
			dsl.JSONRPC(func() {
			})
		})
		dsl.Service("members", func() {
			dsl.JSONRPC(func() {
				if kind == "server" || kind == "bidi" || kind == "keywords" {
					dsl.GET("/rpc")
				} else {
					dsl.POST("/rpc")
				}
			})
			names := []string{"read", "use"}
			if collisions {
				names = append(names, "mount", "service", "methods", "methodNames", "handler", "decoder", "encoder", "errhandler", "conn", "cancel", "r", "w", "readEndpoint", "upgrader", "webSocketConfig", "streamHandler", "send", "sendError", "matchesMethod", "processRequest", "serveHTTP")
			}
			if kind == "keywords" && collisions {
				names = []string{"type", "func", "map", "read"}
			}
			names = append(names, extraNames...)
			for _, name := range names {
				dsl.Method(name, func() {
					payload := func() {
						dsl.Attribute("value", dsl.String)
					}
					if kind == "bidi" {
						dsl.StreamingPayload(payload)
					} else {
						dsl.Payload(payload)
					}
					if kind != "unary" && (kind != "mixed" || name != "read") {
						dsl.StreamingResult(dsl.String)
					} else {
						dsl.Result(dsl.String)
					}
					dsl.JSONRPC(func() {
						if kind == "sse" || (kind == "mixed" && name != "read") {
							dsl.ServerSentEvents()
						}
					})
				})
			}
		})
	}
}
