package http

import (
	"encoding/json/jsontext"
	"errors"
	"slices"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	loom "github.com/CaliLuke/loom/pkg"
)

func TestMatchUntaggedTextTwoDomains(t *testing.T) {
	var private [2]int64
	branches := make([]TextUnionBranch, 2)
	for index, width := range []int{64, 32} {
		b := TextBranch{
			View: func() (ParamValue, error) {
				return ParamValue{Kind: ParamScalar, Present: true, Text: "2147483648"}, nil
			},
			Shape:  ParamBranchShape{Kind: ParamScalar, Element: ParamInteger},
			Member: (&loom.JSONShape{Kind: "integer"}).Match,
			Assign: func(value ParamValue) (bool, error) {
				parsed, err := strconv.ParseInt(value.Text, 10, width)
				if err == nil {
					private[index] = parsed
				}
				return err == nil, nil
			},
		}
		branches[index] = TextUnionBranch{Schema: b.Schema, Decode: b.Decode}
	}
	index, err := MatchUntaggedText("Number", branches, false)
	require.ErrorContains(t, err, "matched 2 branches in schema")
	require.Equal(t, -1, index)
	require.Equal(t, int64(2147483648), private[0])
}

func TestMatchUntaggedTextZeroMatchLeavesDestination(t *testing.T) {
	destination := "sentinel"
	branches := make([]TextUnionBranch, 3)
	for i, shape := range []*loom.JSONShape{
		{Kind: "string", Rules: loom.JSONShapeRules{Patterns: []string{"^[a-z]+$"}}},
		{Kind: "integer"}, {Kind: "array", Child: &loom.JSONShape{Kind: "integer"}},
	} {
		branches[i] = TextUnionBranch{
			Schema: func() (bool, error) {
				return shape.Match(jsontext.Value(`"?"`))
			},
			Decode: func() (bool, error) {
				return false, nil
			},
		}
	}
	index, err := MatchUntaggedText("Value", branches, false)
	if err == nil {
		destination = strconv.Itoa(index)
	}
	require.ErrorContains(t, err, "matched 0 branches in schema")
	require.Equal(t, "sentinel", destination)
}

func TestMatchUntaggedTextOrderIndependence(t *testing.T) {
	fatal := errors.New("broken codec")
	for _, broken := range []bool{false, true} {
		count := 0
		branches := []TextUnionBranch{
			{Schema: func() (bool, error) {
				count++
				return true, nil
			}, Decode: func() (bool, error) {
				return true, nil
			}},
			{Schema: func() (bool, error) {
				count++
				return false, nil
			}, Decode: func() (bool, error) {
				if broken {
					return false, fatal
				}
				return false, nil
			}},
		}
		for _, reversed := range []bool{false, true} {
			count = 0
			if reversed {
				slices.Reverse(branches)
			}
			index, err := MatchUntaggedText("Value", branches, false)
			if broken {
				require.ErrorIs(t, err, fatal)
			} else {
				require.NoError(t, err)
				require.Equal(t, 2, count)
				want := 0
				if reversed {
					want = 1
				}
				require.Equal(t, want, index)
			}
		}
	}
}

func TestMatchUntaggedTextClosedFormRoot(t *testing.T) {
	match := func() (bool, error) {
		return true, nil
	}
	miss := func() (bool, error) {
		return false, nil
	}
	var members []string
	branches := []TextUnionBranch{
		{Schema: match, Decode: match, Annotations: func() (TextUnionAnnotations, error) {
			return TextUnionAnnotations{Members: members, Evaluated: []string{"x"}}, nil
		}},
		{Schema: miss, Decode: miss},
	}
	for _, tc := range []struct {
		members []string
		want    int
	}{
		{nil, 0}, {[]string{"x"}, 0}, {[]string{"x", "extra"}, -1},
	} {
		members = tc.members
		index, err := MatchUntaggedText("Body", branches, true)
		if tc.want < 0 {
			require.ErrorContains(t, err, "unevaluated")
		} else {
			require.NoError(t, err)
		}
		require.Equal(t, tc.want, index)
	}
	branches[1].Schema = match
	branches[1].Decode = match
	_, err := MatchUntaggedText("Body", branches, true)
	require.ErrorContains(t, err, "matched 2 branches in schema")
	branches[1].Schema = miss
	branches[1].Decode = miss
	branches[0].Annotations = nil
	_, err = MatchUntaggedText("Body", branches, true)
	require.ErrorContains(t, err, "annotations")
}
