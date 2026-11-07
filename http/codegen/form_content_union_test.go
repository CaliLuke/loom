package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestFormContentUnionUsesCapturedItems(t *testing.T) {
	root := RunHTTPDSL(t, testdata.UntaggedFormDSL)
	services := CreateHTTPServices(root)
	functions := renderTransportFuncs(t, append(ServerFiles("", services), ClientFiles("", services)...))
	decoder := functions["DecodeContentRequest"]
	require.Contains(t, decoder, "loomhttp.FormTextValue")
	require.Contains(t, decoder, `loomhttp.FormJSONValue(parameterFormPairs, "value", loomhttp.FieldItems)`)
	require.Contains(t, decoder, `loomhttp.DecodeJSONField(parameterForm, "value", &v, false, loomhttp.FieldItems)`)
	require.Contains(t, decoder, `loomhttp.DecodeFormValue(parameterForm, "value", &v)`)
	require.Contains(t, decoder, `loomhttp.MatchUntaggedText("value", branches, false)`)
	require.NotContains(t, decoder, "json.Marshal")
	require.Contains(t, functions["DecodeContentOnlyRequest"], "parameterForm := r.PostForm")
	encoder := functions["EncodeContentRequest"]
	require.Contains(t, encoder, "selected[index] != matchedPath[index]")
	require.Contains(t, encoder, `contentValues["value"]`)
	require.Contains(t, functions["DecodeContentNestedRequest"], "loomhttp.ComposeTextUnion")
}
