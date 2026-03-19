package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONExampleHandlesEmptyMaps(t *testing.T) {
	require.NotPanics(t, func() {
		require.Equal(t, "{}", jsonExample(map[int]int{}))
	})
}
