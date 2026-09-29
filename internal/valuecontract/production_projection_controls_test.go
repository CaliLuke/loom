package valuecontract

import (
	"encoding/base64"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

type productionProjectionCustom struct {
	calls *int
}

func (value productionProjectionCustom) MarshalJSON() ([]byte, error) {
	*value.calls++
	return []byte(`"owned"`), nil
}

func checkProductionProjectionCorruption(t *testing.T, executable string) {
	t.Helper()
	cases := productionProjectionCases()
	require.NotEmpty(t, cases)
	controls := []struct {
		name    string
		fixture string
		corrupt func(*testing.T, productionProjected) productionProjected
	}{
		{"double encoding", "bytes exactly once", func(t *testing.T, result productionProjected) productionProjected {
			raw := referenceDecode[string](t, jsontext.Value(result.wire))
			result.wire = productionJSON(t, base64.StdEncoding.EncodeToString([]byte(raw)))
			return result
		}},
		{"branch substitution", "tagged retained branch", func(t *testing.T, result productionProjected) productionProjected {
			object := referenceDecode[map[string]jsontext.Value](t, jsontext.Value(result.wire))
			object["type"] = jsontext.Value(`"Text"`)
			result.wire = productionCanonical(t, jsontext.Value(productionJSON(t, object)))
			return result
		}},
		{"lost explicit presence", "presence retained empty", func(t *testing.T, result productionProjected) productionProjected {
			object := referenceDecode[map[string]jsontext.Value](t, jsontext.Value(result.wire))
			require.Contains(t, object, "explicit")
			delete(object, "explicit")
			result.wire = productionCanonical(t, jsontext.Value(productionJSON(t, object)))
			return result
		}},
		{"always omit", "bytes exactly once", func(_ *testing.T, _ productionProjected) productionProjected {
			return productionProjected{outcome: "incomplete"}
		}},
	}
	require.Len(t, controls, 4)
	for _, control := range controls {
		t.Run(control.name, func(t *testing.T) {
			found := false
			for _, tc := range cases {
				if tc.name != control.fixture {
					continue
				}
				found = true
				actual, expected := productionProjectPair(t, executable, tc, expr.ValuePlanRuntime)
				require.Equal(t, expected, actual, "unmodified production result must first pass")
				corrupted := control.corrupt(t, actual)
				require.NotEqual(t, expected, corrupted, "the exact outcome/wire comparison must reject the injected defect")
			}
			require.True(t, found, "negative control must execute its fixture")
		})
	}
}

// Custom materialization is an explicit tested boundary, not a claim that the
// builtin Lean interpreter models callback execution or arbitrary custom types.
func checkProductionProjectionExternal(t *testing.T) {
	t.Helper()
	for _, tc := range []struct {
		name    string
		typ     expr.DataType
		use     expr.ValuePlanUse
		outcome expr.ProjectionOutcome
		calls   int
	}{
		{"Any runtime owned snapshot", expr.Any, expr.ValuePlanRuntime, expr.ProjectionEmitted, 1},
		{"declared docs external snapshot", expr.String, expr.ValuePlanDocumentation, expr.ProjectionEmitted, 1},
		{"declared runtime needs external contract", expr.String, expr.ValuePlanRuntime, expr.ProjectionUnsupported, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			context := expr.NewValueContext()
			attribute := &expr.AttributeExpr{Type: tc.typ}
			occurrence, err := context.NewOccurrence(attribute)
			require.NoError(t, err)
			source := context.SupplyValue(expr.ValueInput{Raw: productionProjectionCustom{calls: &calls}})
			result := context.Resolve(occurrence, source, expr.ValueRoleExample)
			require.Equal(t, expr.ValueUnsupported, result.Outcome())
			require.Zero(t, calls)
			plan, err := context.NewValuePlan(occurrence, expr.ValuePlanRequest{Target: attribute, Codec: expr.ValueCodecJSON, Use: tc.use})
			require.NoError(t, err)
			for range 2 {
				projected := context.ProjectJSON(result, plan)
				require.Equal(t, tc.outcome, projected.Outcome())
				if wire, present := projected.JSON(); present {
					var decoded string
					require.NoError(t, json.Unmarshal(wire, &decoded))
					require.Equal(t, "owned", decoded)
				}
			}
			require.Equal(t, tc.calls, calls)
		})
	}
}
