package expr

import (
	"encoding/json/jsontext"
	"testing"

	"github.com/stretchr/testify/require"
)

func boundForTest(value any) *NumericBound {
	bound, err := NewNumericBound(value)
	if err != nil {
		panic(err)
	}
	return &bound
}

func matchSchemaForTest(t *testing.T, plan *valuePlanNode, wire jsontext.Value) bool {
	t.Helper()
	matched, err := schemaJSON(plan, wire)
	require.NoError(t, err)
	return matched
}
