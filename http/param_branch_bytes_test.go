package http

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParamBytesSchemaProjection(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value ParamValue
		shape ParamBranchShape
		want  string
	}{
		{"scalar", ParamValue{Kind: ParamScalar, Present: true, Text: "é"}, ParamBranchShape{Kind: ParamScalar, Element: ParamBytes}, `"w6k="`},
		{"binary scalar", ParamValue{Kind: ParamScalar, Present: true, Text: string([]byte{0xff, 0})}, ParamBranchShape{Kind: ParamScalar, Element: ParamBytes}, `"/wA="`},
		{"binary list", ParamValue{Kind: ParamList, Present: true, Items: []string{string([]byte{0xff, 0})}}, ParamBranchShape{Kind: ParamList, Element: ParamBytes}, `["/wA="]`},
		{"list", ParamValue{Kind: ParamList, Present: true, Items: []string{"hello", ""}}, ParamBranchShape{Kind: ParamList, Element: ParamBytes}, `["aGVsbG8=",""]`},
		{"object", ParamValue{Kind: ParamPairs, Present: true, Pairs: []ParamPair{{Key: "data", Value: "hello"}}}, ParamBranchShape{Kind: ParamPairs, Properties: map[string]ParamScalarKind{"data": ParamBytes}}, `{"data":"aGVsbG8="}`},
		{"map", ParamValue{Kind: ParamPairs, Present: true, Pairs: []ParamPair{{Key: "data", Value: "hello"}}}, ParamBranchShape{Kind: ParamPairs, MapValue: ParamBytes}, `{"data":"aGVsbG8="}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wire, matched, err := ParamValueJSON(tc.value, tc.shape)
			require.NoError(t, err)
			require.True(t, matched)
			require.JSONEq(t, tc.want, string(wire))
		})
	}
}

func TestParamStringRejectsInvalidUTF8(t *testing.T) {
	_, matched, err := ParamValueJSON(ParamValue{Kind: ParamScalar, Present: true, Text: string([]byte{0xff})}, ParamBranchShape{Kind: ParamScalar, Element: ParamString})
	require.NoError(t, err)
	require.False(t, matched)
}
