package expr

import (
	"context"
	"math"
	"os"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/testprocess"
)

type traversalCodecByte uint8

func (traversalCodecByte) MarshalJSON() ([]byte, error) {
	return []byte(`"custom"`), nil
}

func TestValueSnapshotRetainsNonReflexiveEntries(t *testing.T) {
	raw := map[float64][]string{}
	raw[math.NaN()] = []string{"one"}
	raw[math.NaN()] = []string{"two"}
	snapshot := snapshotValueSource(raw)
	require.NoError(t, snapshot.err)
	owned := snapshot.raw.(map[float64][]string)
	require.Len(t, owned, 2)
	values := make([]string, 0, len(owned))
	for key, value := range owned {
		require.True(t, math.IsNaN(key))
		values = append(values, value[0])
	}
	slices.Sort(values)
	require.Equal(t, []string{"one", "two"}, values)
	for _, value := range raw {
		value[0] = "changed"
	}
	for _, value := range owned {
		require.NotEqual(t, "changed", value[0])
	}
	for _, typ := range []DataType{Any, &Map{KeyType: &AttributeExpr{Type: Any}, ElemType: &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: String}}}}} {
		c := NewValueContext()
		occurrence, err := c.NewOccurrence(&AttributeExpr{Type: typ})
		require.NoError(t, err)
		result := c.Resolve(occurrence, c.SupplyValue(ValueInput{Raw: raw}), ValueRoleExample)
		require.Equal(t, ValueInvalid, result.Outcome())
		legacy, ok := result.LegacyValue()
		require.True(t, ok)
		require.Len(t, legacy, 2)
	}
}

func TestValueRawByteChildrenUseSharedBoundary(t *testing.T) {
	for _, raw := range []any{[]traversalCodecByte{1}, [1]traversalCodecByte{1}, []any{traversalCodecByte(1)}} {
		c := NewValueContext()
		occurrence, err := c.NewOccurrence(&AttributeExpr{Type: Any})
		require.NoError(t, err)
		result := c.Resolve(occurrence, c.SupplyValue(ValueInput{Raw: raw}), ValueRoleExample)
		require.Equal(t, ValueUnsupported, result.Outcome(), "%T", raw)
	}
	// Native byte arrays/slices keep their codec shape; defined builtin element
	// types without codec methods remain finite Any arrays, not custom values.
	for _, raw := range []any{[]byte{1}, [1]byte{1}, []eligibilityByte{1}, [1]eligibilityByte{1}} {
		c := NewValueContext()
		occurrence, err := c.NewOccurrence(&AttributeExpr{Type: Any})
		require.NoError(t, err)
		result := c.Resolve(occurrence, c.SupplyValue(ValueInput{Raw: raw}), ValueRoleExample)
		require.Equal(t, ValueResolved, result.Outcome(), "%T", raw)
		value, ok := result.Value()
		require.True(t, ok)
		owned, ok := value.RawAny()
		require.True(t, ok)
		require.Equal(t, raw, owned)
	}
}

func TestValueSynthesisCycleTermination(t *testing.T) {
	const child = "LOOM_VALUE_SYNTHESIS_CYCLE_CHILD"
	if os.Getenv(child) == "1" {
		checkValueSynthesisCycles(t)
		return
	}
	executable, err := os.Executable()
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	command := testprocess.CommandContext(ctx, executable, "-test.run=^TestValueSynthesisCycleTermination$", "-test.count=1")
	command.Env = append(os.Environ(), child+"=1")
	output, err := command.CombinedOutput()
	require.NoError(t, err, "synthesis must terminate after a rejected cycle: %s", output)
}

func checkValueSynthesisCycles(t *testing.T) {
	t.Helper()
	mapping := map[string]any{}
	mapping["self"] = mapping
	sequence := make([]any, 1)
	sequence[0] = sequence
	for _, cycle := range []any{mapping, sequence} {
		one := 1
		child := &AttributeExpr{Type: Any, UserExamples: []*ExampleExpr{{Value: cycle}}}
		attribute := &AttributeExpr{Type: &Array{ElemType: child}, Validation: &ValidationExpr{MinLength: &one, MaxLength: &one}}
		c := NewValueContext()
		occurrence, err := c.NewOccurrence(attribute)
		require.NoError(t, err)
		selection := c.SelectExample(occurrence, ExamplePolicy{Reachable: true})
		result := c.Synthesize(selection, NewRandom("cycle"))
		require.Equal(t, ValueInvalid, result.Outcome())
		require.Equal(t, "cyclic", result.Diagnostics()[0].Code)
		raw, ok := result.LegacyValue()
		require.True(t, ok)
		require.Len(t, raw, 1)
		owned := reflect.ValueOf(raw).Index(0).Interface()
		require.NotEqual(t, reflect.ValueOf(cycle).Pointer(), reflect.ValueOf(owned).Pointer())
		require.ErrorContains(t, snapshotValueSource(owned).err, "cyclic")
		again := c.Synthesize(selection, NewRandom("other"))
		require.Equal(t, result.SourceID(), again.SourceID())
	}
}
