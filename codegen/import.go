package codegen

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"goa.design/goa/v3/expr"
	goa "goa.design/goa/v3/pkg"
)

// DesignVersion contains the major component of the version of Goa used to
// author the design - either 2 or 3. This value is initialized when the
// generated tool is invoked by retrieving the information passed on the
// command line by the goa tool.
var DesignVersion = goa.Major

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

// GoaImport creates an import for a Goa package.
func GoaImport(rel string) *ImportSpec {
	name := ""
	if rel == "" {
		name = "goa"
		rel = "pkg"
	}
	return GoaNamedImport(rel, name)
}

// GoaNamedImport creates an import for a Goa package with the given name.
func GoaNamedImport(rel, name string) *ImportSpec {
	root := "goa.design/goa"
	if DesignVersion > 2 {
		root += "/v" + strconv.Itoa(DesignVersion)
	}
	if rel != "" {
		rel = "/" + rel
	}
	return &ImportSpec{Name: name, Path: root + rel}
}

// Code returns the Go import statement for the ImportSpec.
func (s *ImportSpec) Code() string {
	if len(s.Name) > 0 {
		return fmt.Sprintf(`%s "%s"`, s.Name, s.Path)
	}
	return fmt.Sprintf(`"%s"`, s.Path)
}

// UserTypeLocation returns the location of the user type if set via the
// struct:pkg:path metadata, nil otherwise..
func UserTypeLocation(dt expr.DataType) *Location {
	ut, ok := dt.(expr.UserType)
	if !ok {
		return nil
	}
	p, ok := ut.Attribute().Meta.Last("struct:pkg:path")
	if !ok || p == "" {
		return nil
	}
	return &Location{
		FilePath:      filepath.Join(filepath.FromSlash(p), SnakeCase(ut.Name())+".go"),
		RelImportPath: p,
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
		// Copy loop variable into body so next iteration doesn't overwrite its address https://stackoverflow.com/questions/27610039/golang-appending-leaves-only-last-element
		cp := imp
		imports = append(imports, &cp)
	}
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
