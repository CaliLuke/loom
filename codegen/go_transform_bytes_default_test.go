package codegen

import (
	"encoding/json/jsontext"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/testingx"
)

type bytesDefaultType struct {
	name         string
	datatype     expr.DataType
	meta         expr.MetaExpr
	defaultValue any
	bytes        bool
	customAny    bool
	defaultText  string
}

// TestBytesDefaultTransformLayouts compiles the real transformer against fields
// independently emitted by NameScope. The context matrix preserves the existing
// target default policy while distinguishing missing bytes from supplied empty bytes.
func TestBytesDefaultTransformLayouts(t *testing.T) {
	var source, entries strings.Builder
	source.WriteString(bytesDefaultPrelude)
	for _, typ := range bytesDefaultTypes() {
		for mask := range 64 {
			sourceDefaults, targetDefaults := mask&1 != 0, mask&2 != 0
			sourcePointer, targetPointer := mask&4 != 0, mask&8 != 0
			required, defaulted := mask&16 != 0, mask&32 != 0
			name := fmt.Sprintf("%s_%d", typ.name, mask)
			field := &expr.AttributeExpr{Type: typ.datatype, Meta: typ.meta}
			if defaulted {
				field.DefaultValue = typ.defaultValue
			}
			parent := &expr.AttributeExpr{Type: &expr.Object{{Name: "data", Attribute: field}}}
			if required {
				parent.Validation = &expr.ValidationExpr{Required: []string{"data"}}
			}
			scope := NewNameScope()
			src := NewAttributeContext(sourcePointer, false, sourceDefaults, "", scope)
			dst := NewAttributeContext(targetPointer, false, targetDefaults, "", scope)
			code, helpers, err := GoTransform(parent, parent, "source", "target", src, dst, "", true)
			require.NoError(t, err, name)
			require.Empty(t, helpers, name)
			sourceType := scope.GoTypeDef(parent, sourcePointer, sourceDefaults)
			targetType := scope.GoTypeDef(parent, targetPointer, targetDefaults)
			fmt.Fprintf(&source, "type source_%s %s\ntype target_%s %s\n", name, sourceType, name, targetType)
			fmt.Fprintf(&source, "func convert_%s(source *source_%s) any {\n%s\nreturn target\n}\n", name, name, code)
			applyDefault := defaulted && targetDefaults && !targetPointer && !required && !typ.customAny
			fmt.Fprintf(&entries, "{name:%q, bytes:%t, sourceDefaults:%t, applyDefault:%t, required:%t, defaultText:%q, value:new(source_%s), convert:func(v any) any {return convert_%s(v.(*source_%s))}},\n", name, typ.bytes, sourceDefaults, applyDefault, required, typ.defaultText, name, name, name)
		}
	}
	source.WriteString("func TestDefaults(t *testing.T) {\ncases:=[]defaultCase{\n")
	source.WriteString(entries.String())
	source.WriteString("}\ncheckDefaults(t,cases)\n}\n")
	runBytesDefaultModule(t, source.String())
}

// TestDefaultValueStorageMetadata checks the physical override boundary even
// when its semantic source is Bytes. Unknown named custom types stay comparable;
// this does not infer their underlying Go definition or widen literal support.
func TestDefaultValueStorageMetadata(t *testing.T) {
	for _, tc := range []struct {
		name     string
		typeName string
		want     bool
	}{
		{"native_bytes", "", true},
		{"custom_comparable", "CustomBytes", false},
		{"custom_string", "string", false},
		{"slice", "[]byte", true},
		{"map", "map[string]int", true},
		{"raw_json", "jsontext.Value", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			att := &expr.AttributeExpr{Type: expr.Bytes}
			if tc.typeName != "" {
				att.Meta = expr.MetaExpr{"struct:field:type": {tc.typeName}}
			}
			require.Equal(t, tc.want, defaultValueUsesNil(att))
		})
	}
}

func bytesDefaultTypes() []bytesDefaultType {
	blob := &expr.UserTypeExpr{TypeName: "Blob", AttributeExpr: &expr.AttributeExpr{Type: expr.Bytes}}
	chain := &expr.UserTypeExpr{TypeName: "BlobAlias", AttributeExpr: &expr.AttributeExpr{Type: blob}}
	return []bytesDefaultType{
		{name: "native", datatype: expr.Bytes, defaultValue: []byte("hi"), bytes: true, defaultText: "hi"},
		{name: "named", datatype: blob, defaultValue: []byte("hi"), bytes: true, defaultText: "hi"},
		{name: "chain", datatype: chain, defaultValue: []byte("hi"), bytes: true, defaultText: "hi"},
		{name: "raw_json", datatype: expr.String, defaultValue: jsontext.Value(`"hi"`), bytes: true, defaultText: `"hi"`,
			meta: expr.MetaExpr{"struct:field:type": {"jsontext.Value", "encoding/json/jsontext"}}},
		{name: "string", datatype: expr.String, defaultValue: "hi", defaultText: "hi"},
		{name: "custom_string", datatype: expr.String, defaultValue: "hi", defaultText: "hi",
			meta: expr.MetaExpr{"struct:field:type": {"CustomString"}}},
		{name: "custom_any", datatype: expr.Any, defaultValue: "hi", customAny: true,
			meta: expr.MetaExpr{"struct:field:type": {"string"}}},
	}
}

func runBytesDefaultModule(t *testing.T, source string) {
	t.Helper()
	dir := t.TempDir()
	module := fmt.Sprintf("module example.com/bytesdefaults\n\ngo 1.27\nrequire github.com/CaliLuke/loom v0.0.0\nreplace github.com/CaliLuke/loom => %s\n", testingx.RepoRoot())
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(module), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "defaults_test.go"), []byte(source), 0o600))
	output, err := testingx.RunCmd(dir, "go", "mod", "tidy")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(dir, "go", "test", "-count=1", "./...")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(dir, "go", "vet", "./...")
	require.NoError(t, err, output)
}

const bytesDefaultPrelude = `package bytesdefaults

import (
 "encoding/json/jsontext"
 "reflect"
 "testing"
)

type Blob []byte
type BlobAlias Blob
type CustomString string
type defaultCase struct {
 name string
 bytes bool
 sourceDefaults bool
 applyDefault bool
 required bool
 defaultText string
 value any
 convert func(any) any
}

func checkDefaults(t *testing.T, cases []defaultCase) {
 for _, tc := range cases {
  t.Run(tc.name,func(t *testing.T) {
   for _, state:=range []string{"absent", "empty", "nonempty"} {
    if tc.required && state=="absent" { continue }
    field:=reflect.ValueOf(tc.value).Elem().FieldByName("Data")
    field.SetZero()
    pointer:=field.Kind()==reflect.Pointer
    if state!="absent" {
     if pointer {
      field.Set(reflect.New(field.Type().Elem()))
      field=field.Elem()
     }
     value:=""
     if state=="nonempty" { value="ok" }
     input:=reflect.ValueOf(value)
     if tc.bytes { input=reflect.ValueOf([]byte(value)) }
     field.Set(input.Convert(field.Type()))
    }
    result:=reflect.ValueOf(tc.convert(tc.value)).Elem().FieldByName("Data")
    nilValue:=false
    if result.Kind()==reflect.Pointer {
     nilValue=result.IsNil()
     if !nilValue { result=result.Elem() }
    }
    got:=""
    if !nilValue {
     if tc.bytes {
      nilValue=result.IsNil()
      got=string(result.Bytes())
     } else { got=result.String() }
    }
    want:=""
    if state=="nonempty" { want="ok" }
    zero:=state=="absent" || (!tc.bytes && !pointer && state=="empty")
    defaulted:=zero && tc.applyDefault && (tc.bytes || pointer || tc.sourceDefaults)
    if defaulted { want=tc.defaultText }
    if got!=want { t.Errorf("state=%s got=%q want=%q",state,got,want) }
    if tc.bytes && state=="empty" && nilValue { t.Error("supplied empty bytes became nil") }
    if defaulted && nilValue { t.Error("default value remained nil") }
   }
  })
 }
}
`
