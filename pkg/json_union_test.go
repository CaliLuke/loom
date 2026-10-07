package loom

import (
	"encoding/json/jsontext"
	"errors"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

type failingJSONCandidate struct{}

var errJSONCandidate = errors.New("unexpected application codec")

func (*failingJSONCandidate) UnmarshalJSON([]byte) error {
	return errJSONCandidate
}

func TestJSONCandidatePropagatesUnexpectedError(t *testing.T) {
	var value failingJSONCandidate
	matched, err := DecodeJSONCandidate([]byte(`{}`), &value)
	require.False(t, matched)
	require.ErrorIs(t, err, errJSONCandidate)
}

func TestMatchUntaggedJSON(t *testing.T) {
	integer := &JSONShape{Kind: "integer"}
	array := &JSONShape{Kind: "array", Child: integer}
	page := &JSONShape{Kind: "object", Closed: true, Fields: []JSONShapeField{{Name: "total", Required: true, Shape: integer}}}
	for _, tc := range []struct {
		name, wire string
		want       int
	}{
		{"empty array", `[]`, 0}, {"array", `[1,2]`, 0}, {"page", `{"total":0}`, 1},
		{"wrong element", `["wrong"]`, -1}, {"fraction", `[1.5]`, -1},
		{"required", `{}`, -1}, {"case sensitive", `{"Total":0}`, -1},
		{"closed", `{"total":0,"extra":1}`, -1}, {"null element", `[null]`, -1},
		{"null root", `null`, -1}, {"malformed", `[`, -1}, {"trailing", `[] {}`, -1},
		{"duplicate", `{"total":0,"total":1}`, -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, reversed := range []bool{false, true} {
				var items []int
				var object struct {
					Total int `json:"total"`
				}
				branches := []JSONUnionBranch{
					{Schema: array, Decode: func(wire jsontext.Value) (bool, error) {
						return DecodeJSONCandidate(wire, &items)
					}},
					{Schema: page, Decode: func(wire jsontext.Value) (bool, error) {
						return DecodeJSONCandidate(wire, &object)
					}},
				}
				if reversed {
					slices.Reverse(branches)
				}
				got, err := MatchUntaggedJSON("Result", []byte(tc.wire), branches)
				if tc.want < 0 {
					require.Error(t, err)
					require.Equal(t, -1, got)
					continue
				}
				require.NoError(t, err)
				want := tc.want
				if reversed {
					want = 1 - want
				}
				require.Equal(t, want, got)
			}
		})
	}
}

func TestUntaggedSchemaAndDecoderIndependence(t *testing.T) {
	array := &JSONShape{Kind: "array", Child: &JSONShape{Kind: "integer"}}
	var signed []int32
	var unsigned []uint32
	branches := []JSONUnionBranch{
		{Schema: array, Decode: func(wire jsontext.Value) (bool, error) {
			return DecodeJSONCandidate(wire, &signed)
		}},
		{Schema: array, Decode: func(wire jsontext.Value) (bool, error) {
			return DecodeJSONCandidate(wire, &unsigned)
		}},
	}
	for _, wire := range []string{`[-1]`, `[]`, `[1]`} {
		_, err := MatchUntaggedJSON("Numbers", []byte(wire), branches)
		require.ErrorContains(t, err, "matched 2 branches in schema")
	}
	branches[1].Schema = &JSONShape{Kind: "object"}
	branches[0].Decode = func(jsontext.Value) (bool, error) {
		return false, nil
	}
	branches[1].Decode = func(jsontext.Value) (bool, error) {
		return true, nil
	}
	_, err := MatchUntaggedJSON("Numbers", []byte(`[-1]`), branches)
	require.ErrorContains(t, err, "identities differ")
	failure := errors.New("broken predicate")
	branches[1].Decode = func(jsontext.Value) (bool, error) {
		return false, failure
	}
	branches[0].Decode = func(jsontext.Value) (bool, error) {
		return true, nil
	}
	_, err = MatchUntaggedJSON("Numbers", []byte(`[-1]`), branches)
	require.ErrorIs(t, err, failure)
	slices.Reverse(branches)
	_, err = MatchUntaggedJSON("Numbers", []byte(`[-1]`), branches)
	require.ErrorIs(t, err, failure)
}

func TestJSONShapePredicates(t *testing.T) {
	for _, tc := range []struct {
		name           string
		shape          *JSONShape
		valid, invalid string
	}{
		{"unicode length", &JSONShape{Kind: "string", Rules: JSONShapeRules{MinLength: new(2), MaxLength: new(2)}}, `"éa"`, `"é"`},
		{"decoded byte length", &JSONShape{Kind: "bytes", Rules: JSONShapeRules{MinLength: new(2), MaxLength: new(2)}}, `"aGk="`, `"YQ=="`},
		{"numeric precision", &JSONShape{Kind: "integer", Rules: JSONShapeRules{Minimum: "9007199254740993"}}, `9007199254740993`, `9007199254740992`},
		{"exclusive bound", &JSONShape{Kind: "number", Rules: JSONShapeRules{ExclusiveMinimum: "1"}}, `1.1`, `1`},
		{"conjoined enums", &JSONShape{Kind: "integer", Rules: JSONShapeRules{Enums: [][]jsontext.Value{{[]byte(`1`), []byte(`2`)}, {[]byte(`2`)}}}}, `2.0`, `1`},
		{"nullable member", &JSONShape{Kind: "string", Nullable: true}, `null`, `3`},
		{"non-null elements", &JSONShape{Kind: "array", Child: &JSONShape{Kind: "string", Nullable: true}, NonNullableElements: true}, `["x"]`, `[null]`},
		{"format", &JSONShape{Kind: "string", Rules: JSONShapeRules{Formats: []Format{FormatEmail}}}, `"user@example.com"`, `"invalid"`},
		{"pattern", &JSONShape{Kind: "string", Rules: JSONShapeRules{Patterns: []string{"^a", "z$"}}}, `"az"`, `"ab"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			matched, err := tc.shape.Match([]byte(tc.valid))
			require.NoError(t, err)
			require.True(t, matched)
			matched, err = tc.shape.Match([]byte(tc.invalid))
			require.NoError(t, err)
			require.False(t, matched)
		})
	}
	for _, shape := range []*JSONShape{
		{Kind: "array"}, {Kind: "string", Rules: JSONShapeRules{Patterns: []string{"["}}},
		{Kind: "string", Rules: JSONShapeRules{Formats: []Format{"unknown"}}},
		{Kind: "integer", Rules: JSONShapeRules{Minimum: "not a number"}},
	} {
		_, err := shape.Match([]byte(`[]`))
		require.Error(t, err, "invalid configuration must fail even when wire does not match")
	}
}

func TestJSONShapeNumericCapacityFailureIsFatal(t *testing.T) {
	for _, kind := range []string{"integer", "number"} {
		shape := &JSONShape{Kind: kind}
		_, err := shape.Match([]byte("1e1000001"))
		require.ErrorContains(t, err, "exact matching capacity")
	}
}
