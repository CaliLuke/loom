package codegen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

func TestFormLeafDecodeWithoutFieldsCompiles(t *testing.T) {
	for _, tc := range []struct {
		name   string
		object expr.Object
		fields map[string]httpencoding.FieldCodec
	}{
		{name: "empty object"},
		{
			name:   "all fields omitted",
			object: expr.Object{{Name: "ignored", Attribute: &expr.AttributeExpr{Type: expr.String}}},
			fields: map[string]httpencoding.FieldCodec{"ignored": {Omitted: true}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := &TypeData{attribute: &expr.AttributeExpr{Type: &tc.object}}
			contract := &expr.HTTPFormEncoding{Fields: tc.fields}
			source := "package generated\nfunc decode(destination *struct{}) error {\n" + renderFormLeafDecode(body, contract, "destination") + "return nil\n}\n"
			files := token.NewFileSet()
			file, err := parser.ParseFile(files, "decode.go", source, 0)
			require.NoError(t, err)
			config := &types.Config{}
			_, err = config.Check("generated", files, []*ast.File{file}, nil)
			require.NoError(t, err, source)
		})
	}
}
