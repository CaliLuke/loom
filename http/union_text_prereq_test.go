package http

import (
	"encoding/json/jsontext"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/httpencoding"
)

func TestTextUnionCodecPrerequisite(t *testing.T) {
	explode := false
	resolved, keys, err := httpencoding.Resolve(httpencoding.NamedQuery, "id", httpencoding.BranchShape{Shape: httpencoding.Union, Branches: []httpencoding.BranchShape{
		{Shape: httpencoding.Scalar}, {Shape: httpencoding.Sequence},
	}}, httpencoding.Serialization{Style: "form", Explode: &explode})
	require.NoError(t, err)
	require.Equal(t, httpencoding.KeySet{Literals: []string{"id"}}, keys)
	codecs := resolved.Branches
	require.Len(t, codecs, 2)
	require.Equal(t, httpencoding.Codec{Style: httpencoding.Form, Explode: true, Shape: httpencoding.Scalar}, codecs[0].Codec)
	require.Equal(t, httpencoding.Codec{Style: httpencoding.Form, Shape: httpencoding.Sequence}, codecs[1].Codec)

	_, keys, err = httpencoding.Resolve(httpencoding.NamedQuery, "id", httpencoding.BranchShape{Shape: httpencoding.Union, Branches: []httpencoding.BranchShape{
		{Shape: httpencoding.ScalarMap}, {Shape: httpencoding.ScalarMap},
	}}, httpencoding.Serialization{Style: "form"})
	require.NoError(t, err)
	require.Equal(t, httpencoding.KeySet{Residual: true}, keys)
	require.Error(t, httpencoding.CheckNamespaces(httpencoding.NamedQuery, []httpencoding.Binding{
		{Name: "union", Keys: keys}, {Name: "other", Keys: httpencoding.KeySet{Residual: true}},
	}))
	_, _, err = httpencoding.Resolve(httpencoding.NamedQuery, "id", httpencoding.BranchShape{Shape: httpencoding.Union, Branches: []httpencoding.BranchShape{
		{Shape: httpencoding.Scalar}, {Shape: httpencoding.Object},
	}}, httpencoding.Serialization{Style: "deepObject"})
	require.ErrorContains(t, err, "branch 0")

	for _, kind := range []ParamScalarKind{ParamString, ParamInteger} {
		branch := TextBranch{
			View: func() (ParamValue, error) {
				return ParamValue{Kind: ParamScalar, Present: true, Text: "42"}, nil
			},
			Shape: ParamBranchShape{Kind: ParamScalar, Element: kind},
			Member: func(value jsontext.Value) (bool, error) {
				return value.Kind() == '"' && kind == ParamString || value.Kind() == '0' && kind == ParamInteger, nil
			},
		}
		member, err := branch.Schema()
		require.NoError(t, err)
		require.True(t, member)
	}
	calls := 0
	for _, width := range []int{32, 64} {
		branch := TextBranch{
			View: func() (ParamValue, error) {
				return ParamValue{Kind: ParamScalar, Present: true, Text: "2147483648"}, nil
			},
			Assign: func(value ParamValue) (bool, error) {
				calls++
				_, err := strconv.ParseInt(value.Text, 10, width)
				return err == nil, nil
			},
		}
		decoded, err := branch.Decode()
		require.NoError(t, err)
		require.Equal(t, width == 64, decoded)
	}
	require.Equal(t, 2, calls)

	pairs, err := ParseRawPairs("n=a%2Cb,c")
	require.NoError(t, err)
	value, err := DecodePairsParam(pairs, "n", ParamCodec{Style: ParamForm}, ParamList, ParamNamespace{})
	require.NoError(t, err)
	require.True(t, value.Present)
	require.Equal(t, []string{"a,b", "c"}, value.Items)
}
