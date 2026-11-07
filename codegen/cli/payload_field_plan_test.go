package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/testingx"
	"github.com/stretchr/testify/require"
)

func TestPayloadBuilderOwnsFieldErrorPlan(t *testing.T) {
	source := &expr.AttributeExpr{Type: expr.Int}
	target := &expr.AttributeExpr{Type: expr.Int, Meta: expr.MetaExpr{"struct:field:type": {"int8"}}}
	data := &BuildFunctionData{Name: "BuildPayload", ResultType: "*Payload", Fields: []*FieldData{{VarName: "value", TypeRef: "int", Init: codegen.Expr("value = 200")}}, PayloadInit: &PayloadInitData{
		ReturnIsStruct: true, ReturnTypeName: "Payload",
		Args: []*codegen.InitArgData{{Name: "value", Type: expr.Int, Attribute: source, FieldName: "Value", FieldType: expr.Int, FieldAttribute: target}},
	}}
	text := codegen.SectionCode(t, PayloadBuilderSection(data, codegen.NewNameScope()))
	require.Contains(t, text, "transformErr := new(error)")
	require.Contains(t, text, "return zero, *transformErr")
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/fieldplan\n\ngo 1.27.3\n"), 0600))
	program := "package fieldplan\nimport (\"fmt\";\"testing\")\ntype Payload struct{Value int8}\n" + text + "\nfunc TestRejected(t *testing.T){v,err:=BuildPayload();if err==nil||v!=nil{t.Fatalf(\"value=%v err=%v\",v,err)}}\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "fieldplan_test.go"), []byte(program), 0600))
	output, err := testingx.RunCmd(dir, "go", "test", "./...")
	require.NoError(t, err, output)
}
