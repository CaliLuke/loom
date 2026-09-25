package codegen

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/naming"
	loom "github.com/CaliLuke/loom/pkg"
)

// DesignVersion contains the major component of the version of Loom used to
// author the design. This value is initialized when the generated tool is
// invoked by retrieving the information passed on the command line by the
// loom tool.
var DesignVersion = loom.Major

type (
	// ImportSpec defines a generated import statement.
	ImportSpec struct {
		// Name of imported package if needed.
		Name string
		// Go import path of package.
		Path string
	}

	// Location defines a file location and import details.
	Location struct {
		// FilePath is the path to the file.
		FilePath string
		// RelImportPath is the Go import path starting after the gen
		// folder.
		RelImportPath string
	}
)

// NewImport creates an import spec.
func NewImport(name, path string) *ImportSpec {
	return &ImportSpec{Name: name, Path: path}
}

// SimpleImport creates an import with no explicit path component.
func SimpleImport(path string) *ImportSpec {
	return &ImportSpec{Path: path}
}

// LoomImport creates an import for a Loom package.
func LoomImport(rel string) *ImportSpec {
	name := ""
	if rel == "" {
		name = "loom"
		rel = "pkg"
	}
	return LoomNamedImport(rel, name)
}

// LoomNamedImport creates an import for a Loom package with the given name.
func LoomNamedImport(rel, name string) *ImportSpec {
	root := "github.com/CaliLuke/loom"
	if rel != "" {
		rel = "/" + rel
	}
	return &ImportSpec{Name: name, Path: root + rel}
}

// AliasClashingImports returns the aliases under which a generated file that
// lists imports must import the packages at paths, keyed by path, so that no
// two of its imports share a name. Only the packages of paths whose names
// clash get an alias: every other import keeps its name, and so does the
// first package of paths, in order, among those that share a name that no
// other import uses. An alias is the package name followed by a number that
// makes it unique in the file. The paths absent from imports are ignored.
//
// codegen.File removes the unused imports of a name together, so a file that
// lists two imports of one name compiles only when it uses neither. Aliasing
// the packages of a file that compiles therefore leaves the file unchanged.
func AliasClashingImports(imports []*ImportSpec, paths []string) map[string]string {
	names := make(map[string]string, len(paths))
	for _, path := range paths {
		names[path] = ""
	}
	scope := NewNameScope()
	reserve := func(name string) bool {
		if scope.PeekUnique(name) != name {
			return false
		}
		scope.Unique(name)
		return true
	}
	for _, spec := range imports {
		if spec == nil {
			continue
		}
		if _, ok := names[spec.Path]; !ok {
			reserve(importName(spec))
		}
	}
	var clashing []string
	for _, path := range paths {
		spec := findImport(imports, path)
		if spec == nil || names[path] != "" {
			continue
		}
		names[path] = importName(spec)
		if !reserve(names[path]) {
			clashing = append(clashing, path)
		}
	}
	aliases := make(map[string]string, len(clashing))
	for _, path := range clashing {
		aliases[path] = scope.Unique(names[path])
	}
	return aliases
}

// Code returns the Go import statement for the ImportSpec.
func (s *ImportSpec) Code() string {
	if len(s.Name) > 0 {
		return fmt.Sprintf(`%s "%s"`, s.Name, s.Path)
	}
	return fmt.Sprintf(`"%s"`, s.Path)
}

// UserTypeLocation returns the location of the user type if set via the
// struct:pkg:path metadata, nil otherwise. The location escapes each non-ASCII
// rune of the metadata value as the generated service package paths do.
func UserTypeLocation(dt expr.DataType) *Location {
	ut, ok := dt.(expr.UserType)
	if !ok {
		return nil
	}
	p, ok := ut.Attribute().Meta.Last("struct:pkg:path")
	if !ok || p == "" {
		return nil
	}
	// Go import paths are ASCII only; escape the non-ASCII runes of the
	// design path the same way as service package paths.
	rel := naming.EscapeNonASCII(p)
	return &Location{
		FilePath:      filepath.Join(filepath.FromSlash(rel), SnakeCase(ut.Name())+".go"),
		RelImportPath: rel,
	}
}

// PackageName returns the package name of the given location.
func (loc *Location) PackageName() string {
	if loc == nil {
		return ""
	}
	return Goify(filepath.Base(loc.RelImportPath), false)
}

// JoinImportPath constructs a generated import path by joining the generation
// package root with a path relative to the generated `gen` tree.
func JoinImportPath(genpkg, rel string) string {
	if rel == "" {
		return ""
	}
	base := strings.TrimSuffix(genpkg, "/")
	for strings.HasSuffix(base, "/gen") {
		base = strings.TrimSuffix(base, "/gen")
	}
	return filepath.ToSlash(filepath.Join(base, "gen", rel))
}

// GetMetaType retrieves the type and package defined by the struct:field:type
// metadata if any.
func GetMetaType(att *expr.AttributeExpr) (typeName string, importS *ImportSpec) {
	if att == nil {
		return
	}
	if args, ok := att.Meta["struct:field:type"]; ok {
		if len(args) > 0 {
			typeName = args[0]
		}
		if len(args) > 1 {
			importS = &ImportSpec{Path: args[1]}
		}
		if len(args) > 2 {
			importS.Name = args[2]
		}
	}
	return
}

// GetMetaTypeImports parses the attribute for all user defined imports
func GetMetaTypeImports(att *expr.AttributeExpr) []*ImportSpec {
	return safelyGetMetaTypeImports(att, nil)
}

// GatherAttributeImports collects import specifications required by the given
// attribute, including meta-type imports and user types placed in external
// generated packages.
func GatherAttributeImports(genpkg string, att *expr.AttributeExpr) []*ImportSpec {
	uniq := make(map[string]*ImportSpec)
	var visit func(*expr.AttributeExpr)
	visit = func(a *expr.AttributeExpr) {
		if a == nil {
			return
		}
		for _, im := range GetMetaTypeImports(a) {
			if im != nil && im.Path != "" {
				uniq[im.Path] = im
			}
		}
		switch dt := a.Type.(type) {
		case expr.UserType:
			if loc := UserTypeLocation(dt); loc != nil && loc.RelImportPath != "" {
				imp := &ImportSpec{
					Name: loc.PackageName(),
					Path: JoinImportPath(genpkg, loc.RelImportPath),
				}
				uniq[imp.Path] = imp
			}
			visit(dt.Attribute())
		case *expr.Array:
			visit(dt.ElemType)
		case *expr.Map:
			visit(dt.KeyType)
			visit(dt.ElemType)
		case *expr.Object:
			for _, nat := range *dt {
				visit(nat.Attribute)
			}
		case expr.CompositeExpr:
			visit(dt.Attribute())
		}
	}
	visit(att)
	if len(uniq) == 0 {
		return nil
	}
	paths := make([]string, 0, len(uniq))
	for p := range uniq {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	imports := make([]*ImportSpec, 0, len(paths))
	for _, p := range paths {
		imports = append(imports, uniq[p])
	}
	return imports
}

// safelyGetMetaTypeImports parses attributes while keeping track of previous usertypes to avoid infinite recursion
func safelyGetMetaTypeImports(att *expr.AttributeExpr, seen map[string]struct{}) []*ImportSpec {
	if att == nil {
		return nil
	}
	if seen == nil {
		seen = make(map[string]struct{})
	}
	uniqueImports := make(map[ImportSpec]struct{})
	imports := make([]*ImportSpec, 0)

	switch t := att.Type.(type) {
	case expr.UserType:
		if _, wasSeen := seen[t.ID()]; wasSeen {
			return imports
		}
		seen[t.ID()] = struct{}{}
		for _, im := range safelyGetMetaTypeImports(t.Attribute(), seen) {
			if im != nil {
				uniqueImports[*im] = struct{}{}
			}
		}
	case *expr.Array:
		for _, im := range safelyGetMetaTypeImports(t.ElemType, seen) {
			if im != nil {
				uniqueImports[*im] = struct{}{}
			}
		}
	case *expr.Map:
		for _, im := range safelyGetMetaTypeImports(t.ElemType, seen) {
			if im != nil {
				uniqueImports[*im] = struct{}{}
			}
		}
		for _, im := range safelyGetMetaTypeImports(t.KeyType, seen) {
			if im != nil {
				uniqueImports[*im] = struct{}{}
			}
		}
	case *expr.Object:
		for _, na := range *t {
			for _, im := range safelyGetMetaTypeImports(na.Attribute, seen) {
				if im != nil {
					uniqueImports[*im] = struct{}{}
				}
			}
		}
	}
	_, im := GetMetaType(att)
	if im != nil {
		uniqueImports[*im] = struct{}{}
	}
	for imp := range uniqueImports {
		imports = append(imports, &imp)
	}
	sortImportSpecs(imports)
	return imports
}

// AddServiceMetaTypeImports adds meta type imports for each method of the service expr
func AddServiceMetaTypeImports(header *SectionTemplate, svc *expr.ServiceExpr) {
	for _, m := range svc.Methods {
		AddImport(header, GetMetaTypeImports(m.Payload)...)
		AddImport(header, GetMetaTypeImports(m.StreamingPayload)...)
		AddImport(header, GetMetaTypeImports(m.Result)...)
	}
}

// sortImportSpecs orders imports by path, then by name, so that output built
// from map iteration does not depend on the process.
func sortImportSpecs(imports []*ImportSpec) {
	sort.Slice(imports, func(i, j int) bool {
		if imports[i].Path != imports[j].Path {
			return imports[i].Path < imports[j].Path
		}
		return imports[i].Name < imports[j].Name
	})
}

// importName returns the name under which spec imports its package.
func importName(spec *ImportSpec) string {
	if spec.Name != "" {
		return spec.Name
	}
	return inferPackageName(spec.Path)
}

// findImport returns the import of imports at path, nil if there is none.
func findImport(imports []*ImportSpec, path string) *ImportSpec {
	for _, spec := range imports {
		if spec != nil && spec.Path == path {
			return spec
		}
	}
	return nil
}
