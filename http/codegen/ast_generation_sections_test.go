package codegen

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	servicecodegen "github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/codegen/testutil"
)

func TestHTTPUnionSectionStructuredDeclarations(t *testing.T) {
	code := codegen.SectionCode(t, unionTypeSection("server-union-type", &servicecodegen.UnionTypeData{
		Name:     "Selection",
		KindName: "SelectionKind",
		TypeKey:  "type",
		ValueKey: "value",
		Fields: []*servicecodegen.UnionFieldData{
			{
				Name:               "text",
				KindConst:          "SelectionKindText",
				FieldName:          "Text",
				FieldType:          "SelectionText",
				EmitPrimitiveAlias: true,
				PrimitiveAliasType: "string",
				TypeTag:            "text",
			},
			{
				Name:      "payload",
				KindConst: "SelectionKindPayload",
				FieldName: "Payload",
				FieldType: "*PayloadBody",
				TypeTag:   "payload",
			},
		},
	}))

	require.Contains(t, code, "type SelectionText string")
	require.Contains(t, code, "type Selection struct {")
	testutil.NewGoldenFile(t, filepath.Join("testdata", "golden")).
		StringContent(code).
		Path("union_type_section_selection.golden").
		CompareContent()
}
