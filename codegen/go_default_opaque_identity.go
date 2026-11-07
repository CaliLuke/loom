package codegen

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
)

type opaqueDefaultPackage struct {
	path, name string
	imported   *ImportSpec
}

// opaqueDefaultIdentity proves explicit structural syntax against the concrete
// source type. Named components require the declared import path; their opaque
// definitions are not inspected or inferred. The returned qualifiers also own
// literal rewriting, including empty or nil containers with imported elements.
func opaqueDefaultIdentity(actual reflect.Type, target string, imported *ImportSpec) (packagePath, packageName, qualifier string, valid bool) {
	identity := opaqueDefaultPackage{imported: imported}
	if !identity.visit(actual) {
		return "", "", "", false
	}
	qualifier, valid = opaqueDefaultQualifier(target, identity.name, imported)
	if !valid {
		return "", "", "", false
	}
	source, err := parser.ParseExpr(actual.String())
	if err != nil || !qualifyOpaqueDefaultSyntax(source, identity.name, qualifier) {
		return "", "", "", false
	}
	var rendered bytes.Buffer
	if err := format.Node(&rendered, token.NewFileSet(), source); err != nil {
		return "", "", "", false
	}
	return identity.path, identity.name, qualifier, canonicalDefaultType(target) == canonicalDefaultType(rendered.String())
}

func opaqueDefaultQualifier(target, packageName string, imported *ImportSpec) (string, bool) {
	syntax, err := parser.ParseExpr(target)
	if err != nil {
		return "", false
	}
	qualifier, valid := "", true
	ast.Inspect(syntax, func(node ast.Node) bool {
		if selector, ok := node.(*ast.SelectorExpr); ok {
			name, ok := selector.X.(*ast.Ident)
			switch {
			case !ok || packageName == "" || !token.IsExported(selector.Sel.Name):
				valid = false
			case qualifier != "" && qualifier != name.Name:
				valid = false
			default:
				qualifier = name.Name
			}
		}
		return true
	})
	if !valid || packageName != "" && qualifier == "" {
		return "", false
	}
	if qualifier != "" {
		declaredQualifier := packageName
		if imported != nil && imported.Name != "" {
			declaredQualifier = imported.Name
		}
		if qualifier != declaredQualifier {
			return "", false
		}
	}
	return qualifier, true
}

func qualifyOpaqueDefaultSyntax(expression ast.Node, packageName, qualifier string) bool {
	valid := true
	ast.Inspect(expression, func(node ast.Node) bool {
		if selector, ok := node.(*ast.SelectorExpr); ok {
			name, ok := selector.X.(*ast.Ident)
			if !ok || name.Name != packageName || !token.IsExported(selector.Sel.Name) {
				valid = false
			} else {
				name.Name = qualifier
			}
		}
		return true
	})
	return valid
}

func (p *opaqueDefaultPackage) visit(typ reflect.Type) bool {
	if typ.PkgPath() != "" {
		if p.imported == nil || p.imported.Path != typ.PkgPath() || !token.IsExported(typ.Name()) {
			return false
		}
		name := strings.SplitN(typ.String(), ".", 2)[0]
		if p.name != "" && p.name != name {
			return false
		}
		p.path, p.name = typ.PkgPath(), name
		return true
	}
	switch typ.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array:
		return p.visit(typ.Elem())
	case reflect.Map:
		return p.visit(typ.Key()) && p.visit(typ.Elem())
	case reflect.Struct:
		for i := 0; i < typ.NumField(); i++ {
			if !p.visit(typ.Field(i).Type) {
				return false
			}
		}
	}
	return true
}
