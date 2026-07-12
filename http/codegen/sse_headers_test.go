package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderSSEInitHeadersDisablesProxyBuffering(t *testing.T) {
	code := renderSSEInitHeadersBody()

	require.Contains(t, code, `if header.Get("X-Accel-Buffering") == ""`)
	require.Contains(t, code, `header.Set("X-Accel-Buffering", "no")`)
}
