package testdata

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"github.com/CaliLuke/loom/dsl"
)

// ClientMemberIdentityDSL covers concrete transport receiver roles independently
// of canonical service and wire method identities.
func ClientMemberIdentityDSL(kind string) func() {
	return func() {
		rpc := strings.HasPrefix(kind, "rpc-")
		if kind == "http-fixed" || kind == "http-mixed-stream-collision" {
			dsl.Interceptor("audit")
		}
		dsl.API("members", func() {
			if rpc {
				dsl.JSONRPC(func() {
				})
			}
		})
		dsl.Service("members", func() {
			if kind == "http-fixed" || kind == "http-mixed-stream-collision" {
				dsl.ClientInterceptor("audit")
			}
			if rpc {
				dsl.JSONRPC(func() {
					if kind == "rpc-websocket" {
						dsl.GET("/rpc")
					} else {
						dsl.POST("/rpc")
					}
				})
			}
			for _, name := range clientMemberNames(kind) {
				clientMemberMethod(kind, name)
			}
		})
	}
}

// ClientMemberDeclarations parses emitted fields and methods by concrete receiver.
// It retains duplicates so callers can check uniqueness and complete role coverage.
func ClientMemberDeclarations(sources []string) (map[string][]string, error) {
	members := make(map[string][]string)
	add := func(receiver, name string) {
		members[receiver] = append(members[receiver], name)
	}
	for index, source := range sources {
		file, err := parser.ParseFile(token.NewFileSet(), fmt.Sprintf("client%d.go", index), source, 0)
		if err != nil {
			return nil, err
		}
		for _, declaration := range file.Decls {
			switch declaration := declaration.(type) {
			case *ast.FuncDecl:
				if declaration.Recv == nil {
					continue
				}
				receiver := declaration.Recv.List[0].Type
				if pointer, ok := receiver.(*ast.StarExpr); ok {
					receiver = pointer.X
				}
				if identifier, ok := receiver.(*ast.Ident); ok {
					add(identifier.Name, declaration.Name.Name)
				}
			case *ast.GenDecl:
				for _, spec := range declaration.Specs {
					typ, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					structure, ok := typ.Type.(*ast.StructType)
					if !ok {
						continue
					}
					for _, field := range structure.Fields.List {
						for _, name := range field.Names {
							add(typ.Name.Name, name.Name)
						}
					}
				}
			}
		}
	}
	return members, nil
}

func clientMemberNames(kind string) []string {
	names := []string{"close", "isClosed"}
	switch kind {
	case "http-ordinary":
		names = append(names, "doer")
	case "rpc-ordinary":
		names = append(names, "read", "readDoer")
	case "http-fixed", "http-grouped":
		names = []string{"restoreResponseBody", "restoreResponseBodyEndpoint", "read", "readDoer", "buildReadRequest", "readDoerDoer"}
		if kind == "http-grouped" {
			names = append(names, "write")
		}
	case "http-mixed":
		names = []string{"watch", "restoreResponseBody", "buildWatchRequest"}
	case "http-mixed-stream-collision":
		names = []string{"watch", "watchStream", "watchStreamEndpoint"}
	case "rpc-unary":
		names = []string{"doer", "restoreResponseBody", "doerEndpoint", "read", "buildReadRequest"}
	case "rpc-sse":
		names = []string{"watch", "watchDoer", "buildWatchRequest"}
	case "rpc-websocket":
		names = []string{"close", "isClosed", "closeEndpoint", "doer", "restoreResponseBody"}
	}
	return names
}

func clientMemberMethod(kind, name string) {
	dsl.Method(name, func() {
		switch kind {
		case "rpc-websocket", "http-websocket":
			dsl.StreamingPayload(dsl.String)
			dsl.Result(dsl.String)
		case "rpc-sse":
			dsl.Payload(dsl.String)
			dsl.StreamingResult(dsl.String)
		default:
			dsl.Payload(dsl.String)
			dsl.Result(dsl.String)
			if strings.HasPrefix(kind, "http-mixed") && name == "watch" {
				dsl.StreamingResult(dsl.String)
			}
		}
		if strings.HasPrefix(kind, "rpc-") {
			dsl.JSONRPC(func() {
				if kind == "rpc-sse" {
					dsl.ServerSentEvents()
				}
			})
		} else {
			dsl.HTTP(func() {
				path := "/" + name
				if kind == "http-grouped" {
					path = "/read/" + name
				}
				if kind == "http-websocket" {
					dsl.GET(path)
				} else {
					dsl.POST(path)
				}
				if strings.HasPrefix(kind, "http-mixed") && name == "watch" {
					dsl.ServerSentEvents()
				}
			})
		}
	})
}
