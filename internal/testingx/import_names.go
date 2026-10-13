package testingx

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/CaliLuke/loom/internal/naming"
)

// ServicePackageImportNames returns sorted external import aliases that share a
// generated Go file with a service or views import identified by genpkg. It skips
// service/view declarations themselves, blank/dot imports and imports within the
// application module (the parent of genpkg). Imports in separate files do not
// share this collision boundary, even if the files belong to the same package.
//
// The controlled transport fixtures using this helper have no authored external
// custom types. Only under that assumption do these aliases identify framework
// imports: rendered syntax cannot establish the provenance of arbitrary external
// imports. Default names use Loom's import-name convention; explicit aliases are
// preserved. This is representative fixture coverage, not a template inventory.
func ServicePackageImportNames(t testing.TB, dir, genpkg string) []string {
	t.Helper()
	module := path.Dir(genpkg)
	names := make(map[string]struct{})
	err := filepath.WalkDir(dir, func(filename string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(filename, ".go") {
			return err
		}
		if isServicePackageDir(dir, filepath.Dir(filename), genpkg) {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), filename, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		if len(serviceImportNames(file, genpkg)) == 0 {
			return nil
		}
		for _, spec := range file.Imports {
			importPath, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return err
			}
			if importPath == module || strings.HasPrefix(importPath, module+"/") {
				continue
			}
			var explicit string
			if spec.Name != nil {
				explicit = spec.Name.Name
			}
			if explicit != "_" && explicit != "." {
				names[naming.ImportName(importPath, explicit)] = struct{}{}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parse generated imports: %v", err)
	}
	result := make([]string, 0, len(names))
	for name := range names {
		result = append(result, name)
	}
	slices.Sort(result)
	return result
}
