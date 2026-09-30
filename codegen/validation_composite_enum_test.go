package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestCompositeEnumValidationCompilesAndExecutes(t *testing.T) {
	object := &expr.AttributeExpr{
		Type: &expr.Object{{Name: "s", Attribute: &expr.AttributeExpr{Type: expr.String}}},
		Validation: &expr.ValidationExpr{Values: []any{
			map[string]any{"s": "x"},
		}},
	}
	mapping := &expr.AttributeExpr{
		Type: &expr.Map{
			KeyType:  &expr.AttributeExpr{Type: expr.String},
			ElemType: &expr.AttributeExpr{Type: expr.Float32},
		},
		Validation: &expr.ValidationExpr{Values: []any{
			map[string]float64{"score": 1.23456789},
		}},
	}
	union := &expr.AttributeExpr{
		Type: &expr.Union{TypeName: "Value", Values: []*expr.NamedAttributeExpr{
			{Name: "score", Attribute: &expr.AttributeExpr{Type: expr.Float32}},
			{Name: "text", Attribute: &expr.AttributeExpr{Type: expr.String}},
		}},
		Validation: &expr.ValidationExpr{Values: []any{float64(1.23456789)}},
	}
	ctx := NewAttributeContext(true, false, true, "", NewNameScope())
	objectCode := validationCode(object, ctx, true, false, "target", "object")
	mapCode := validationCode(mapping, ctx, true, false, "target", "map")
	unionCode := validationCode(union, ctx, true, false, "target", "union")
	for _, code := range []string{objectCode, mapCode, unionCode} {
		require.Contains(t, code, "loom.JSONValueFrom(target)")
		require.Contains(t, code, "loom.JSONValueEqual(encoded,")
	}

	source := fmt.Sprintf(`package composite
import (
 "testing"
 loom "github.com/CaliLuke/loom/pkg"
)
type Record struct { S string %[1]sjson:"s"%[1]s }
func validateObject(target *Record) (err error) {
%[2]s
return
}
func validateMap(target map[string]float32) (err error) {
%[3]s
return
}
func validateUnion(target loom.JSONValue) (err error) {
%[4]s
return
}
func TestComposite(t *testing.T) {
 if err := validateObject(&Record{S:"x"}); err != nil { t.Errorf("valid object: %%v", err) }
 if err := validateObject(&Record{S:"y"}); err == nil { t.Error("invalid object accepted") }
 if err := validateMap(map[string]float32{"score":1.23456789}); err != nil { t.Errorf("valid map: %%v", err) }
 if err := validateMap(map[string]float32{"score":1.2345677}); err == nil { t.Error("invalid map accepted") }
 if err := validateUnion(loom.JSONValue(%[1]s{"type":"score","value":1.2345679}%[1]s)); err != nil { t.Errorf("valid union: %%v", err) }
 if err := validateUnion(loom.JSONValue(%[1]s{"type":"score","value":1.2345677}%[1]s)); err == nil { t.Error("invalid union accepted") }
}
`, "`", objectCode, mapCode, unionCode)
	dir := t.TempDir()
	module := fmt.Sprintf("module example.com/composite\n\ngo 1.27\nrequire github.com/CaliLuke/loom v0.0.0\nreplace github.com/CaliLuke/loom => %s\n", testingx.RepoRoot())
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(module), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "composite_test.go"), []byte(source), 0o600))
	output, err := testingx.RunCmd(dir, "go", "mod", "tidy")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(dir, "go", "test", "./...")
	require.NoError(t, err, output)
}
