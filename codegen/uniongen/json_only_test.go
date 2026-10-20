package uniongen

import (
	"bytes"
	"testing"

	"github.com/dave/jennifer/jen"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
)

func TestJSONOnlyUnionEmission(t *testing.T) {
	for _, jsonOnly := range []bool{false, true} {
		t.Run(map[bool]string{false: "JSON and form", true: "JSON only"}[jsonOnly], func(t *testing.T) {
			data := &Type{
				Name: "Choice", KindName: "ChoiceKind", TypeKey: "kind", ValueKey: "value",
				JSONOnly: jsonOnly,
				Fields:   []*Field{{Name: "text", KindConst: "ChoiceText", FieldName: "Text", FieldType: "string", TypeTag: "text"}},
			}
			var output bytes.Buffer
			section := &codegen.JenniferSection{Build: func(stmt *jen.Statement) {
				Emit(stmt, data)
			}}
			require.NoError(t, section.Write(&output))
			for _, name := range []string{"MarshalJSON", "UnmarshalJSON", "AsText", "SetText", "Validate"} {
				require.Contains(t, output.String(), name)
			}
			if jsonOnly {
				require.NotContains(t, output.String(), "MarshalForm")
				require.NotContains(t, output.String(), "UnmarshalForm")
				require.NotContains(t, output.String(), "loomhttp")
			} else {
				require.Contains(t, output.String(), "MarshalForm")
				require.Contains(t, output.String(), "UnmarshalForm")
			}
		})
	}
}
