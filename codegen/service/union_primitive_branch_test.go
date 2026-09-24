package service

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
)

// TestConstructorUnionPrimitiveBranchTypes checks that constructor OneOf
// unions with bare primitive branches use the native Go type of each branch
// and declare only the loom.JSONValue alias of an Any branch, so the service
// file never redeclares a predeclared Go type.
func TestConstructorUnionPrimitiveBranchTypes(t *testing.T) {
	type branch struct {
		fieldType string
		alias     bool
	}
	cases := []struct {
		name     string
		variants func() []any
		union    string
		want     map[string]branch
	}{
		{
			name:     "string and int",
			variants: func() []any { return []any{dsl.String, dsl.Int} },
			union:    "IntOrString",
			want:     map[string]branch{"String": {fieldType: "string"}, "Int": {fieldType: "int"}},
		},
		{
			name:     "int64 and float64",
			variants: func() []any { return []any{dsl.Int64, dsl.Float64} },
			union:    "Float64OrInt64",
			want:     map[string]branch{"Int64": {fieldType: "int64"}, "Float64": {fieldType: "float64"}},
		},
		{
			name:     "boolean and bytes",
			variants: func() []any { return []any{dsl.Boolean, dsl.Bytes} },
			union:    "BooleanOrBytes",
			want:     map[string]branch{"Boolean": {fieldType: "bool"}, "Bytes": {fieldType: "[]byte"}},
		},
		{
			name:     "any and string",
			variants: func() []any { return []any{dsl.Any, dsl.String} },
			union:    "AnyOrString",
			want:     map[string]branch{"Any": {fieldType: "AnyOrStringAny", alias: true}, "String": {fieldType: "string"}},
		},
		{
			name: "primitive and user type",
			variants: func() []any {
				leaf := dsl.Type("Leaf", func() {
					dsl.Attribute("name", dsl.String)
				})
				return []any{dsl.String, leaf}
			},
			union: "LeafOrString",
			want:  map[string]branch{"String": {fieldType: "string"}, "Leaf": {fieldType: "*Leaf"}},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				variants := c.variants()
				dsl.Service("Svc", func() {
					dsl.Method("Payload", func() {
						dsl.Payload(dsl.OneOf(variants[0], variants[1:]...))
					})
					dsl.Method("Field", func() {
						dsl.Payload(func() {
							dsl.Attribute("pick", dsl.OneOf(variants[0], variants[1:]...))
						})
					})
				})
			})
			services := NewServicesData(root)
			var union *UnionTypeData
			for _, u := range services.Get("Svc").unions {
				if u.Name == c.union {
					union = u
				}
			}
			require.NotNil(t, union, "union %s not generated", c.union)
			require.Len(t, union.Fields, len(c.want))
			for _, field := range union.Fields {
				want, ok := c.want[field.Name]
				require.True(t, ok, field.Name)
				require.Equal(t, want.fieldType, field.FieldType, field.Name)
				require.Equal(t, want.alias, field.EmitPrimitiveAlias, field.Name)
			}

			files := Files("github.com/CaliLuke/loom/example", root.Services[0], services, make(map[string][]string))
			require.NotEmpty(t, files)
			var buf bytes.Buffer
			for _, s := range files[0].AllSections() {
				require.NoError(t, s.Write(&buf))
			}
			file, err := parser.ParseFile(token.NewFileSet(), files[0].Path, buf.Bytes(), parser.SkipObjectResolution)
			require.NoError(t, err, buf.String())
			for _, decl := range file.Decls {
				gen, ok := decl.(*ast.GenDecl)
				if !ok || gen.Tok != token.TYPE {
					continue
				}
				for _, spec := range gen.Specs {
					name := spec.(*ast.TypeSpec).Name.Name
					require.Falsef(t, isPredeclaredGoType(name), "service file redeclares predeclared type %q", name)
				}
			}
			for _, field := range union.Fields {
				if field.EmitPrimitiveAlias {
					require.Contains(t, buf.String(), "type "+field.FieldType+" = loom.JSONValue")
				}
			}
		})
	}
}

func isPredeclaredGoType(name string) bool {
	switch name {
	case "any", "bool", "byte", "complex64", "complex128", "error", "float32", "float64",
		"int", "int8", "int16", "int32", "int64", "rune", "string",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr":
		return true
	}
	return false
}
