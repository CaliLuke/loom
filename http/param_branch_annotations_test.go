package http

import (
	"testing"

	"github.com/stretchr/testify/require"

	loom "github.com/CaliLuke/loom/pkg"
)

func TestTextBranchAnnotationsUseCodecOwnedView(t *testing.T) {
	for _, tc := range []struct {
		name  string
		codec ParamCodec
		raw   string
		owned []string
		want  []string
	}{
		{"fixed object", ParamCodec{Style: ParamForm, Explode: true}, "x=value&control=ignored&outside=ignored", []string{"x"}, []string{"x"}},
		{"dynamic map", ParamCodec{Style: ParamForm, Explode: true}, "x=value&control=ignored&outside=data", nil, []string{"x", "outside"}},
		{"deep object", ParamCodec{Style: ParamDeepObject, Explode: true}, "value[x]=value&control=ignored&other[x]=ignored", nil, []string{"x"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pairs, err := ParseRawPairs(tc.raw)
			require.NoError(t, err)
			shape := &loom.JSONShape{Kind: "map", Child: &loom.JSONShape{Kind: "string"}}
			branch := TextBranch{
				View: func() (ParamValue, error) {
					return DecodePairsParam(pairs, "value", tc.codec, ParamPairs, ParamNamespace{Literals: []string{"control"}, OwnedLiterals: tc.owned})
				},
				Shape:    ParamBranchShape{Kind: ParamPairs, MapValue: ParamString},
				Member:   shape.Match,
				Evaluate: shape.Evaluated,
			}
			matches, err := branch.Schema()
			require.NoError(t, err)
			require.True(t, matches)
			annotations, err := branch.Annotations()
			require.NoError(t, err)
			require.ElementsMatch(t, tc.want, annotations.Members)
			require.ElementsMatch(t, tc.want, annotations.Evaluated)
		})
	}
}

func TestTextBranchClosureUsesSchemaAnnotations(t *testing.T) {
	shape := &loom.JSONShape{Kind: "object", Fields: []loom.JSONShapeField{
		{Name: "x", Required: true, Shape: &loom.JSONShape{Kind: "string"}},
	}}
	branch := TextBranch{
		View: func() (ParamValue, error) {
			return ParamValue{Kind: ParamPairs, Present: true, Pairs: []ParamPair{{Key: "x", Value: "value"}, {Key: "extra", Value: "data"}}}, nil
		},
		Shape:    ParamBranchShape{Kind: ParamPairs, MapValue: ParamString},
		Member:   shape.Match,
		Evaluate: shape.Evaluated,
		Assign: func(ParamValue) (bool, error) {
			return true, nil
		},
	}
	branches := []TextUnionBranch{
		{Schema: branch.Schema, Decode: branch.Decode, Annotations: branch.Annotations},
		{
			Schema: func() (bool, error) {
				return false, nil
			},
			Decode: func() (bool, error) {
				return false, nil
			},
		},
	}
	index, err := MatchUntaggedText("body", branches, false)
	require.NoError(t, err)
	require.Equal(t, 0, index)
	index, err = MatchUntaggedText("body", branches, true)
	require.ErrorContains(t, err, "unevaluated member")
	require.Equal(t, -1, index)
}
