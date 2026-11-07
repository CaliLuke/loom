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

func TestInitPrimitiveRepresentability(t *testing.T) {
	cases := []struct {
		name                         string
		source                       expr.Primitive
		sourceRef, targetRef, values string
		pointer, checked             bool
	}{
		{"signed_narrow", expr.Int, "int", "int8", "-129, -128, 127, 128, 200", false, true},
		{"signed_unsigned", expr.Int, "int", "uint8", "-1, 0, 200, 255, 256", false, true},
		{"unsigned_signed", expr.UInt64, "uint64", "int64", "0, 9223372036854775807, 9223372036854775808, 18446744073709551615", false, true},
		{"pointer", expr.Int, "int", "int8", "-129, -128, 127, 200", true, true},
		{"opaque_alias", expr.Int, "int", "Small", "-129, -128, 127, 200", false, true},
		{"widen", expr.Int, "int", "int64", "-200, 0, 200", false, false},
	}
	var generated strings.Builder
	generated.WriteString("package conversion\nimport (\"fmt\"; \"errors\"; \"testing\")\ntype Small int8\n")
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			target := &expr.AttributeExpr{Type: tc.source, Meta: expr.MetaExpr{"struct:field:type": {tc.targetRef}}}
			plan, err := InitStructFields([]*InitArgData{{Name: "input", Type: tc.source, Pointer: tc.pointer, FieldName: "Value", FieldType: tc.source, FieldAttribute: target, FieldPointer: tc.pointer}}, "v", "", "", NewNameScope())
			require.NoError(t, err)
			require.Equal(t, tc.checked, plan.ErrorAware)
			sourceRef, targetRef := tc.sourceRef, tc.targetRef
			if tc.pointer {
				sourceRef = "*" + sourceRef
				targetRef = "*" + targetRef
			}
			fmt.Fprintf(&generated, "func convert%d(input %s, previous error) (%s,error) {\nv:=struct{Value %s}{}\n", i, sourceRef, targetRef, targetRef)
			if plan.ErrorAware {
				generated.WriteString("transformErr:= &previous\n")
			}
			generated.WriteString(plan.Code)
			generated.WriteString("return v.Value,previous\n}\n")
			call, actual := "value", "actual"
			zero := "actual != 0"
			if tc.pointer {
				call = "&value"
				actual = "*actual"
				zero = "actual != nil"
			}
			predicate := "value >= -128 && value <= 127"
			switch tc.targetRef {
			case "uint8":
				predicate = "value >= 0 && value <= 255"
			case "int64":
				if tc.source == expr.UInt64 {
					predicate = "value <= 9223372036854775807"
				} else {
					predicate = "true"
				}
			}
			fmt.Fprintf(&generated, `func TestCase%d(t *testing.T) {
for _,value:=range []%s{%s} {
 actual,err:=convert%d(%s,nil)
 if %s {
  if err!=nil {t.Fatal(err)}
  if %s(%s)!=value {t.Fatalf("value changed: %%v",actual)}
 } else if err==nil || %s {t.Fatalf("loss not rejected: %%v / %%v",actual,err)}
}
`, i, tc.sourceRef, tc.values, i, call, predicate, tc.sourceRef, actual, zero)
			if tc.checked {
				input := "200"
				if tc.pointer {
					input = "nil"
				}
				fmt.Fprintf(&generated, "first:=errors.New(\"first\");actual,err:=convert%d(%s,first);if err!=first || %s {t.Fatal(\"first error changed or destination assigned\")}\n", i, input, zero)
			}
			if tc.pointer {
				fmt.Fprintf(&generated, "if actual,err:=convert%d(nil,nil);actual!=nil || err!=nil {t.Fatal(\"absent pointer changed\")}\n", i)
			}
			generated.WriteString("}\n")
		})
	}
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/conversion\n\ngo 1.27.3\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "conversion_test.go"), []byte(generated.String()), 0600))
	output, err := testingx.RunCmd(dir, "go", "test", "-count=1", "./...")
	require.NoError(t, err, output)
}

func TestInitPrimitiveRejectsNonnumericStorage(t *testing.T) {
	target := &expr.AttributeExpr{Type: expr.Int, Meta: expr.MetaExpr{"struct:field:type": {"string"}}}
	_, err := InitStructFields([]*InitArgData{{Name: "input", Type: expr.Int, FieldName: "Value", FieldType: expr.Int, FieldAttribute: target}}, "v", "", "", NewNameScope())
	require.ErrorContains(t, err, "integer storage, got string")
}
