package codegen

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

// TestRootCollectionPresenceTransform preserves native storage across conversion.
func TestRootCollectionPresenceTransform(t *testing.T) {
	array := &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}
	mapping := &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.String}}
	var source strings.Builder
	source.WriteString("package bytesdefaults\nimport (\"reflect\";\"testing\")\ntype Tags []string\ntype Labels map[string]string\n")
	for _, tc := range []struct {
		name string
		typ  expr.DataType
		ref  string
		seed string
	}{
		{"array", array, "[]string", `[]string{"kept"}`},
		{"map", mapping, "map[string]string", `map[string]string{"key":"kept"}`},
		{"named_array", &expr.UserTypeExpr{TypeName: "Tags", AttributeExpr: &expr.AttributeExpr{Type: array}}, "Tags", `Tags{"kept"}`},
		{"named_map", &expr.UserTypeExpr{TypeName: "Labels", AttributeExpr: &expr.AttributeExpr{Type: mapping}}, "Labels", `Labels{"key":"kept"}`},
	} {
		for _, newVar := range []bool{false, true} {
			name := fmt.Sprintf("%s_%t", tc.name, newVar)
			scope := NewNameScope()
			ctx := NewAttributeContext(false, false, false, "", scope)
			att := &expr.AttributeExpr{Type: tc.typ}
			code, helpers, err := GoTransform(att, att, "source", "target", ctx, ctx, "", newVar)
			require.NoError(t, err)
			require.Empty(t, helpers)
			fmt.Fprintf(&source, "func convert_%s(source %s) %s {\n", name, tc.ref, tc.ref)
			if !newVar {
				fmt.Fprintf(&source, "target := %s\n", tc.seed)
			}
			fmt.Fprintf(&source, "%s\nreturn target\n}\n", code)
			fmt.Fprintf(&source, `func Test_%s(t *testing.T) {
 for _, input := range []%s{nil, {}, %s} {
  got := convert_%s(input)
  if !reflect.DeepEqual(input, got) {
   t.Errorf("source=%%#v target=%%#v", input, got)
  }
  if len(input) > 0 {
   result := reflect.ValueOf(got)
   switch result.Kind() {
   case reflect.Slice:
    result.Index(0).SetString("changed")
   case reflect.Map:
    result.SetMapIndex(reflect.ValueOf("key"), reflect.ValueOf("changed"))
   }
   if reflect.DeepEqual(input, got) {
    t.Error("conversion shares populated storage")
   }
  }
 }
}
`, name, tc.ref, tc.seed, name)
		}
	}
	runBytesDefaultModule(t, source.String())
}

// TestNestedCollectionPresenceTransform preserves nil nested collection elements.
func TestNestedCollectionPresenceTransform(t *testing.T) {
	for _, tc := range []struct {
		name string
		typ  expr.DataType
		ref  string
		data string
	}{
		{"array", &expr.Array{ElemType: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}}}, "[][]string", `[][]string{nil, {}, {"kept"}}`},
		{"map", &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.String}}}}, "map[string]map[string]string", `map[string]map[string]string{"nil":nil, "empty":{}, "value":{"key":"kept"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := NewAttributeContext(false, false, false, "", NewNameScope())
			att := &expr.AttributeExpr{Type: tc.typ}
			code, helpers, err := GoTransform(att, att, "source", "target", ctx, ctx, "", true)
			require.NoError(t, err)
			require.Empty(t, helpers)
			source := fmt.Sprintf(`package bytesdefaults
import ("reflect";"testing")
func convert(source %s) %s {
%s
return target
}
func TestNested(t *testing.T) {
 input := %s
 got := convert(input)
 if !reflect.DeepEqual(input, got) {
  t.Errorf("source=%%#v target=%%#v",input,got)
 }
}
`, tc.ref, tc.ref, code, tc.data)
			runBytesDefaultModule(t, source)
		})
	}
}
