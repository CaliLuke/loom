package codegen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	codegentestdata "github.com/CaliLuke/loom/codegen/testdata"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestConstructorPackageInventory(t *testing.T) {
	for _, tc := range []struct {
		name   string
		design func()
	}{
		{"ordinary", func() {
			testdata.ConstructorOwnership("ordinary", false, false)
		}},
		{"streams", clientBodyInitNamesDSL},
		{"multipart", responseContractMultipartServerDSL},
		{"relocated-unions", codegentestdata.UserTypePackageTransportsDSL},
		{"viewed-sse", testdata.SSEViewedResultDSL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := RunHTTPDSL(t, tc.design)
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/inventory", root)
			services := CreateHTTPServices(root)
			for _, service := range services.Expressions.Services {
				sd := services.Get(service.Name())
				CheckConstructorPackageInventoryForTest(t, sd, filepath.Join(dir, "gen", "http", sd.Service.PathName))
			}
		})
	}
}

// CheckConstructorPackageInventoryForTest compares actual declarations with the
// shared allocation inventory. External tests use it for JSON-RPC renderers
// without exposing analysis internals in the production API.
func CheckConstructorPackageInventoryForTest(t *testing.T, sd *ServiceData, path string) {
	t.Helper()
	for _, client := range []bool{false, true} {
		pkg := "server"
		if client {
			pkg = "client"
		}
		names := sd.constructorReservations(client)
		sd.walkConstructors(client, func(init *InitData, _ bool) {
			require.True(t, strings.HasPrefix(init.Name, "New"), "conversion namespace: %s", init.Name)
			names[init.Name] = true
		})
		entries, err := os.ReadDir(filepath.Join(path, pkg))
		require.NoError(t, err)
		for _, entry := range entries {
			if filepath.Ext(entry.Name()) != ".go" {
				continue
			}
			f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(path, pkg, entry.Name()), nil, 0)
			require.NoError(t, err)
			for _, id := range packageDeclarationNames(f) {
				// Conversion builders all use New-prefixed identifiers. Declarations
				// outside that namespace cannot compete, including RPC wrappers.
				if strings.HasPrefix(id, "New") && !names[id] {
					t.Errorf("%s/%s: unreserved %s", pkg, entry.Name(), id)
				}
			}
		}
	}
}

func packageDeclarationNames(file *ast.File) []string {
	var names []string
	for _, decl := range file.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			if decl.Recv == nil {
				names = append(names, decl.Name.Name)
			}
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					names = append(names, spec.Name.Name)
				case *ast.ValueSpec:
					for _, name := range spec.Names {
						names = append(names, name.Name)
					}
				}
			}
		}
	}
	return names
}
