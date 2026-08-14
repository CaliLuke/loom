package openapiimport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiagnosticsClassify(t *testing.T) {
	diagnostics := Diagnostics{
		{Code: "info-metadata", Path: "#/info", Message: "metadata"},
		{Code: "examples", Path: "#/paths/~1pets", Message: "examples"},
		{Code: "servers", Path: "#/servers", Message: "servers"},
		{Code: "future-diagnostic", Path: "#/future", Message: "future"},
	}

	t.Run("strict", func(t *testing.T) {
		fatal, warnings := diagnostics.Classify(false)
		require.Equal(t, diagnostics, fatal)
		require.Empty(t, warnings)
	})

	t.Run("allow lossy", func(t *testing.T) {
		fatal, warnings := diagnostics.Classify(true)
		require.Equal(t, Diagnostics{
			{Code: "servers", Path: "#/servers", Message: "servers"},
			{Code: "future-diagnostic", Path: "#/future", Message: "future"},
		}, fatal)
		require.Equal(t, Diagnostics{
			{Code: "info-metadata", Path: "#/info", Message: "metadata"},
			{Code: "examples", Path: "#/paths/~1pets", Message: "examples"},
		}, warnings)
	})
}
