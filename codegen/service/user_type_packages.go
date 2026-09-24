package service

import (
	"path"
	"slices"
	"strings"

	"github.com/CaliLuke/loom/codegen"
)

type (
	// UserTypePackages names the struct:pkg:path packages that one example
	// file imports. A package whose name clashes with a name already held by
	// the file's scope, such as "types/log" next to the logging import,
	// gets a unique alias.
	UserTypePackages struct {
		genpkg  string
		scope   *codegen.NameScope
		names   map[string]string
		imports []*codegen.ImportSpec
	}
)

// NewUserTypePackages returns an empty set of user type packages for a file
// of the generated module rooted at genpkg. scope must already hold the names
// of the file's other imports.
func NewUserTypePackages(genpkg string, scope *codegen.NameScope) *UserTypePackages {
	return &UserTypePackages{genpkg: genpkg, scope: scope, names: make(map[string]string)}
}

// Add adds the struct:pkg:path packages of the service d in import path
// order, so that alias assignment is deterministic.
func (p *UserTypePackages) Add(d *Data) {
	specs := slices.Clone(d.UserTypeImports)
	slices.SortFunc(specs, func(a, b *codegen.ImportSpec) int {
		return strings.Compare(a.Path, b.Path)
	})
	for _, spec := range specs {
		if _, ok := p.names[spec.Path]; ok {
			continue
		}
		name := p.scope.Unique(spec.Name)
		p.names[spec.Path] = name
		p.imports = append(p.imports, &codegen.ImportSpec{Name: name, Path: spec.Path})
	}
}

// Imports returns the import specifications of the added packages.
func (p *UserTypePackages) Imports() []*codegen.ImportSpec {
	return p.imports
}

// PackageName returns the name that qualifies the types generated at loc:
// the alias of an added package, or the package's own name. It is the
// pkgName argument of codegen.NameScope.GoFullTypeRefWithPackages.
func (p *UserTypePackages) PackageName(loc *codegen.Location) string {
	if name, ok := p.names[userTypeImportPath(p.genpkg, loc)]; ok {
		return name
	}
	return loc.PackageName()
}

// reserveImportNames reserves in scope the names under which specs import
// their packages.
func reserveImportNames(scope *codegen.NameScope, specs []*codegen.ImportSpec) {
	for _, spec := range specs {
		name := spec.Name
		if name == "" {
			name = path.Base(spec.Path)
		}
		scope.Unique(name)
	}
}
