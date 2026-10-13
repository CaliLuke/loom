package codegen

import (
	_ "embed"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
)

//go:embed testdata/server_member_identity_harness.go.txt
var httpServerMemberHarness string

func TestHTTPServerMemberIdentity(t *testing.T) {
	for _, collisions := range []bool{false, true} {
		name := "ordinary"
		if collisions {
			name = "fixed-members"
		}
		t.Run(name, func(t *testing.T) {
			root := RunHTTPDSL(t, serverMemberIdentityDSL(collisions))
			services := CreateHTTPServices(root)
			canonical := services.ServicesData.Get("members")
			names := make(map[string]string, len(canonical.Methods))
			for _, method := range canonical.Methods {
				names[method.Name] = method.VarName
			}
			data := services.Get("members")
			for _, endpoint := range data.Endpoints {
				require.Equal(t, names[endpoint.Method.Name], endpoint.Method.VarName, "transport member allocation must preserve canonical methods")
			}
			source := "package server\n"
			for _, section := range []codegen.Section{serverStructSection(data), serverServiceSection(data), serverUseSection(data), serverMethodNamesSection(data), serverMountSection(data)} {
				source += codegen.SectionCode(t, section)
			}
			file, err := parser.ParseFile(token.NewFileSet(), "server.go", source, 0)
			require.NoError(t, err)
			members := make(map[string]bool)
			for _, decl := range file.Decls {
				if method, ok := decl.(*ast.FuncDecl); ok && method.Recv != nil {
					if members[method.Name.Name] {
						t.Errorf("server member %q is declared more than once", method.Name.Name)
					}
					members[method.Name.Name] = true
				}
				group, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}
				for _, spec := range group.Specs {
					typ, ok := spec.(*ast.TypeSpec)
					if !ok || typ.Name.Name != data.ServerStruct {
						continue
					}
					fields := typ.Type.(*ast.StructType).Fields.List
					for _, field := range fields {
						for _, fieldName := range field.Names {
							if members[fieldName.Name] {
								t.Errorf("server member %q is declared more than once", fieldName.Name)
							}
							members[fieldName.Name] = true
						}
					}
				}
			}
			for _, endpoint := range data.Endpoints {
				delete(members, endpoint.ServerHandler)
			}
			for _, fs := range data.FileServers {
				delete(members, fs.VarName)
			}
			fixed := fixedServerMembers(data)
			require.Len(t, members, len(fixed))
			for _, name := range fixed {
				require.True(t, members[name], "reservation %q must match an emitted member", name)
			}
		})
	}
}

func TestHTTPServerMemberIdentityGenerated(t *testing.T) {
	for _, collisions := range []bool{false, true} {
		name := "ordinary"
		if collisions {
			name = "fixed-members"
		}
		t.Run(name, func(t *testing.T) {
			root := RunHTTPDSL(t, serverMemberIdentityDSL(collisions))
			dir := t.TempDir()
			if output := os.Getenv("LOOM_SERVER_MEMBER_OUTPUT"); output != "" {
				dir = filepath.Join(output, "http", name)
				require.NoError(t, os.MkdirAll(dir, 0o700))
			}
			renderHTTPModule(t, dir, "example.com/servermembers", root)
			if collisions {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "members_test.go"), []byte(httpServerMemberHarness), 0o600))
			}
			if os.Getenv("LOOM_SERVER_MEMBER_GENERATE_ONLY") != "" {
				return
			}
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "build", "./...")
			runGoCommand(t, dir, "vet", "./...")
			if collisions {
				runGoCommand(t, dir, "test", "-race", "./...")
			}
		})
	}
}

func serverMemberIdentityDSL(collisions bool) func() {
	return func() {
		dsl.Service("members", func() {
			names := []string{"read", "use"}
			if collisions {
				names = append(names, "mount", "service", "mounts", "methodNames", "mountHandler", "serviceHandler")
				dsl.Files("/static", "Read")
				dsl.HTTP(func() {
					dsl.RuntimeCORS()
				})
			}
			for _, name := range names {
				dsl.Method(name, func() {
					dsl.Payload(dsl.String)
					dsl.Result(dsl.String)
					dsl.HTTP(func() {
						dsl.POST("/" + name)
						if collisions && name == "mount" {
							dsl.OPTIONS("/" + name)
						}
					})
				})
			}
		})
	}
}
