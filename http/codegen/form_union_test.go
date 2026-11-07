package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestFormUnionGeneratedDomains(t *testing.T) {
	root := RunHTTPDSL(t, testdata.UntaggedFormDSL)
	services := CreateHTTPServices(root)
	functions := renderTransportFuncs(t, append(ServerFiles("", services), ClientFiles("", services)...))
	required := functions["DecodeRootRequiredRequest"]
	require.Contains(t, required, "loomhttp.FormObjectJSON(parameterFormPairs")
	require.Contains(t, required, "Schema: func() (bool, error)")
	require.Contains(t, required, "Decode: func() (bool, error)")
	require.Contains(t, required, `projection0.FieldClaims["count"]`)
	require.NotContains(t, required, "len(r.PostForm) == 0")
	require.NotContains(t, required, "parameterFormPairs.HasInput()")
	require.Contains(t, functions["DecodeRootOptionalRequest"], "parameterFormPairs.HasInput()")
	require.Contains(t, functions["DecodeRootMapRequest"], "loomhttp.FormMapJSON(parameterFormPairs")
	require.Contains(t, functions["DecodeRootNestedRequest"], "loomhttp.ComposeTextUnion")
	require.Contains(t, functions["EncodeRootNestedRequest"], "selected = append(selected")
	require.Contains(t, functions["EncodeRootNestedRequest"], "selected[index] != matchedPath[index]")
	require.Contains(t, functions["DecodeRootClosedRequest"], `loomhttp.MatchUntaggedText("form body", branches, true)`)
}
