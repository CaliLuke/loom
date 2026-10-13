package ir

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchemaComponentPercentEncoding(t *testing.T) {
	for _, ref := range []string{"#/components/schemas/%49temName", "#%2Fcomponents%2Fschemas%2FItemName"} {
		name, ok := schemaComponentName(ref)
		require.True(t, ok)
		require.Equal(t, "ItemName", name)
	}
}
