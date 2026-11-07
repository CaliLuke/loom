package uniongen_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/dave/jennifer/jen"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/uniongen"
	"github.com/CaliLuke/loom/codegen/unionnames"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestPluginUnionMemberNamesCompile(t *testing.T) {
	union := &expr.Union{TypeName: "Choice"}
	for _, name := range []string{"Kind", "Kind2", "a-b", "a_b", "AB2", "MarshalJSON"} {
		union.Values = append(union.Values, &expr.NamedAttributeExpr{Name: name, Attribute: &expr.AttributeExpr{Type: expr.String}})
	}
	members := unionnames.Resolve(union)
	model := &uniongen.Type{Name: "Choice", KindName: "ChoiceKind", TypeKey: "type", ValueKey: "value", JSONOnly: true}
	for _, branch := range union.Values {
		member := members[branch.Name]
		model.Fields = append(model.Fields, &uniongen.Field{
			Name: branch.Name, TypeTag: branch.Name, KindConst: model.KindName + member.Method,
			FieldName: member.Field, MethodName: member.Method, FieldType: "string",
		})
	}
	var output bytes.Buffer
	section := &codegen.JenniferSection{Build: func(stmt *jen.Statement) {
		uniongen.Emit(stmt, model)
	}}
	require.NoError(t, section.Write(&output))
	require.NotContains(t, output.String(), "MarshalFormValues")
	require.Contains(t, output.String(), "AsKind()")
	dir := t.TempDir()
	source := "package plugin\nimport (\"fmt\"; json \"encoding/json/v2\"; \"encoding/json/jsontext\"; loom \"github.com/CaliLuke/loom/pkg\")\n" + output.String()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "union.go"), []byte(source), 0600))
	module := fmt.Sprintf("module example.com/plugin\n\ngo 1.27.0\n\nrequire github.com/CaliLuke/loom v1.0.0\nreplace github.com/CaliLuke/loom => %s\n", testingx.RepoRoot())
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(module), 0600))
	log, err := testingx.RunCmd(dir, "go", "test", "-mod=mod", "./...")
	require.NoError(t, err, log)
}
