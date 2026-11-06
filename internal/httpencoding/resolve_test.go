package httpencoding

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveCells(t *testing.T) {
	styles := []struct {
		name      string
		style     Style
		locations map[Location]bool
	}{
		{"simple", Simple, map[Location]bool{Path: true, Header: true}},
		{"matrix", Matrix, map[Location]bool{Path: true}},
		{"label", Label, map[Location]bool{Path: true}},
		{"form", Form, map[Location]bool{NamedQuery: true, FormField: true, Cookie: true}},
		{"spaceDelimited", SpaceDelimited, map[Location]bool{NamedQuery: true, FormField: true}},
		{"pipeDelimited", PipeDelimited, map[Location]bool{NamedQuery: true, FormField: true}},
		{"deepObject", DeepObject, map[Location]bool{NamedQuery: true, FormField: true}},
		{"cookie", CookieStyle, map[Location]bool{Cookie: true}},
	}
	for _, location := range []Location{Path, NamedQuery, Header, Cookie, FormField} {
		for _, style := range styles {
			for _, explode := range []bool{false, true} {
				for _, shape := range []Shape{Scalar, Sequence, Object, ScalarMap, Nested, UnsupportedShape} {
					t.Run(fmt.Sprintf("%d/%s/%t/%d", location, style.name, explode, shape), func(t *testing.T) {
						value := BranchShape{Shape: shape}
						if shape == Object {
							value.Properties = []PropertyFact{{WireName: "x", Kind: String}}
						}
						codec, keys, err := Resolve(location, "n", value, Serialization{Style: style.name, Explode: &explode})
						accepted := style.locations[location] && shape != Nested && shape != UnsupportedShape
						if style.style == DeepObject {
							accepted = accepted && (shape == Object || shape == ScalarMap)
						}
						if style.style == SpaceDelimited || style.style == PipeDelimited {
							accepted = accepted && !explode && shape != Scalar
						}
						if location == Cookie && style.style == Form {
							accepted = accepted && (shape == Scalar || !explode)
						}
						if !accepted {
							require.Error(t, err)
							return
						}
						require.NoError(t, err)
						require.Equal(t, style.style, codec.Style)
						require.Equal(t, shape, codec.Shape)
						wantExplode := explode
						if shape == Scalar {
							wantExplode = style.style == Form || style.style == CookieStyle
						}
						if style.style == DeepObject {
							wantExplode = false
						}
						require.Equal(t, wantExplode, codec.Explode)
						require.True(t, len(keys.Literals)+len(keys.Prefixes) > 0 || keys.Residual)
					})
				}
			}
		}
	}
}

func TestResolveLocationDefaults(t *testing.T) {
	yes, no := true, false
	for _, tc := range []struct {
		name   string
		loc    Location
		shape  Shape
		source Serialization
		want   Codec
		err    string
	}{
		{"query map JSON", NamedQuery, ScalarMap, Serialization{}, Codec{Style: JSON, Shape: ScalarMap}, ""},
		{"query map reserved true", NamedQuery, ScalarMap, Serialization{AllowReserved: &yes}, Codec{}, "content-based parameters"},
		{"query map reserved false", NamedQuery, ScalarMap, Serialization{AllowReserved: &no}, Codec{}, "content-based parameters"},
		{"query explode opts in", NamedQuery, ScalarMap, Serialization{Explode: &yes}, Codec{Style: Form, Explode: true, Shape: ScalarMap}, ""},
		{"query content style", NamedQuery, ScalarMap, Serialization{MediaType: "application/json", Style: "form"}, Codec{}, "content-based parameters"},
		{"query content explode", NamedQuery, ScalarMap, Serialization{MediaType: "application/json", Explode: &no}, Codec{}, "content-based parameters"},
		{"query content reserved", NamedQuery, ScalarMap, Serialization{MediaType: "application/json", AllowReserved: &no}, Codec{}, "content-based parameters"},
		{"form object JSON", FormField, Object, Serialization{}, Codec{Style: JSON, Shape: Object}, ""},
		{"form content overridden", FormField, Object, Serialization{MediaType: "application/json", Explode: &no}, Codec{Style: Form, Shape: Object}, ""},
		{"form explicit false", FormField, Object, Serialization{AllowReserved: &no}, Codec{Style: Form, Explode: true, Shape: Object}, ""},
		{"form JSON scalar", FormField, Scalar, Serialization{MediaType: "application/json"}, Codec{}, "media type"},
		{"form typeless default", FormField, Union, Serialization{}, Codec{}, "application/octet-stream"},
		{"path content", Path, Scalar, Serialization{MediaType: "text/plain"}, Codec{}, "query-only"},
		{"header content", Header, Scalar, Serialization{MediaType: "application/json"}, Codec{}, "query-only"},
		{"cookie content", Cookie, Scalar, Serialization{MediaType: "application/json"}, Codec{}, "query-only"},
		{"path default", Path, Sequence, Serialization{}, Codec{Style: Simple, Shape: Sequence}, ""},
		{"header default", Header, Object, Serialization{}, Codec{Style: Simple, Shape: Object}, ""},
		{"cookie default", Cookie, Sequence, Serialization{}, Codec{Style: CookieStyle, Explode: true, Shape: Sequence}, ""},
		{"unknown style", NamedQuery, Scalar, Serialization{Style: "mystery"}, Codec{}, "style"},
		{"deep scalar", NamedQuery, Scalar, Serialization{Style: "deepObject"}, Codec{}, "deepObject is defined only for objects"},
		{"nested style", NamedQuery, Nested, Serialization{Style: "form"}, Codec{}, "nested values have no style serialization"},
		{"space exploded", NamedQuery, Sequence, Serialization{Style: "spaceDelimited", Explode: &yes}, Codec{}, "n/a for explode: true"},
		{"pipe scalar", NamedQuery, Scalar, Serialization{Style: "pipeDelimited"}, Codec{}, "n/a for primitives"},
		{"cookie form exploded", Cookie, Sequence, Serialization{Style: "form"}, Codec{}, "wrong cookie delimiter"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, _, err := Resolve(tc.loc, "n", BranchShape{Shape: tc.shape}, tc.source)
			if tc.err != "" {
				require.ErrorContains(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestResolveEffectiveStyleOnce(t *testing.T) {
	branches, keys, err := ResolveUnion(NamedQuery, "n", []BranchShape{{Shape: Scalar}, {Shape: ScalarMap}}, Serialization{})
	require.NoError(t, err)
	require.Equal(t, Form, branches[0].Codec.Style)
	require.Equal(t, Form, branches[1].Codec.Style)
	require.Equal(t, KeySet{Literals: []string{"n"}, Residual: true}, keys)
}

func TestResolveUnion(t *testing.T) {
	t.Run("invalid branch", func(t *testing.T) {
		_, _, err := ResolveUnion(NamedQuery, "n", []BranchShape{{Shape: Scalar}, {Shape: Object}}, Serialization{Style: "deepObject"})
		require.ErrorContains(t, err, "branch 0")
	})
	t.Run("one residual claim", func(t *testing.T) {
		branches, keys, err := ResolveUnion(NamedQuery, "n", []BranchShape{{Shape: ScalarMap}, {Shape: ScalarMap}}, Serialization{Style: "form"})
		require.NoError(t, err)
		require.Len(t, branches, 2)
		require.Equal(t, KeySet{Residual: true}, keys)
		require.NoError(t, CheckNamespaces(NamedQuery, map[string]KeySet{"n": keys}))
	})
	t.Run("nested", func(t *testing.T) {
		branches, keys, err := ResolveUnion(NamedQuery, "n", []BranchShape{{Shape: Union, Branches: []BranchShape{{Shape: Scalar}, {Shape: ScalarMap}}}}, Serialization{})
		require.NoError(t, err)
		require.Len(t, branches[0].Inner, 2)
		require.Equal(t, Form, branches[0].Inner[1].Codec.Style)
		require.True(t, keys.Residual)
	})
}
