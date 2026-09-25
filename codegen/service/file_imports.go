package service

import (
	"maps"
	"slices"
	"strings"

	"github.com/CaliLuke/loom/codegen"
)

// FileImports returns the names that qualify the types of the struct:pkg:path
// packages of the service in a generated file that lists imports, and the
// imports of the file with these names. A package keeps its name unless the
// name clashes with the name of another import of the file, see
// codegen.AliasClashingImports, or imports already list it under another
// name. names maps the relative import paths of the packages imported under
// another name to that name, and is empty when there is none. The file
// renders its code with the data that ServicesData.WithPackageNames returns
// for names. FileImports is idempotent: the imports it returns yield the same
// names. It considers the packages of UserTypeImports only, so it renames
// nothing before SetUserTypeImports.
func (d *Data) FileImports(imports []*codegen.ImportSpec) (names map[string]string, specs []*codegen.ImportSpec) {
	pkgNames := make(map[string]string, len(d.UserTypeImports))
	paths := make([]string, len(d.UserTypeImports))
	for i, spec := range d.UserTypeImports {
		pkgNames[spec.Path] = spec.Name
		paths[i] = spec.Path
	}
	aliases := codegen.AliasClashingImports(imports, paths)
	specs = make([]*codegen.ImportSpec, len(imports))
	for i, spec := range imports {
		specs[i] = spec
		if spec == nil {
			continue
		}
		pkgName, ok := pkgNames[spec.Path]
		if !ok {
			continue
		}
		name := spec.Name
		if alias, ok := aliases[spec.Path]; ok {
			name = alias
			specs[i] = &codegen.ImportSpec{Name: alias, Path: spec.Path}
		}
		if name != pkgName {
			if names == nil {
				names = make(map[string]string)
			}
			names[strings.TrimPrefix(spec.Path, d.genpkg+"/")] = name
		}
	}
	if names == nil {
		return nil, imports
	}
	return names, specs
}

// WithPackageNames returns the services data whose code qualifies the types
// of the struct:pkg:path package at each relative import path of names with
// the mapped name, see codegen.NewNameScopeWithPackageNames. It returns d
// when names is empty. The returned data is analyzed anew, once for equal
// names, and its services keep the user type imports of d.
func (d *ServicesData) WithPackageNames(names map[string]string) *ServicesData {
	if len(names) == 0 {
		return d
	}
	key := packageNamesKey(names)
	if renamed, ok := d.renamed[key]; ok {
		return renamed
	}
	renamed := &ServicesData{
		Root:         d.Root,
		Services:     make(map[string]*Data),
		Ctx:          d.Ctx,
		packageNames: names,
		base:         d,
	}
	if d.renamed == nil {
		d.renamed = make(map[string]*ServicesData)
	}
	d.renamed[key] = renamed
	return renamed
}

// packageNamesKey returns a key that identifies names.
func packageNamesKey(names map[string]string) string {
	var b strings.Builder
	for _, path := range slices.Sorted(maps.Keys(names)) {
		b.WriteString(path)
		b.WriteByte('=')
		b.WriteString(names[path])
		b.WriteByte('\n')
	}
	return b.String()
}

// fileData returns the data of the service name that renders the code of a
// generated file that lists imports, and the imports of the file, see
// Data.FileImports.
func (d *ServicesData) fileData(name string, imports []*codegen.ImportSpec) (*Data, []*codegen.ImportSpec) {
	names, imports := d.Get(name).FileImports(imports)
	return d.WithPackageNames(names).Get(name), imports
}
