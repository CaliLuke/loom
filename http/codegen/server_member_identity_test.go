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
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/testingx"
)

//go:embed testdata/server_member_identity_harness.go.txt
var httpServerMemberHarness string

func TestHTTPServerMemberIdentityGenerated(t *testing.T) {
	output := os.Getenv("LOOM_SERVER_MEMBER_OUTPUT")
	if output != "" {
		output = filepath.Join(output, "http")
	}
	batch := testingx.NewModuleBatch(t, checkoutPinnedLoomModule(t), output)
	for _, collisions := range []bool{false, true} {
		name := "ordinary"
		if collisions {
			name = "fixed-members"
		}
		t.Run(name, func(t *testing.T) {
			root := RunHTTPDSL(t, serverMemberIdentityDSL(collisions))
			assertHTTPServerMemberIdentity(t, root)
			c := batch.Case(t, name, "example.com/servermembers")
			renderHTTPModuleFiles(t, c.Dir, c.ModulePath+"/gen", root)
			if collisions {
				c.WriteFile(t, "members_test.go", httpServerMemberHarness)
			}
		})
	}
	if os.Getenv("LOOM_SERVER_MEMBER_GENERATE_ONLY") == "" {
		batch.Check(t)
	}
}

func assertHTTPServerMemberIdentity(t *testing.T, root *expr.RootExpr) {
	t.Helper()
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
