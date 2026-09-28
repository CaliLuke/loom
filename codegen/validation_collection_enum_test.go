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

// TestCollectionEnumValidation executes the emitted comparisons. The finite
// string domain checks every array of length zero through three over {a,b}.
func TestCollectionEnumValidation(t *testing.T) {
	array := func(element expr.DataType) expr.DataType {
		return &expr.Array{ElemType: &expr.AttributeExpr{Type: element}}
	}
	named := &expr.UserTypeExpr{TypeName: "Labels", AttributeExpr: &expr.AttributeExpr{Type: array(expr.String)}}
	cases := []struct {
		name      string
		datatype  expr.DataType
		reference string
		values    []any
		valid     []string
		invalid   []string
	}{
		{
			name:      "strings",
			datatype:  array(expr.String),
			reference: "[]string",
			values:    []any{[]string{}, []string{"a"}, []string{"a", "b"}},
			valid:     []string{"nil"},
			invalid:   nil,
		},
		{
			name:      "named",
			datatype:  named,
			reference: "Labels",
			values:    []any{[]string{}, []string{"a"}, []string{"a", "b"}},
			valid:     []string{"nil"},
			invalid:   nil,
		},
		{
			name:      "bytes",
			datatype:  expr.Bytes,
			reference: "[]byte",
			values:    []any{[]byte{}, []byte{0, 255}},
			valid:     []string{"nil", "{}", "{0,255}"},
			invalid:   []string{"{255,0}", "{0}", "{0,255,0}"},
		},
		{
			name:      "uint",
			datatype:  array(expr.UInt),
			reference: "[]uint",
			values:    []any{[]uint8{1, 2}},
			valid:     []string{"{1,2}"},
			invalid:   []string{"nil", "{2,1}", "{1,2,3}"},
		},
		{
			name:      "large",
			datatype:  array(expr.UInt64),
			reference: "[]uint64",
			values:    []any{[]uint64{9007199254740993}},
			valid:     []string{"{9007199254740993}"},
			invalid:   []string{"{9007199254740992}"},
		},
		{
			name:      "float",
			datatype:  array(expr.Float64),
			reference: "[]float64",
			values:    []any{[]int{1, 2}},
			valid:     []string{"{1,2}"},
			invalid:   []string{"{1,2.1}"},
		},
		{
			name:      "float32",
			datatype:  array(expr.Float32),
			reference: "[]float32",
			values:    []any{[]float64{0.1}},
			valid:     []string{"{0.1}"},
			invalid:   []string{"{0.2}"},
		},
		{
			name:      "bool",
			datatype:  array(expr.Boolean),
			reference: "[]bool",
			values:    []any{[]bool{true, false}},
			valid:     []string{"{true,false}"},
			invalid:   []string{"{false,true}"},
		},
		{
			name:      "nested",
			datatype:  array(array(expr.String)),
			reference: "[][]string",
			values:    []any{[][]string{{"a"}, {"b"}}},
			valid:     []string{"{{\"a\"},{\"b\"}}"},
			invalid:   []string{"nil", "{{\"a\"}}", "{{\"b\"},{\"a\"}}"},
		},
		{
			name:      "byte_elements",
			datatype:  array(expr.Bytes),
			reference: "[][]byte",
			values:    []any{[][]byte{{0, 255}}},
			valid:     []string{"{{0,255}}"},
			invalid:   []string{"{{255,0}}"},
		},
		{
			name:      "any_elements",
			datatype:  array(expr.Any),
			reference: "[]loom.JSONValue",
			values:    []any{[]any{map[string]any{"a": 1, "b": 2}}},
			valid:     []string{"{loom.JSONValue(`{\"b\":2.0,\"a\":1}`)}"},
			invalid:   []string{"{loom.JSONValue(`{\"a\":1}`)}", "{loom.JSONValue(`invalid`)}"},
		},
		{
			name:      "rounded_float32",
			datatype:  array(expr.Float32),
			reference: "[]float32",
			values:    []any{[]float64{1.23456789}},
			valid:     []string{"{1.23456789}"},
			invalid:   []string{"{1.2345677}"},
		},
		{
			name:      "float64_from_float32",
			datatype:  array(expr.Float64),
			reference: "[]float64",
			values:    []any{[]float32{0.1}},
			valid:     []string{"{0.1}"},
			invalid:   []string{"{0.10000000149011612}"},
		},
		{
			name:      "rounded_float64",
			datatype:  array(expr.Float64),
			reference: "[]float64",
			values:    []any{[]uint64{9007199254740993}},
			valid:     []string{"{9007199254740993}"},
			invalid:   []string{"{9007199254740994}"},
		},
		{
			name:      "nil_array_enum",
			datatype:  array(expr.String),
			reference: "[]string",
			values:    []any{[]string(nil)},
			valid:     []string{"nil", "{}"},
			invalid:   []string{"{\"a\"}"},
		},
		{
			name:      "string_byte_enum",
			datatype:  expr.Bytes,
			reference: "[]byte",
			values:    []any{"ab"},
			valid:     []string{"{'a','b'}"},
			invalid:   []string{"{'b','a'}"},
		},
		{
			name:      "nil_byte_enum",
			datatype:  expr.Bytes,
			reference: "[]byte",
			values:    []any{[]byte(nil)},
			valid:     []string{"nil", "{}"},
			invalid:   []string{"{1}"},
		},
		{
			name:      "nested_nil_enum",
			datatype:  array(array(expr.String)),
			reference: "[][]string",
			values:    []any{[][]string{nil}},
			valid:     []string{"{nil}", "{{}}"},
			invalid:   []string{"{}", "{{\"a\"}}"},
		},
		{
			name:      "nested_string_bytes",
			datatype:  array(expr.Bytes),
			reference: "[][]byte",
			values:    []any{[]string{"ab"}},
			valid:     []string{"{{'a','b'}}"},
			invalid:   []string{"{{'b','a'}}"},
		},
		{
			name:      "named_float32",
			datatype:  &expr.UserTypeExpr{TypeName: "Scores", AttributeExpr: &expr.AttributeExpr{Type: array(&expr.UserTypeExpr{TypeName: "Score", AttributeExpr: &expr.AttributeExpr{Type: expr.Float32}})}},
			reference: "Scores",
			values:    []any{[]float64{1.23456789}},
			valid:     []string{"{1.23456789}"},
			invalid:   []string{"{1.2345677}"},
		},
		{
			name:      "named_string_bytes",
			datatype:  array(&expr.UserTypeExpr{TypeName: "Blob", AttributeExpr: &expr.AttributeExpr{Type: expr.Bytes}}),
			reference: "[]Blob",
			values:    []any{[]string{"ab"}},
			valid:     []string{"{{'a','b'}}"},
			invalid:   []string{"{{'b','a'}}"},
		},
		{
			name:      "map_float32",
			datatype:  array(&expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.Float32}}),
			reference: "[]map[string]float32",
			values:    []any{[]map[string]float64{{"a": 1.23456789}}},
			valid:     []string{"{{\"a\":1.23456789}}"},
			invalid:   []string{"{{\"a\":1.2345677}}"},
		},
		{
			name:      "object_float32",
			datatype:  array(&expr.Object{{Name: "score:rating", Attribute: &expr.AttributeExpr{Type: expr.Float32}}}),
			reference: "[]*Record",
			values:    []any{[]map[any]any{{"score:rating": 1.23456789}}},
			valid:     []string{"{{Score:1.23456789}}"},
			invalid:   []string{"{{Score:1.2345677}}"},
		},
		{
			name:      "struct_float32",
			datatype:  array(&expr.Object{{Name: "score:rating", Attribute: &expr.AttributeExpr{Type: expr.Float32}}}),
			reference: "[]*Record",
			values: []any{[]struct {
				Score float64 `json:"rating"`
			}{{Score: 1.23456789}}},
			valid:   []string{"{{Score:1.23456789}}"},
			invalid: []string{"{{Score:1.2345677}}"},
		},
		{
			name:      "union_float32",
			datatype:  array(&expr.Union{TypeName: "Value", Values: []*expr.NamedAttributeExpr{{Name: "score", Attribute: &expr.AttributeExpr{Type: expr.Float32}}, {Name: "text", Attribute: &expr.AttributeExpr{Type: expr.String}}}}),
			reference: "[]loom.JSONValue",
			values:    []any{[]float64{1.23456789}},
			valid:     []string{"{loom.JSONValue(`{\"type\":\"score\",\"value\":1.2345679}`)}"},
			invalid:   []string{"{loom.JSONValue(`{\"type\":\"score\",\"value\":1.2345677}`)}"},
		},
		{
			name:      "any_typed_nil",
			datatype:  array(expr.Any),
			reference: "[]loom.JSONValue",
			values:    []any{[]any{[]byte(nil)}},
			valid:     []string{"{loom.JSONValue(\"\\\"\\\"\")}"},
			invalid:   []string{"{loom.JSONValue(`null`)}"},
		},
	}
	var source strings.Builder
	source.WriteString(`package enums
import (
 "testing"
 loom "github.com/CaliLuke/loom/pkg"
)
type Labels []string
type Score float32
type Scores []Score
type Blob []byte
`)
	source.WriteString("type Record struct {\n Score float32 `json:\"rating\"`\n}\n")
	for _, tc := range cases {
		for _, required := range []bool{true, false} {
			name := fmt.Sprintf("%s_%t", tc.name, required)
			attribute := &expr.AttributeExpr{Type: tc.datatype, Validation: &expr.ValidationExpr{Values: tc.values}}
			ctx := NewAttributeContext(false, false, false, "", NewNameScope())
			code := validationCode(attribute, ctx, required, false, "target", "value")
			require.Contains(t, code, "loom.JSONValueFrom(target)")
			require.Contains(t, code, "loom.JSONValueEqual(encoded,")
			fmt.Fprintf(&source, "func validate_%s(target %s) (err error) {\n%s\nreturn\n}\n", name, tc.reference, code)
			fmt.Fprintf(&source, "func Test_%s(t *testing.T) {\n", name)
			if tc.name == "strings" || tc.name == "named" {
				fmt.Fprintf(&source, `for length := 0; length <= 3; length++ {
 for bits := 0; bits < 1<<length; bits++ {
  value := make(%s, length)
  for i := range value {
   value[i] = "a"
   if bits & (1<<i) != 0 {
 value[i] = "b"
 }
  }
  want := length == 0 || length == 1 && value[0] == "a" || length == 2 && value[0] == "a" && value[1] == "b"
  if got := validate_%s(value); (got == nil) != want {
 t.Errorf("value=%%v err=%%v want=%%t", value, got, want)
 }
 }
}
`, tc.reference, name)
			}
			for _, group := range []struct {
				values []string
				valid  bool
			}{{tc.valid, true}, {tc.invalid, false}} {
				for _, value := range group.values {
					literal := tc.reference + value
					valid := group.valid
					if value == "nil" {
						literal = tc.reference + "(nil)"
						valid = valid || !required
					}
					fmt.Fprintf(&source, "if err := validate_%s(%s); (err == nil) != %t {\n t.Errorf(%q, err)\n }\n", name, literal, valid, "value "+value+": %v")
				}
			}
			source.WriteString("}\n")
		}
	}
	dir := t.TempDir()
	module := fmt.Sprintf("module example.com/enums\n\ngo 1.27\nrequire github.com/CaliLuke/loom v0.0.0\nreplace github.com/CaliLuke/loom => %s\n", testingx.RepoRoot())
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(module), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "enum_test.go"), []byte(source.String()), 0o600))
	output, err := testingx.RunCmd(dir, "go", "mod", "tidy")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(dir, "go", "test", "./...")
	require.NoError(t, err, output)
}
