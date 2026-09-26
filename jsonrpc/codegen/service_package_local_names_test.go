package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cg "github.com/CaliLuke/loom/codegen"
	servicecodegen "github.com/CaliLuke/loom/codegen/service"
	. "github.com/CaliLuke/loom/dsl"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
	"github.com/CaliLuke/loom/internal/testingx"
)

// localNameMarker marks every name that the local name designs give to
// generated code, so that the names of the generated code alone remain.
const localNameMarker = "zz"

// localNameDesigns are the JSON-RPC designs whose generated transport and
// example files the local name tests inspect and compile.
var localNameDesigns = []struct {
	Name   string
	Design func(services ...string)
}{
	{Name: "http-sse", Design: localNameSSEDSL},
	{Name: "websocket", Design: localNameWebSocketDSL},
}

// TestServicePackageNamedLikeLocalGeneratedModuleBuilds collects the
// receivers, parameters, locals and package-level names that the generated
// JSON-RPC transport and example files declare where they would hide a
// service package imported under its own name. It checks that services
// named after each of them import their package under an alias and that the
// generated module and example compile.
func TestServicePackageNamedLikeLocalGeneratedModuleBuilds(t *testing.T) {
	for _, c := range localNameDesigns {
		t.Run(c.Name, func(t *testing.T) {
			const modulePath = "example.com/localnames"
			dir := t.TempDir()
			renderLocalNameModule(t, dir, modulePath, c.Design, localNameMarker+"svc")
			names := testingx.ServicePackageShadowNames(t, dir, modulePath+"/gen", localNameMarker)
			require.NotEmpty(t, names)

			root := RunJSONRPCDSL(t, func() {
				c.Design(names...)
			})
			services := CreateJSONRPCServices(root)
			for _, name := range names {
				assert.NotEqualf(t, name, services.Get(name).Service.PkgName, "service %q is imported under its own name", name)
			}

			sweep := t.TempDir()
			renderLocalNameModule(t, sweep, "example.com/localnamesweep", c.Design, names...)
			runGoJSONRPCTestCommand(t, sweep, "mod", "tidy")
			runGoJSONRPCTestCommand(t, sweep, "vet", "./...")
		})
	}
}

// renderLocalNameModule renders the JSON-RPC transport, the views, the client
// CLI and the example server and CLI of design with the given services to
// dir.
func renderLocalNameModule(t *testing.T, dir, modulePath string, design func(...string), services ...string) {
	t.Helper()
	root := RunJSONRPCDSL(t, func() {
		design(services...)
	})
	renderJSONRPCModule(t, dir, modulePath, root)
	genpkg := modulePath + "/gen"
	data := CreateJSONRPCServices(root)
	httpData := httpcodegen.NewServicesData(data.ServicesData, root.API.HTTP)
	for _, svc := range root.Services {
		if views := servicecodegen.ViewsFile(genpkg, svc, data.ServicesData); views != nil {
			renderCodegenFiles(t, dir, []*cg.File{views})
		}
	}
	renderCodegenFiles(t, dir, ClientCLIFiles(genpkg, data))
	renderCodegenFiles(t, dir, ExampleServerFiles(genpkg, data, httpcodegen.ExampleServerFiles(genpkg, httpData)))
	renderCodegenFiles(t, dir, ExampleCLIFiles(genpkg, data))
}

// localNameTypes declares the types of the local name designs.
func localNameTypes() (frame, item any) {
	frame = Type("ZzFrame", func() {
		Attribute("zzvalue", String)
	})
	item = ResultType("application/vnd.zzitem", func() {
		TypeName("ZzItem")
		Attributes(func() {
			Attribute("zzid", String)
			Attribute("zzname", String)
		})
		View("default", func() {
			Attribute("zzid")
			Attribute("zzname")
		})
	})
	return frame, item
}

// localNameSSEDSL declares JSON-RPC services with the given names with unary
// methods that take and return objects, arrays, errors and security, a
// notification and server-sent events methods. Every name it gives to
// generated code contains localNameMarker.
func localNameSSEDSL(services ...string) {
	frame, item := localNameTypes()
	var JWTAuth = JWTSecurity("zzjwt", func() {
		Scope("zz:read")
	})
	API("zzapi", func() {
		JSONRPC(func() {})
	})
	for _, name := range services {
		Service(name, func() {
			JSONRPC(func() {
				POST("/rpc/" + name)
			})
			Method("zzadd", func() {
				Payload(func() {
					ID("zzid", String)
					Attribute("zza", Int)
					Attribute("zzm", MapOf(String, Int))
				})
				Result(func() {
					ID("zzid", String)
					Attribute("zzsum", Int)
				})
				Error("zzbad")
				JSONRPC(func() {
					Response("zzbad", 1)
				})
			})
			Method("zzsecure", func() {
				Security(JWTAuth)
				Payload(func() {
					ID("zzid", String)
					Token("zztoken", String)
				})
				Result(item)
				JSONRPC(func() {})
			})
			Method("zzlist", func() {
				Result(ArrayOf(frame))
				JSONRPC(func() {})
			})
			Method("zznotify", func() {
				Payload(func() {
					Attribute("zza", Int)
				})
				JSONRPC(func() {})
			})
			Method("zzwatch", func() {
				Payload(func() {
					ID("zzid", String)
				})
				StreamingResult(func() {
					ID("zzid", String)
					Attribute("zzv", String)
				})
				Error("zzgone")
				JSONRPC(func() {
					ServerSentEvents()
					Response("zzgone", 2)
				})
			})
			Method("zzitems", func() {
				Payload(func() {
					ID("zzid", String)
				})
				StreamingResult(frame)
				JSONRPC(func() {
					ServerSentEvents()
				})
			})
		})
	}
}

// localNameWebSocketDSL declares JSON-RPC services with the given names
// served over a WebSocket with bidirectional, client streaming, server
// streaming and notification streaming methods. Every name it gives to
// generated code contains localNameMarker.
func localNameWebSocketDSL(services ...string) {
	frame, _ := localNameTypes()
	API("zzapi", func() {
		JSONRPC(func() {})
	})
	for _, name := range services {
		Service(name, func() {
			JSONRPC(func() {
				GET("/rpc/" + name)
			})
			Method("zzexchange", func() {
				StreamingPayload(frame)
				StreamingResult(frame)
				Error("zzbad")
				JSONRPC(func() {
					Response("zzbad", 1)
				})
			})
			Method("zzupload", func() {
				StreamingPayload(frame)
				Result(frame)
				JSONRPC(func() {})
			})
			Method("zzfeed", func() {
				Payload(func() {
					ID("zzid", String)
				})
				StreamingResult(frame)
				JSONRPC(func() {})
			})
			Method("zzping", func() {
				StreamingPayload(frame)
				JSONRPC(func() {})
			})
		})
	}
}
