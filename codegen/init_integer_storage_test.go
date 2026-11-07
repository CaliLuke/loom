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

func TestInitIntegerStorageContract(t *testing.T) {
	for _, storage := range []string{"float32", "float64", "complex64", "string", "bool"} {
		t.Run(storage, func(t *testing.T) {
			target := &expr.AttributeExpr{Type: expr.Int64, Meta: expr.MetaExpr{"struct:field:type": {storage}}}
			_, err := InitStructFields([]*InitArgData{{Name: "value", Type: expr.Int64, FieldName: "Value", FieldType: expr.Int64, FieldAttribute: target}}, "v", "", "", NewNameScope())
			require.ErrorContains(t, err, "integer storage")
		})
	}
}

func TestInitOpaqueIntegerCapability(t *testing.T) {
	target := &expr.AttributeExpr{Type: expr.Int64, Meta: expr.MetaExpr{"struct:field:type": {"External"}}}
	plan, err := InitStructFields([]*InitArgData{{Name: "value", Type: expr.Int64, FieldName: "Value", FieldType: expr.Int64, FieldAttribute: target}}, "v", "", "", NewNameScope())
	require.NoError(t, err)
	require.Contains(t, plan.Code, "const _ = ^External(0)")
	for _, underlying := range []string{"int64", "float64"} {
		t.Run(underlying, func(t *testing.T) {
			source := fmt.Sprintf("package capability\nimport \"fmt\"\ntype External %s\nfunc convert(value int64)(External,error){\nv:=struct{Value External}{}\ntransformErr:=new(error)\n%s\nreturn v.Value,*transformErr\n}\n", underlying, plan.Code)
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/capability\n\ngo 1.27.3\n"), 0600))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "capability.go"), []byte(source), 0600))
			output, err := testingx.RunCmd(dir, "go", "build", "./...")
			if underlying == "int64" {
				require.NoError(t, err, output)
			} else {
				require.Error(t, err, "opaque float storage must not compile as integer storage")
				t.Logf("opaque float compile rejection: %s", output)
				require.True(t, strings.Contains(output, "^") && strings.Contains(output, "float64"), output)
			}
		})
	}
}
