package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/testingx"
)

// TestBytesValidationPhysicalLayout compiles validators against independently
// generated field declarations. Native nil-capable values, forced alias pointers
// and ordinary optional aliases must retain their different physical layouts.
func TestBytesValidationPhysicalLayout(t *testing.T) {
	alias := func(name string, underlying expr.DataType) *expr.UserTypeExpr {
		return &expr.UserTypeExpr{TypeName: name, AttributeExpr: &expr.AttributeExpr{Type: underlying}}
	}
	blob := alias("Blob", expr.Bytes)
	chain := alias("BlobAlias", blob)
	label := alias("Label", expr.String)
	types := []struct {
		name   string
		typeOf expr.DataType
		bytes  bool
	}{
		{"bytes", expr.Bytes, true},
		{"named_bytes", blob, true},
		{"alias_chain", chain, true},
		{"string", expr.String, false},
		{"named_string", label, false},
	}
	var source, entries strings.Builder
	source.WriteString(bytesValidationLayoutPrelude)
	minimum, maximum := 2, 3
	for _, typ := range types {
		for _, pointer := range []bool{false, true} {
			for _, required := range []bool{false, true} {
				for _, defaulted := range []bool{false, true} {
					for _, useDefault := range []bool{false, true} {
						name := fmt.Sprintf("%s_p%t_r%t_d%t_u%t", typ.name, pointer, required, defaulted, useDefault)
						field := &expr.AttributeExpr{Type: typ.typeOf, Validation: &expr.ValidationExpr{
							MinLength: &minimum, MaxLength: &maximum,
						}}
						if defaulted {
							field.DefaultValue = "hi"
							if typ.bytes {
								field.DefaultValue = []byte("hi")
							}
						}
						parent := &expr.AttributeExpr{Type: &expr.Object{{Name: "data", Attribute: field}}}
						if required {
							parent.Validation = &expr.ValidationExpr{Required: []string{"data"}}
						}
						scope := NewNameScope()
						ctx := NewAttributeContext(pointer, false, useDefault, "", scope)
						definition := scope.GoTypeDef(parent, pointer, useDefault)
						validation := ValidationCode(parent, nil, ctx, true, false, false, "body")
						fmt.Fprintf(&source, "type case_%s %s\nfunc validate_%s(body *case_%s) (err error) {\n%s\nreturn\n}\n", name, definition, name, name, validation)
						missingInvalid := required || !pointer && defaulted && useDefault
						fmt.Fprintf(&entries, "{name:%q, bytes:%t, missingInvalid:%t, value:new(case_%s), validate:func(v any) error {return validate_%s(v.(*case_%s))}},\n", name, typ.bytes, missingInvalid, name, name, name)
					}
				}
			}
		}
	}
	source.WriteString("func TestLayouts(t *testing.T) {\ncases:=[]layoutCase{\n")
	source.WriteString(entries.String())
	source.WriteString("}\ncheckLayouts(t,cases)\n}\n")
	dir := t.TempDir()
	module := fmt.Sprintf("module example.com/validationlayout\n\ngo 1.27\nrequire github.com/CaliLuke/loom v0.0.0\nreplace github.com/CaliLuke/loom => %s\n", testingx.RepoRoot())
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(module), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "layout_test.go"), []byte(source.String()), 0o600))
	output, err := testingx.RunCmd(dir, "go", "mod", "tidy")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(dir, "go", "test", "-count=1", "./...")
	require.NoError(t, err, output)
}

// TestBytesValidationExtractedValues checks that presence wrappers hand their
// concrete slice value to validation even when the enclosing context is forced.
func TestBytesValidationExtractedValues(t *testing.T) {
	minimum := 2
	for _, nullable := range []bool{false, true} {
		t.Run(fmt.Sprintf("nullable=%t", nullable), func(t *testing.T) {
			blob := &expr.UserTypeExpr{TypeName: "Blob", AttributeExpr: &expr.AttributeExpr{
				Type: expr.Bytes, Validation: &expr.ValidationExpr{MinLength: &minimum},
			}}
			field := &expr.AttributeExpr{Type: blob, Nullable: nullable}
			parent := &expr.AttributeExpr{Type: &expr.Object{{Name: "data", Attribute: field}}}
			ctx := NewAttributeContext(true, false, false, "", NewNameScope())
			ctx.JSONPresence = true
			code := ValidationCode(parent, nil, ctx, true, false, false, "body")
			require.Contains(t, code, "body.Data.Value()")
			if nullable {
				require.Contains(t, code, "len([]byte(actual))")
			} else {
				require.Contains(t, code, "len(actual)")
			}
			require.NotContains(t, code, "[]byte(*actual)")
		})
	}
	for _, named := range []bool{false, true} {
		t.Run(fmt.Sprintf("any_named=%t", named), func(t *testing.T) {
			var typ expr.DataType = expr.Any
			if named {
				typ = &expr.UserTypeExpr{TypeName: "Arbitrary", AttributeExpr: &expr.AttributeExpr{Type: expr.Any}}
			}
			ctx := NewAttributeContext(true, false, false, "", NewNameScope())
			data := newValidationRenderData(&expr.AttributeExpr{Type: typ}, ctx, false, named, "value", "body")
			require.NotContains(t, data.TargetValue, "*")
		})
	}
}

const bytesValidationLayoutPrelude = `package validationlayout

import (
 "reflect"
 "testing"
 "unicode/utf8"
 loom "github.com/CaliLuke/loom/pkg"
)

type Blob []byte
type BlobAlias Blob
type Label string
type layoutCase struct {
 name string
 bytes bool
 missingInvalid bool
 value any
 validate func(any) error
}

func checkLayouts(t *testing.T, cases []layoutCase) {
 for _, tc := range cases {
  t.Run(tc.name,func(t *testing.T) {
   if err:=tc.validate(tc.value); (err!=nil)!=tc.missingInvalid {
    t.Errorf("zero storage error=%v want invalid=%t",err,tc.missingInvalid)
   }
   for _, value:=range []string{"", "h", "hi", "hey", "long"} {
    field:=reflect.ValueOf(tc.value).Elem().FieldByName("Data")
    if field.Kind()==reflect.Pointer {
     field.Set(reflect.New(field.Type().Elem()))
     field=field.Elem()
    }
    input:=reflect.ValueOf(value)
    if tc.bytes {input=reflect.ValueOf([]byte(value))}
    field.Set(input.Convert(field.Type()))
    want:=len(value)>=2 && len(value)<=3
    if err:=tc.validate(tc.value); (err==nil)!=want {
     t.Errorf("value %q error=%v want valid=%t",value,err,want)
    }
   }
  })
 }
}
`
