package http

import (
	"encoding/json/jsontext"
	"errors"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTextBranchSeam(t *testing.T) {
	for _, text := range []string{"2147483648", "abc"} {
		for _, width := range []int{32, 64} {
			t.Run(text+"/"+strconv.Itoa(width), func(t *testing.T) {
				branch := TextBranch{
					View: func() (ParamValue, error) {
						return ParamValue{Kind: ParamScalar, Present: true, Text: text}, nil
					},
					Shape: ParamBranchShape{Kind: ParamScalar, Element: ParamInteger},
					Member: func(value jsontext.Value) (bool, error) {
						return value.Kind() == '0', nil
					},
					Assign: func(value ParamValue) (bool, error) {
						_, err := strconv.ParseInt(value.Text, 10, width)
						return err == nil, nil
					},
				}
				member, err := branch.Schema()
				require.NoError(t, err)
				require.Equal(t, text != "abc", member)
				decoded, err := branch.Decode()
				require.NoError(t, err)
				require.Equal(t, text != "abc" && width == 64, decoded)
			})
		}
	}
	fatal := errors.New("invalid schema configuration")
	branch := TextBranch{
		View: func() (ParamValue, error) {
			return ParamValue{Kind: ParamScalar, Present: true, Text: "1"}, nil
		},
		Shape: ParamBranchShape{Kind: ParamScalar, Element: ParamInteger},
		Member: func(jsontext.Value) (bool, error) {
			return false, fatal
		},
	}
	_, err := branch.Schema()
	require.ErrorIs(t, err, fatal)
	for _, tc := range []struct {
		text     string
		kind     ParamScalarKind
		accepted bool
	}{
		{"9007199254740993", ParamInteger, true}, {"1e1000", ParamNumber, true},
		{"1.0", ParamInteger, true}, {"1.5", ParamInteger, false},
		{"01", ParamNumber, false}, {"NaN", ParamNumber, false},
		{"true", ParamBoolean, true}, {"1", ParamBoolean, false},
	} {
		raw, ok, err := ParamValueJSON(ParamValue{Kind: ParamScalar, Present: true, Text: tc.text}, ParamBranchShape{Kind: ParamScalar, Element: tc.kind})
		require.NoError(t, err)
		require.Equal(t, tc.accepted, ok, tc.text)
		if ok {
			require.Equal(t, tc.text, string(raw))
		}
	}
}

func TestTextBranchInvalidUTF8IsMismatch(t *testing.T) {
	for _, tc := range []struct {
		name  string
		raw   string
		codec ParamCodec
		shape ParamBranchShape
	}{
		{"string", "n=%FF", ParamCodec{Style: ParamForm}, ParamBranchShape{Kind: ParamScalar, Element: ParamString}},
		{"bytes", "n=%FF", ParamCodec{Style: ParamForm}, ParamBranchShape{Kind: ParamScalar, Element: ParamBytes}},
		{"list", "n=%FF", ParamCodec{Style: ParamForm, Explode: true}, ParamBranchShape{Kind: ParamList, Element: ParamString}},
		{"map value", "key=%FF", ParamCodec{Style: ParamForm, Explode: true}, ParamBranchShape{Kind: ParamPairs, MapValue: ParamString}},
		{"map key", "%FF=x", ParamCodec{Style: ParamForm, Explode: true}, ParamBranchShape{Kind: ParamPairs, MapValue: ParamString}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pairs, err := ParseRawPairs(tc.raw)
			require.NoError(t, err)
			called := false
			branch := TextBranch{
				View: func() (ParamValue, error) {
					return DecodePairsParam(pairs, "n", tc.codec, tc.shape.Kind, ParamNamespace{})
				},
				Shape: tc.shape,
				Member: func(jsontext.Value) (bool, error) {
					called = true
					return true, nil
				},
			}
			member, err := branch.Schema()
			require.NoError(t, err)
			require.False(t, member)
			require.False(t, called)
		})
	}
}
