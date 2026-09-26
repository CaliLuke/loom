package testdatacompile

import (
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"

	"github.com/CaliLuke/loom/internal/loomsource"
)

type design struct {
	id         string
	importPath string
	name       string
}

var domains = []string{
	"codegen/service", "http/codegen", "grpc/codegen", "jsonrpc/codegen", "expr",
}

func discover(t *testing.T, source string) []design {
	t.Helper()
	patterns := make([]string, len(domains))
	for i, domain := range domains {
		patterns[i] = "./" + domain + "/testdata"
	}
	pkgs, err := packages.Load(&packages.Config{
		Dir: source, Env: append(os.Environ(), "GOWORK=off"), Mode: packages.NeedName | packages.NeedTypes | packages.NeedImports | packages.NeedDeps,
	}, patterns...)
	require.NoError(t, err)
	require.Len(t, pkgs, len(domains))
	var designs []design
	for _, pkg := range pkgs {
		require.Empty(t, pkg.Errors, "%s", pkg.PkgPath)
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			if !obj.Exported() {
				continue
			}
			switch obj.(type) {
			case *types.Func, *types.Var:
			default:
				continue
			}
			sig, ok := obj.Type().Underlying().(*types.Signature)
			if !ok || sig.Params().Len() != 0 || sig.Results().Len() != 0 {
				continue
			}
			domain := strings.TrimSuffix(strings.TrimPrefix(pkg.PkgPath, "github.com/CaliLuke/loom/"), "/testdata")
			designs = append(designs, design{id: domain + "/" + name, importPath: pkg.PkgPath, name: name})
		}
	}
	sort.Slice(designs, func(i, j int) bool {
		return designs[i].id < designs[j].id
	})
	return designs
}

// TestCatalog checks discovery coverage and validates every manifest entry.
func TestCatalog(t *testing.T) {
	root, err := loomsource.RepositoryRoot(".")
	require.NoError(t, err)
	designs := discover(t, root)
	seen := make(map[string]bool)
	for _, d := range designs {
		require.False(t, seen[d.id], "duplicate design %s", d.id)
		seen[d.id] = true
	}
	for _, domain := range domains {
		found := false
		for _, d := range designs {
			found = found || strings.HasPrefix(d.id, domain+"/")
		}
		require.True(t, found, "no designs found in %s", domain)
	}
	// Representative names prove that exported function-valued variables are
	// discovered, without pinning a count that would hide new designs.
	require.True(t, seen["jsonrpc/codegen/JSONRPCSSEStringDSL"])
	require.True(t, seen["expr/FilesTooManyArgErrorDSL"])
	validateExpectations(t, filepath.Join(root, "internal", "testdatacompile", "expectations.json"), seen)
}
