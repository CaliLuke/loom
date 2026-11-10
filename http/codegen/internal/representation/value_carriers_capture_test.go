package representation

import (
	"testing"

	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestRuntimeTargetDefersUnusedExampleCapture(t *testing.T) {
	for _, test := range []struct {
		name                   string
		schema, local, capture bool
	}{
		{"runtime inherited source", false, false, false},
		{"runtime authored override", false, true, true},
		{"schema annotations", true, false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			context := expr.NewValueContext()
			attribute := &expr.AttributeExpr{Type: expr.String}
			occurrence, err := context.NewOccurrence(attribute)
			require.NoError(t, err)
			target := expr.DupAtt(attribute)
			if test.local {
				target.UserExamples = []*expr.ExampleExpr{{Value: "local"}}
			}
			source := &service.ValueData{Context: context, Occurrence: occurrence}
			value := ValueTarget(source, target, "", expr.ValueCodecJSON, test.schema)
			require.NoError(t, value.Error)
			require.Equal(t, test.capture, value.ExampleOccurrence.ID() != (expr.ValueIdentity{}))
			if test.local {
				require.NotSame(t, source, value.Source)
				require.Equal(t, source, value.Anchor)
			}
			document := DocumentTarget(source, target, "", false, false)
			require.NoError(t, document.Error)
			require.True(t, document.Plan.Root().Valid(), "explicit documentation keeps its capture even without a root example")
		})
	}
}

func TestRuntimeTargetRejectsMissingContext(t *testing.T) {
	value := ValueTarget(&service.ValueData{}, &expr.AttributeExpr{Type: expr.String}, "", expr.ValueCodecJSON, false)
	require.ErrorContains(t, value.Error, "no effective service occurrence")
}
