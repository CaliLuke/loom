package loom

import (
	"encoding/json/jsontext"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

type nestedJSONCandidate struct{}

func (*nestedJSONCandidate) UnmarshalJSON(data []byte) error {
	_, err := MatchUntaggedJSON("Inner", data, []JSONUnionBranch{
		{Schema: &JSONShape{Kind: "integer"}, Decode: func(jsontext.Value) (bool, error) {
			return false, nil
		}},
		{Schema: &JSONShape{Kind: "string"}, Decode: func(jsontext.Value) (bool, error) {
			return false, nil
		}},
	})
	return err
}

func TestJSONCandidateNestedMismatch(t *testing.T) {
	for _, wire := range []string{`{"value":1}`, `{"value":"text"}`, `{"value":{}}`} {
		var value struct {
			Value nestedJSONCandidate `json:"value"`
		}
		matched, err := DecodeJSONCandidate([]byte(wire), &value)
		require.NoError(t, err, wire)
		require.False(t, matched, wire)
	}
}

func TestJSONCandidateNestedFailureClassification(t *testing.T) {
	var unexpected struct {
		Value failingJSONCandidate `json:"value"`
	}
	matched, err := DecodeJSONCandidate([]byte(`{"value":{}}`), &unexpected)
	require.False(t, matched)
	require.ErrorIs(t, err, errJSONCandidate)

	for _, reversed := range []bool{false, true} {
		var inner struct {
			Value nestedJSONCandidate `json:"value"`
		}
		var other map[string]int
		branches := []JSONUnionBranch{
			{Schema: &JSONShape{Kind: "string"}, Decode: func(wire jsontext.Value) (bool, error) {
				return DecodeJSONCandidate(wire, &inner)
			}},
			{Schema: &JSONShape{Kind: "map", Child: &JSONShape{Kind: "integer"}}, Decode: func(wire jsontext.Value) (bool, error) {
				return DecodeJSONCandidate(wire, &other)
			}},
		}
		want := 1
		if reversed {
			slices.Reverse(branches)
			want = 0
		}
		selected, err := MatchUntaggedJSON("Outer", []byte(`{"value":1}`), branches)
		require.NoError(t, err)
		require.Equal(t, want, selected)
	}
}
