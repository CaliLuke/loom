package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	loomcodegen "github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func TestCustomMetadataTextStorage(t *testing.T) {
	for _, scalar := range []bool{true, false} {
		t.Run(map[bool]string{true: "scalar", false: "array"}[scalar], func(t *testing.T) {
			field := &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:field:type": {"uuid.UUID", "github.com/google/uuid"}}}
			attribute := field
			if !scalar {
				attribute = &expr.AttributeExpr{Type: &expr.Array{ElemType: field}}
			}
			storage, err := metadataStorage(attribute, false, loomcodegen.NewNameScope())
			require.NoError(t, err)
			physical := storage.attribute
			if !scalar {
				physical = expr.AsArray(physical.Type).ElemType
			}
			require.Equal(t, "string", loomcodegen.NewNameScope().GoTypeRef(physical))
			require.Equal(t, []string{"uuid.UUID", "github.com/google/uuid"}, field.Meta["struct:field:type"], "staging must not mutate authored occurrence")
		})
	}
}
