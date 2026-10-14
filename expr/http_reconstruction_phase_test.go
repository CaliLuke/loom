package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequiredReconstructionPhase(t *testing.T) {
	for _, selected := range []bool{false, true} {
		t.Run(map[bool]string{false: "listed", true: "selected"}[selected], func(t *testing.T) {
			source := &AttributeExpr{Type: &Object{
				{Name: "sent:s", Attribute: &AttributeExpr{Type: String}},
				{Name: "lost:l", Attribute: &AttributeExpr{Type: String, DefaultValue: "fallback"}},
			}, Validation: &ValidationExpr{Required: []string{"sent:s", "lost:l"}}}
			body := &AttributeExpr{Type: &Object{{Name: "sent:wire", Attribute: &AttributeExpr{Type: String}}}, References: []DataType{source.Type}}
			if selected {
				body = &AttributeExpr{Type: String, Meta: MetaExpr{"origin:attribute": {"sent"}}}
			}
			require.Equal(t, []string{"lost:l"}, missingReconstructionFields(source, body, nil))
			require.Empty(t, missingReconstructionFields(source, body, []string{"lost"}))
			source.Finalize()
			body.Finalize()
			require.Equal(t, []string{"lost:l"}, missingReconstructionFields(source, body, nil))
			require.Empty(t, missingReconstructionFields(source, body, []string{"lost:l"}))
		})
	}
}
