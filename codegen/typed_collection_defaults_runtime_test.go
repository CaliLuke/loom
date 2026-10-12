//nolint:errcheck // Generator test writes only to an in-memory builder.
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

func TestTypedCollectionDefaultRuntime(t *testing.T) {
	var model strings.Builder
	model.WriteString(`package defaults

import (
 "reflect"
 "testing"
 loom "github.com/CaliLuke/loom/pkg"
)
`)
	for _, carrier := range []string{"Native", "Optional", "Nullable"} {
		field := &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Float32}}}
		field.Nullable = carrier == "Nullable"
		source := &expr.AttributeExpr{Type: &expr.UserTypeExpr{
			TypeName:      "Source" + carrier,
			AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{{Name: "values", Attribute: expr.DupAtt(field)}}},
		}}
		field.DefaultValue = []float64{1.5}
		target := &expr.AttributeExpr{Type: &expr.UserTypeExpr{
			TypeName:      "Target" + carrier,
			AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{{Name: "values", Attribute: field}}},
		}}
		scope := NewNameScope()
		sourceCtx := NewAttributeContext(false, false, false, "", scope)
		sourceCtx.JSONPresence = carrier == "Optional"
		targetCtx := NewAttributeContext(false, false, true, "", scope)
		code, _, err := GoTransform(source, target, "source", "target", sourceCtx, targetCtx, "", true)
		require.NoError(t, err)
		sourceRef, targetRef := "[]float32", "[]float32"
		if carrier == "Optional" {
			sourceRef = "loom.Optional[[]float32]"
		}
		if carrier == "Nullable" {
			sourceRef, targetRef = "loom.Nullable[[]float32]", "loom.Nullable[[]float32]"
		}
		fmt.Fprintf(&model, "type Source%s struct {\n Values %s\n}\ntype Target%s struct {\n Values %s\n}\n", carrier, sourceRef, carrier, targetRef)
		fmt.Fprintf(&model, "func convert%s(source *Source%s) *Target%s {\n%s\nreturn target\n}\n", carrier, carrier, carrier, code)
	}
	model.WriteString(typedCollectionDefaultRuntimeChecks)
	dir := t.TempDir()
	module := fmt.Sprintf("module example.com/defaults\n\ngo 1.27.0\n\nrequire github.com/CaliLuke/loom v1.10.0-alpha.9.0.20261012201303-8c7d8d711b9e\nreplace github.com/CaliLuke/loom => %s\n", testingx.RepoRoot())
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(module), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "defaults_test.go"), []byte(model.String()), 0o600))
	for _, args := range [][]string{{"mod", "tidy"}, {"test", "-race", "./..."}, {"vet", "./..."}} {
		output, err := testingx.RunCmd(dir, "go", args...)
		require.NoError(t, err, output)
	}
}

const typedCollectionDefaultRuntimeChecks = `
func TestDefaultStates(t *testing.T) {
 want := []float32{1.5}
 if got := convertNative(&SourceNative{}).Values; !reflect.DeepEqual(got, want) {
  t.Errorf("native default=%v", got)
 }
 if got := convertOptional(&SourceOptional{}).Values; !reflect.DeepEqual(got, want) {
  t.Errorf("optional default=%v", got)
 }
 got, ok := convertNullable(&SourceNullable{}).Values.Value()
 if !ok || !reflect.DeepEqual(got, want) {
  t.Errorf("nullable default=%v, present=%v", got, ok)
 }
 for _, supplied := range [][]float32{{}, {2.5}} {
  if got := convertNative(&SourceNative{Values: supplied}).Values; !reflect.DeepEqual(got, supplied) {
   t.Errorf("native supplied=%v, got=%v", supplied, got)
  }
  if got := convertOptional(&SourceOptional{Values: loom.OptionalValue(supplied)}).Values; !reflect.DeepEqual(got, supplied) {
   t.Errorf("optional supplied=%v, got=%v", supplied, got)
  }
  got, ok := convertNullable(&SourceNullable{Values: loom.NullableValue(supplied)}).Values.Value()
  if !ok || !reflect.DeepEqual(got, supplied) {
   t.Errorf("nullable supplied=%v, got=%v, present=%v", supplied, got, ok)
  }
 }
 if !convertNullable(&SourceNullable{Values: loom.NullValue[[]float32]()} ).Values.IsNull() {
  t.Error("nullable null was defaulted")
 }
}
`
