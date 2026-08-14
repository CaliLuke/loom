package openapi

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestScopedExtensionsFromExprKeepsScopesDistinct(t *testing.T) {
	meta := expr.MetaExpr{
		"openapi:extension:x-generic":                  {`{"kind":"generic"}`},
		"openapi:parameter:extension:x-parameter":      {`{"kind":"parameter"}`},
		"openapi:requestBody:extension:x-request":      {`{"kind":"request"}`},
		"openapi:response:extension:x-response":        {`{"kind":"response"}`},
		"openapi:schema:extension:x-schema":            {`{"kind":"schema"}`},
		"openapi:parameter:extension:not-an-extension": {`true`},
	}

	require.Equal(t, map[string]any{"x-generic": map[string]any{"kind": "generic"}}, ExtensionsFromExpr(meta))
	require.Equal(t, map[string]any{"x-parameter": map[string]any{"kind": "parameter"}}, ScopedExtensionsFromExpr(meta, "parameter"))
	require.Equal(t, map[string]any{"x-request": map[string]any{"kind": "request"}}, ScopedExtensionsFromExpr(meta, "requestBody"))
	require.Equal(t, map[string]any{"x-response": map[string]any{"kind": "response"}}, ScopedExtensionsFromExpr(meta, "response"))
	require.Equal(t, map[string]any{"x-schema": map[string]any{"kind": "schema"}}, ScopedExtensionsFromExpr(meta, "schema"))
}
