package ir

import (
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/testingx"
	"github.com/stretchr/testify/require"
	"github.com/xeipuuv/gojsonschema"
)

// TestEmittedBoundsMatchGeneratedValidation compares schema acceptance with
// compiled generated validators. It does not infer runtime behavior from bytes.
func TestEmittedBoundsMatchGeneratedValidation(t *testing.T) {
	minimum, maximum := float64(2), float64(17)
	minLength, maxLength := 2, 3
	tests := []struct {
		name           string
		attribute      *expr.AttributeExpr
		valid, invalid []string
	}{
		{name: "map", attribute: &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.String}}, Validation: &expr.ValidationExpr{MinLength: &minLength, MaxLength: &maxLength}}, valid: []string{`{"a":"a","b":"b"}`, `{"a":"a","b":"b","c":"c"}`}, invalid: []string{`{"a":"a"}`, `{"a":"a","b":"b","c":"c","d":"d"}`}},
		{name: "uint32", attribute: &expr.AttributeExpr{Type: expr.UInt32, Validation: &expr.ValidationExpr{Minimum: boundForTest(minimum), Maximum: boundForTest(maximum)}}, valid: []string{"2", "17"}, invalid: []string{"1", "18"}},
		{name: "uint64", attribute: &expr.AttributeExpr{Type: expr.UInt64, Validation: &expr.ValidationExpr{Minimum: boundForTest(minimum), Maximum: boundForTest(maximum)}}, valid: []string{"2", "17"}, invalid: []string{"1", "18"}},
		{name: "float64", attribute: &expr.AttributeExpr{Type: expr.Float64, Validation: &expr.ValidationExpr{Minimum: boundForTest(minimum), Maximum: boundForTest(maximum)}}, valid: []string{"2", "2.5", "17"}, invalid: []string{"1.5", "17.5"}},
	}
	var source strings.Builder
	source.WriteString("package bounds\nimport (\n\"encoding/json/v2\"\n\"testing\"\nloom \"github.com/CaliLuke/loom/pkg\"\n)\n")
	for _, tc := range tests {
		analyzer := NewAnalyzer(expr.NewRandom(tc.name), false)
		schema := analyzer.AnalyzeSchema(tc.attribute)
		require.NoError(t, analyzer.Err())
		encoded, err := json.Marshal(RenderSchema(schema))
		require.NoError(t, err)
		compiled, err := gojsonschema.NewSchema(gojsonschema.NewBytesLoader(encoded))
		require.NoError(t, err)
		scope := codegen.NewNameScope()
		ctx := codegen.NewAttributeContext(false, false, false, "", scope)
		validation := codegen.ValidationCode(tc.attribute, nil, ctx, true, false, false, "value")
		typeRef := scope.GoTypeRef(tc.attribute)
		fmt.Fprintf(&source, "func validate_%s(value %s) (err error) {\n%s\nreturn\n}\n", tc.name, typeRef, validation)
		fmt.Fprintf(&source, "func Test_%s(t *testing.T) {\n", tc.name)
		for _, group := range []struct {
			values []string
			valid  bool
		}{{tc.valid, true}, {tc.invalid, false}} {
			for _, wire := range group.values {
				result, err := compiled.Validate(gojsonschema.NewStringLoader(wire))
				require.NoError(t, err)
				require.Equal(t, group.valid, result.Valid(), "%s schema with %s: %v", tc.name, wire, result.Errors())
				fmt.Fprintf(&source, "{\nvar value %s\nif err := json.Unmarshal([]byte(%q), &value); err != nil {\nt.Fatal(err)\n}\nif err := validate_%s(value); (err == nil) != %t {\nt.Errorf(%q, err)\n}\n}\n", typeRef, wire, tc.name, group.valid, "runtime disagrees for "+wire+": %v")
			}
		}
		source.WriteString("}\n")
	}
	dir := t.TempDir()
	module := fmt.Sprintf("module example.com/bounds\n\ngo 1.27\nrequire github.com/CaliLuke/loom v0.0.0\nreplace github.com/CaliLuke/loom => %s\n", testingx.RepoRoot())
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(module), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bounds_test.go"), []byte(source.String()), 0600))
	output, err := testingx.RunCmd(dir, "go", "test", "-mod=mod", "-count=1", "./...")
	require.NoError(t, err, output)
}
