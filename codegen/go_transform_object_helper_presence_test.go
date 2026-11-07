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

func TestObjectTransformHelpersPreserveNil(t *testing.T) {
	var generated strings.Builder
	generated.WriteString("package helpers\nimport \"testing\"\n")
	index := 0
	for _, empty := range []bool{true, false} {
		for _, requiredFirst := range []bool{true, false} {
			sourceName, targetName := fmt.Sprintf("Source%d", index), fmt.Sprintf("Target%d", index)
			object := func(name string) *expr.AttributeExpr {
				fields := &expr.Object{}
				attribute := &expr.AttributeExpr{Type: fields}
				if !empty {
					*fields = append(*fields, &expr.NamedAttributeExpr{Name: "text", Attribute: &expr.AttributeExpr{Type: expr.String}})
					attribute.Validation = &expr.ValidationExpr{Required: []string{"text"}}
				}
				return &expr.AttributeExpr{Type: &expr.UserTypeExpr{
					TypeName: name, UID: name,
					AttributeExpr: attribute,
				}}
			}
			wrap := func(child *expr.AttributeExpr) *expr.AttributeExpr {
				fields := &expr.Object{
					{Name: "required", Attribute: child},
					{Name: "optional", Attribute: child},
				}
				if !requiredFirst {
					(*fields)[0], (*fields)[1] = (*fields)[1], (*fields)[0]
				}
				return &expr.AttributeExpr{Type: fields, Validation: &expr.ValidationExpr{Required: []string{"required"}}}
			}
			context := NewAttributeContext(false, false, true, "", NewNameScope())
			plan, err := GoTransform(wrap(object(sourceName)), wrap(object(targetName)), "source", "target", context, context, "convert", true)
			require.NoError(t, err)
			require.Len(t, plan.Helpers, 1, "both occurrences must share one helper")
			helper := plan.Helpers[0]
			fields := ""
			if !empty {
				fields = "Text string"
			}
			fmt.Fprintf(&generated, "type %s struct {%s}\ntype %s struct {%s}\n", sourceName, fields, targetName, fields)
			fmt.Fprintf(&generated, "func %s(v %s) %s {\n%s\nreturn res\n}\n", helper.Name, helper.ParamTypeRef, helper.ResultTypeRef, helper.Code)
			fmt.Fprintf(&generated, "func TestCase%d(t *testing.T) {\nif got := %s(nil); got != nil {t.Error(\"nil object became supplied\")}\n", index, helper.Name)
			fmt.Fprintf(&generated, "source := &%s{}\ngot := %s(source)\nif got == nil {t.Fatal(\"supplied empty object disappeared\")}\n", sourceName, helper.Name)
			if !empty {
				fmt.Fprintf(&generated, "source.Text = \"value\"\ngot = %s(source)\nif got == nil || got.Text != source.Text {t.Error(\"supplied field changed\")}\n", helper.Name)
			}
			generated.WriteString("}\n")
			index++
		}
	}
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/helpers\n\ngo 1.27.3\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "helpers_test.go"), []byte(generated.String()), 0o600))
	output, err := testingx.RunCmd(dir, "go", "test", "./...")
	require.NoError(t, err, output)
}
