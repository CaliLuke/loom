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

// TestNamedCollectionPresenceTransform keeps the destination collection name
// when a conversion produces an unnamed slice or map inside a presence wrapper.
func TestNamedCollectionPresenceTransform(t *testing.T) {
	cases := []struct {
		name                  string
		typ                   expr.DataType
		definition, populated string
	}{
		{"array", &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}, "[]string", `["a","b"]`},
		{"map", &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.Int}}, "map[string]int", `{"a":7}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, nullable := range []bool{false, true} {
				name := "optional"
				if nullable {
					name = "nullable"
				}
				t.Run(name, func(t *testing.T) {
					sourceType := &expr.UserTypeExpr{TypeName: "Source", AttributeExpr: &expr.AttributeExpr{Type: c.typ, Nullable: nullable}}
					targetType := &expr.UserTypeExpr{TypeName: "Target", AttributeExpr: &expr.AttributeExpr{Type: c.typ, Nullable: nullable}}
					source := &expr.AttributeExpr{Type: sourceType}
					target := &expr.AttributeExpr{Type: targetType}
					if !nullable {
						source = &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "SourcePayload", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{{Name: "values", Attribute: source}}}}}
						target = &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "TargetPayload", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{{Name: "values", Attribute: target}}}}}
					}
					sourceCtx := NewAttributeContext(false, false, true, "", NewNameScope())
					targetCtx := NewAttributeContext(false, false, true, "", NewNameScope())
					targetCtx.JSONPresence = !nullable
					code, _, err := GoTransform(source, target, "source", "target", sourceCtx, targetCtx, "convert", true)
					require.NoError(t, err)
					require.Contains(t, code, ".SetValue(")
					require.NotContains(t, code, " = loom.NullableValue(")
					require.NotContains(t, code, " = loom.OptionalValue(")
					signature, body := "source *SourcePayload) *TargetPayload", optionalCollectionModel
					if nullable {
						signature, body = "source loom.Nullable[Source]) loom.Nullable[Target]", nullableCollectionModel
					}
					model := strings.NewReplacer("TYPE_DEFINITION", c.definition, "SIGNATURE", signature, "TRANSFORM", code, "CHECK_STATES", body, "POPULATED", c.populated).Replace(namedCollectionModel)
					dir := t.TempDir()
					module := fmt.Sprintf("module example.com/presencemodel\n\ngo 1.27.0\n\nrequire github.com/CaliLuke/loom v0.0.0\nreplace github.com/CaliLuke/loom => %s\n", testingx.RepoRoot())
					require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(module), 0o600))
					require.NoError(t, os.WriteFile(filepath.Join(dir, "presence_test.go"), []byte(model), 0o600))
					output, err := testingx.RunCmd(dir, "go", "mod", "tidy")
					require.NoError(t, err, output)
					output, err = testingx.RunCmd(dir, "go", "test", "./...")
					require.NoError(t, err, output)
				})
			}
		})
	}
}

// TestNamedCollectionDefaultTransform checks default application separately from
// null and concrete values, which must keep their authored state.
func TestNamedCollectionDefaultTransform(t *testing.T) {
	cases := []struct {
		name                  string
		typ                   expr.DataType
		definition, populated string
		defaultValue          any
	}{
		{"array", &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}, "[]string", `["a","b"]`, []string{"a", "b"}},
		{"map", &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.Int}}, "map[string]int", `{"a":7}`, map[string]int{"a": 7}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sourceType := &expr.UserTypeExpr{TypeName: "Source", AttributeExpr: &expr.AttributeExpr{Type: c.typ, Nullable: true}}
			targetType := &expr.UserTypeExpr{TypeName: "Target", AttributeExpr: &expr.AttributeExpr{Type: c.typ, Nullable: true}}
			source := &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "SourcePayload", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{{Name: "values", Attribute: &expr.AttributeExpr{Type: sourceType}}}}}}
			target := &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "TargetPayload", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{{Name: "values", Attribute: &expr.AttributeExpr{Type: targetType, DefaultValue: c.defaultValue}}}}}}
			context := NewAttributeContext(false, false, true, "", NewNameScope())
			code, _, err := GoTransform(source, target, "source", "target", context, context, "convert", true)
			require.NoError(t, err)
			require.NotContains(t, code, " = loom.NullableValue(")
			body := strings.ReplaceAll(nullableCollectionModel, "convert(", "convertValue(")
			body = strings.Replace(body, `if convertValue(loom.Nullable[Source]{}).Present() {
  t.Error("absent value became present")
 }`, `defaulted, ok := convertValue(loom.Nullable[Source]{}).Value()
 if !ok { t.Error("default was not applied") }
 checkValue(t, defaulted, populated)`, 1)
			model := strings.NewReplacer("TYPE_DEFINITION", c.definition, "SIGNATURE", "source *SourcePayload) *TargetPayload", "TRANSFORM", code, "CHECK_STATES", body, "POPULATED", c.populated, "Values Source", "Values loom.Nullable[Source]", "loom.Optional[Target]", "loom.Nullable[Target]").Replace(namedCollectionModel)
			model += `
func convertValue(value loom.Nullable[Source]) loom.Nullable[Target] {
 return convert(&SourcePayload{Values: value}).Values
}
`
			dir := t.TempDir()
			module := fmt.Sprintf("module example.com/presencemodel\n\ngo 1.27.0\n\nrequire github.com/CaliLuke/loom v0.0.0\nreplace github.com/CaliLuke/loom => %s\n", testingx.RepoRoot())
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(module), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "presence_test.go"), []byte(model), 0o600))
			output, err := testingx.RunCmd(dir, "go", "mod", "tidy")
			require.NoError(t, err, output)
			output, err = testingx.RunCmd(dir, "go", "test", "./...")
			require.NoError(t, err, output)
		})
	}
}

const namedCollectionModel = `package presence_test

import (
 "encoding/json/v2"
 "reflect"
 "testing"
 loom "github.com/CaliLuke/loom/pkg"
)

type Source TYPE_DEFINITION
type Target TYPE_DEFINITION
type SourcePayload struct { Values Source }
type TargetPayload struct { Values loom.Optional[Target] }

func convert(SIGNATURE {
 TRANSFORM
 return target
}

func TestStates(t *testing.T) {
 var populated Source
 if err := json.Unmarshal([]byte(` + "`POPULATED`" + `), &populated); err != nil {
  t.Fatal(err)
 }
 CHECK_STATES
}

func checkValue(t *testing.T, got Target, want Source) {
 t.Helper()
 if !reflect.DeepEqual(Source(got), want) {
  t.Errorf("value=%#v want=%#v", got, want)
 }
}
`

const nullableCollectionModel = `
 if convert(loom.Nullable[Source]{}).Present() {
  t.Error("absent value became present")
 }
 if !convert(loom.NullValue[Source]()).IsNull() {
  t.Error("null state lost")
 }
 for _, value := range []Source{Source{}, populated} {
  got := convert(loom.NullableValue(value))
  actual, ok := got.Value()
  if !ok || got.IsNull() {
   t.Error("concrete value lost")
  }
  checkValue(t, actual, value)
 }
`

const optionalCollectionModel = `
 if convert(&SourcePayload{}).Values.Present() {
  t.Error("absent value became present")
 }
 for _, value := range []Source{Source{}, populated} {
  got := convert(&SourcePayload{Values:value})
  actual, ok := got.Values.Value()
  if !ok {
   t.Error("concrete value lost")
  }
  checkValue(t, actual, value)
 }
`
