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

// TestObjectDefaultCarrierRuntime compiles the shared transform independently
// of transports and checks absence, explicit zero and nullable null.
func TestObjectDefaultCarrierRuntime(t *testing.T) {
	for _, carrier := range []string{"Native", "Optional", "Nullable"} {
		t.Run(carrier, func(t *testing.T) {
			leaf := &expr.AttributeExpr{Type: &expr.Object{{Name: "x", Attribute: &expr.AttributeExpr{Type: expr.Int}}}, Validation: &expr.ValidationExpr{Required: []string{"x"}}}
			object := expr.AsObject(leaf.Type)
			*object = append(*object,
				&expr.NamedAttributeExpr{Name: "state", Attribute: &expr.AttributeExpr{Type: expr.Any, Nullable: true}},
				&expr.NamedAttributeExpr{Name: "nested", Attribute: &expr.AttributeExpr{Type: &expr.Object{{Name: "y", Attribute: &expr.AttributeExpr{Type: expr.Int}}}, Nullable: true}},
			)
			leaf.Nullable = carrier == "Nullable"
			source := &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Source", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: expr.DupAtt(leaf)}}}}}
			leaf.DefaultValue = map[string]any{"x": 1, "state": nil, "nested": map[string]any{"y": 2}}
			target := &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Target", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: leaf}}}}}
			scope := NewNameScope()
			sourceCtx := NewAttributeContext(false, false, false, "", scope)
			sourceCtx.JSONPresence = carrier == "Optional"
			targetCtx := NewAttributeContext(false, false, true, "", scope)
			code, helpers, err := GoTransform(source, target, "source", "target", sourceCtx, targetCtx, "", true)
			require.NoError(t, err)
			require.Empty(t, helpers)
			concrete := scope.GoTypeName(concretePresenceAttribute(leaf))
			sourceRef, targetRef := "*"+concrete, "*"+concrete
			if carrier == "Optional" {
				sourceRef = "loom.Optional[" + concrete + "]"
			}
			readValue := "value := result.Value"
			if carrier == "Nullable" {
				sourceRef, targetRef = "loom.Nullable["+concrete+"]", "loom.Nullable["+concrete+"]"
				readValue = "value, present := result.Value.Value()\nif !present {\n t.Fatal(\"expected value\")\n}"
			}
			nullCheck := ""
			if carrier == "Nullable" {
				nullCheck = `var source Source
 if err := json.Unmarshal([]byte("{\"value\":null}"), &source); err != nil {
  t.Fatal(err)
 }
 if !convert(&source).Value.IsNull() {
  t.Error("null was replaced by default")
 }`
			}
			model := fmt.Sprintf(`package defaults
import (
 json "encoding/json/v2"
 "testing"
 loom "github.com/CaliLuke/loom/pkg"
)
var _ loom.Optional[int]
type Source struct {
 Value %s `+"`json:\"value\"`"+`
}
type Target struct {
 Value %s `+"`json:\"value\"`"+`
}
func convert(source *Source) *Target {
%s
return target
}
func TestStates(t *testing.T) {
 for _, tc := range []struct { input string; want int }{{"{}",1},{"{\"value\":{\"x\":0}}",0},{"{\"value\":{\"x\":2}}",2}} {
  var source Source
  if err := json.Unmarshal([]byte(tc.input), &source); err != nil {
   t.Fatal(err)
  }
  result := convert(&source)
  %s
  if tc.input == "{}" {
   if !value.State.IsNull() {
    t.Error("nested nullable Any default lost null")
   }
   nested, present := value.Nested.Value()
   if !present || nested.Y == nil || *nested.Y != 2 {
    t.Error("nested nullable object default lost its pointer field")
   }
  }
  if value.X != tc.want {
   t.Errorf("input=%%s got=%%d want=%%d", tc.input, value.X, tc.want)
  }
 }
 %s
}
`, sourceRef, targetRef, code, readValue, nullCheck)
			dir := t.TempDir()
			module := fmt.Sprintf("module example.com/objectdefaultcarriers\n\ngo 1.27.0\n\nrequire github.com/CaliLuke/loom v1.10.0-alpha.9.0.20261012214204-66a9dc9090a0\nreplace github.com/CaliLuke/loom => %s\n", testingx.RepoRoot())
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(module), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "defaults_test.go"), []byte(model), 0o600))
			for _, args := range [][]string{{"mod", "tidy"}, {"test", "-race", "./..."}, {"vet", "./..."}} {
				output, err := testingx.RunCmd(dir, "go", args...)
				require.NoError(t, err, strings.TrimSpace(output))
			}
		})
	}
}
