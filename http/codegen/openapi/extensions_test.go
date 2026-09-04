package openapi

import (
	"encoding/json/jsontext"
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
		"openapi:extension:x-exact":                    {`{"z":9007199254740993,"a":1}`},
	}

	extensions := ExtensionsFromExpr(meta)
	require.Equal(t, jsontext.Value(`{"kind":"generic"}`), extensions["x-generic"])
	require.Equal(t, jsontext.Value(`{"a":1,"z":9007199254740993}`), extensions["x-exact"])
	require.Equal(t, jsontext.Value(`{"kind":"parameter"}`), ScopedExtensionsFromExpr(meta, "parameter")["x-parameter"])
	require.Equal(t, jsontext.Value(`{"kind":"request"}`), ScopedExtensionsFromExpr(meta, "requestBody")["x-request"])
	require.Equal(t, jsontext.Value(`{"kind":"response"}`), ScopedExtensionsFromExpr(meta, "response")["x-response"])
	require.Equal(t, jsontext.Value(`{"kind":"schema"}`), ScopedExtensionsFromExpr(meta, "schema")["x-schema"])
}
