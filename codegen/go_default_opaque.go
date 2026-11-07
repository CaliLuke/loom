package codegen

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"math"
	"reflect"
	"strings"

	"github.com/CaliLuke/loom/expr"
)

type opaqueLiteralVisit struct {
	typ     reflect.Type
	pointer uintptr
}

// opaqueDefaultLiteral requires proof of physical identity before rendering a
// composite. Go type metadata alone cannot establish an unnamed source's
// assignability to an imported named type. No codec supplies that missing proof.
func opaqueDefaultLiteral(att *expr.AttributeExpr, source any, path string) (string, error) {
	value := reflect.ValueOf(source)
	if !value.IsValid() {
		return "", fmt.Errorf("%s: unsupported opaque default materialization: missing concrete Go type", path)
	}
	switch value.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.Struct, reflect.Pointer, reflect.Interface:
	default:
		return formatAttributeGoLiteral(att, source), nil
	}
	target, imported := GetMetaType(att)
	// jsontext.Value has an existing explicit byte-slice materializer.
	if target == "jsontext.Value" && value.Kind() == reflect.Slice && value.Type().Elem().Kind() == reflect.Uint8 {
		return formatAttributeGoLiteral(att, source), nil
	}
	fail := func(reason string) (string, error) {
		return "", fmt.Errorf("%s: unsupported opaque default materialization into %s: %s; supply a constructible value of the declared Go type", path, target, reason)
	}
	packagePath, packageName, qualifier, proven := opaqueDefaultIdentity(value.Type(), target, imported)
	if !proven {
		return fail("source type " + value.Type().String() + " does not prove the target representation")
	}
	if err := inspectOpaqueLiteral(value, packagePath, make(map[opaqueLiteralVisit]bool)); err != nil {
		return fail(err.Error())
	}
	expression, err := parser.ParseExpr(formatGoLiteral(source))
	if err != nil {
		return fail("source has no Go composite literal")
	}
	valid := qualifyOpaqueDefaultSyntax(expression, packageName, qualifier)
	ast.Inspect(expression, func(node ast.Node) bool {
		// fmt renders nested nonnil pointers as address conversions, not
		// reconstructible state. Only a typed nil conversion is a literal.
		if call, ok := node.(*ast.CallExpr); ok {
			if len(call.Args) != 1 {
				valid = false
			} else if argument, ok := call.Args[0].(*ast.Ident); !ok || argument.Name != "nil" {
				valid = false
			}
		}
		return true
	})
	if !valid {
		return fail("literal contains inaccessible types or unconstructible state")
	}
	var rendered bytes.Buffer
	if err := format.Node(&rendered, token.NewFileSet(), expression); err != nil {
		return fail("source has no renderable Go literal")
	}
	return rendered.String(), nil
}

// canonicalDefaultType compares only explicit type syntax and builtin aliases;
// it does not discover the definition of an opaque named type.
func canonicalDefaultType(name string) string {
	expression, err := parser.ParseExpr(name)
	if err != nil {
		return "invalid:" + name
	}
	ast.Inspect(expression, func(node ast.Node) bool {
		if ident, ok := node.(*ast.Ident); ok {
			switch ident.Name {
			case "byte":
				ident.Name = "uint8"
			case "rune":
				ident.Name = "int32"
			}
		}
		return true
	})
	var rendered bytes.Buffer
	if err := format.Node(&rendered, token.NewFileSet(), expression); err != nil {
		return "invalid:" + name
	}
	return strings.ReplaceAll(rendered.String(), "interface{}", "any")
}

func inspectOpaqueLiteral(value reflect.Value, packagePath string, active map[opaqueLiteralVisit]bool) error {
	if !value.IsValid() {
		return nil
	}
	typ := value.Type()
	if typ.PkgPath() != "" && (typ.PkgPath() != packagePath || !token.IsExported(typ.Name())) {
		return fmt.Errorf("type %s is not accessible through the declared import", typ)
	}
	if value.CanInterface() {
		if _, custom := value.Interface().(fmt.GoStringer); custom {
			return fmt.Errorf("type %s supplies custom Go formatting instead of a known literal", typ)
		}
		if _, custom := value.Interface().(fmt.Formatter); custom {
			return fmt.Errorf("type %s supplies custom formatting instead of a known literal", typ)
		}
	}
	switch value.Kind() {
	case reflect.Interface:
		return inspectOpaqueInterface(value, packagePath, active)
	case reflect.Map, reflect.Slice, reflect.Pointer:
		return inspectOpaqueReference(value, packagePath, active)
	case reflect.Array:
		return inspectOpaqueElements(value, packagePath, active)
	case reflect.Struct:
		return inspectOpaqueFields(value, packagePath, active)
	case reflect.Float32, reflect.Float64:
		if math.IsInf(value.Float(), 0) || math.IsNaN(value.Float()) {
			return fmt.Errorf("nonfinite numbers have no Go literal")
		}
	case reflect.Bool, reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
	default:
		return fmt.Errorf("%s has no supported Go literal", typ)
	}
	return nil
}

func inspectOpaqueInterface(value reflect.Value, packagePath string, active map[opaqueLiteralVisit]bool) error {
	if value.IsNil() {
		return nil
	}
	actual := value.Elem()
	if actual.Type().PkgPath() != "" {
		switch actual.Kind() {
		case reflect.Map, reflect.Slice, reflect.Array, reflect.Struct, reflect.Pointer:
		default:
			return fmt.Errorf("dynamic type %s has no identity-preserving literal", actual.Type())
		}
	}
	return inspectOpaqueLiteral(actual, packagePath, active)
}

func inspectOpaqueReference(value reflect.Value, packagePath string, active map[opaqueLiteralVisit]bool) error {
	if value.IsNil() {
		return nil
	}
	visit := opaqueLiteralVisit{typ: value.Type(), pointer: uintptr(value.UnsafePointer())}
	if active[visit] {
		return fmt.Errorf("cyclic Go values have no finite literal")
	}
	active[visit] = true
	defer delete(active, visit)
	switch value.Kind() {
	case reflect.Pointer:
		if value.Elem().Kind() != reflect.Struct {
			return fmt.Errorf("pointer state has no composite literal")
		}
		return inspectOpaqueLiteral(value.Elem(), packagePath, active)
	case reflect.Map:
		entries := value.MapRange()
		for entries.Next() {
			if err := inspectOpaqueLiteral(entries.Key(), packagePath, active); err != nil {
				return err
			}
			if err := inspectOpaqueLiteral(entries.Value(), packagePath, active); err != nil {
				return err
			}
		}
		return nil
	default:
		return inspectOpaqueElements(value, packagePath, active)
	}
}

func inspectOpaqueElements(value reflect.Value, packagePath string, active map[opaqueLiteralVisit]bool) error {
	for i := 0; i < value.Len(); i++ {
		if err := inspectOpaqueLiteral(value.Index(i), packagePath, active); err != nil {
			return err
		}
	}
	return nil
}

func inspectOpaqueFields(value reflect.Value, packagePath string, active map[opaqueLiteralVisit]bool) error {
	typ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		if !typ.Field(i).IsExported() {
			return fmt.Errorf("field %s.%s is not accessible", typ, typ.Field(i).Name)
		}
		if err := inspectOpaqueLiteral(value.Field(i), packagePath, active); err != nil {
			return err
		}
	}
	return nil
}
