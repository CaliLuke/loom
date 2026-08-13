package openapiv3

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToJSONUsesV2Semantics(t *testing.T) {
	value := map[string]any{
		"nil_map":   map[string]string(nil),
		"nil_slice": []string(nil),
	}

	got, err := toJSON(nil, value)

	require.NoError(t, err)
	require.Equal(t, `{"nil_map":{},"nil_slice":[]}`, got)
}
