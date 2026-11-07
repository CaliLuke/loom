package http

import (
	"encoding/json/jsontext"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	loom "github.com/CaliLuke/loom/pkg"
)

func TestFormObjectJSON(t *testing.T) {
	for _, tc := range []struct {
		name, input, want string
		match             bool
	}{
		{"empty present body", "", `{}`, true},
		{"separator only", "&&", `{}`, true},
		{"exact integer", "count=999999999999999999999999999999", `{"count":999999999999999999999999999999}`, true},
		{"first scalar value", "count=7&count=oops", `{"count":7}`, true},
		{"JSON null", "data=null", `{"data":null}`, true},
		{"nested JSON names", "data=%7B%22json_name%22%3A1%7D", `{"data":{"json_name":1}}`, true},
		{"decoded unknown key", "a%20b=%FF", `{}`, true},
		{"repeated unknown key", "extra=%FF&extra=anything", `{}`, true},
		{"unknown empty key", "=data", `{}`, true},
		{"invalid integer", "count=oops", "", false},
		{"invalid JSON", "data=%7B", "", false},
		{"invalid unknown name", "%FF=value", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pairs, err := ParseRawPairs(tc.input)
			require.NoError(t, err)
			fields := []FormField{
				{Name: "count", Keys: FormKeySet{Literals: []string{"count"}}, View: func() (jsontext.Value, bool, error) {
					return FormTextValue(formTestTextBranch(pairs, "count", ParamBranchShape{Kind: ParamScalar, Element: ParamInteger}, &loom.JSONShape{Kind: "integer"}))
				}},
				{Name: "data", Keys: FormKeySet{Literals: []string{"data"}}, View: func() (jsontext.Value, bool, error) {
					return FormJSONValue(pairs, "data", FieldValue)
				}},
			}
			projection, match, err := FormObjectJSON(pairs, fields)
			require.NoError(t, err)
			require.Equal(t, tc.match, match)
			require.Equal(t, tc.want, string(projection.Value))
		})
	}
}

func TestFormJSONValue(t *testing.T) {
	for _, tc := range []struct {
		input, want string
		match       bool
	}{
		{"", "", true},
		{"n=null", "null", true},
		{"n=+1+", " 1 ", true},
		{"n=null&n=invalid", "null", true},
		{"n=", "", false},
		{"n=+", "", false},
		{"n=%7B", "", false},
		{"n=1+2", "", false},
		{"n=%7B%22x%22%3A1%2C%22x%22%3A2%7D", "", false},
	} {
		t.Run(tc.input, func(t *testing.T) {
			pairs, err := ParseRawPairs(tc.input)
			require.NoError(t, err)
			raw, match, err := FormJSONValue(pairs, "n", FieldValue)
			require.NoError(t, err)
			require.Equal(t, tc.match, match)
			require.Equal(t, tc.want, string(raw))
			if tc.input == "" {
				require.Nil(t, raw)
			}
		})
	}
}

func TestFormTextValueCodecGrammar(t *testing.T) {
	for _, tc := range []struct {
		input, want string
		match       bool
	}{
		{"", "", true},
		{"n=a%2Cb,c", `["a,b","c"]`, true},
		{"n=", `[]`, true},
		{"n=a&n=b", "", false},
	} {
		t.Run(tc.input, func(t *testing.T) {
			pairs, err := ParseRawPairs(tc.input)
			require.NoError(t, err)
			branch := formTestTextBranch(pairs, "n", ParamBranchShape{Kind: ParamList, Element: ParamString}, nil)
			raw, match, err := FormTextValue(branch)
			require.NoError(t, err)
			require.Equal(t, tc.match, match)
			require.Equal(t, tc.want, string(raw))
		})
	}
}

func TestFormObjectJSONUnionClaims(t *testing.T) {
	pairs, err := ParseRawPairs("x=ok&y=oops")
	require.NoError(t, err)
	branches := make([]FormValueBranch, 0, 2)
	for _, spec := range []struct {
		name     string
		kind     ParamScalarKind
		jsonKind string
	}{{"x", ParamString, "string"}, {"y", ParamInteger, "integer"}} {
		shape := &loom.JSONShape{Kind: "object", Fields: []loom.JSONShapeField{{Name: spec.name, Required: true, Shape: &loom.JSONShape{Kind: spec.jsonKind}}}}
		branch := TextBranch{
			View: func() (ParamValue, error) {
				return DecodePairsParam(pairs, "choice", ParamCodec{Style: ParamForm, Explode: true}, ParamPairs, ParamNamespace{OwnedLiterals: []string{spec.name}})
			},
			Shape: ParamBranchShape{Kind: ParamPairs, Properties: map[string]ParamScalarKind{spec.name: spec.kind}},
		}
		branches = append(branches, FormValueBranch{View: func() (jsontext.Value, bool, error) {
			return FormTextValue(branch)
		}, Member: shape.Match, Evaluate: shape.Evaluated})
	}
	fields := []FormField{{Name: "choice", Keys: FormKeySet{Literals: []string{"x", "y"}}, View: func() (jsontext.Value, bool, error) {
		return FormTextUnionValue("choice", branches, false)
	}}}
	projection, match, err := FormObjectJSON(pairs, fields)
	require.NoError(t, err)
	require.True(t, match)
	require.Equal(t, `{"choice":{"x":"ok"}}`, string(projection.Value))
	require.ElementsMatch(t, []string{"x", "y"}, projection.FieldClaims["choice"])
	root := &loom.JSONShape{Kind: "object", Closed: true, Fields: []loom.JSONShapeField{{Name: "choice", Shape: &loom.JSONShape{Kind: "any"}}}}
	match, err = projection.Schema(root.Match, true)
	require.NoError(t, err)
	require.True(t, match)

	// The rejected selected-only policy leaks y into the root instance.
	fields[0].Keys.Literals = []string{"x"}
	projection, match, err = FormObjectJSON(pairs, fields)
	require.NoError(t, err)
	require.True(t, match)
	match, err = projection.Schema(root.Match, true)
	require.NoError(t, err)
	require.False(t, match)
}

func TestFormTextUnionValue(t *testing.T) {
	for _, tc := range []struct {
		name, input, want string
		match             bool
	}{
		{"absent", "", "", true},
		{"wide integer", "n=999999999999999999999999", "999999999999999999999999", true},
		{"boolean", "n=true", "true", true},
		{"outer string", "n=ok", `"ok"`, true},
		{"no member", "n=other", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pairs, err := ParseRawPairs(tc.input)
			require.NoError(t, err)
			inner := []FormValueBranch{
				formTestValueBranch(pairs, ParamInteger, &loom.JSONShape{Kind: "integer"}),
				formTestValueBranch(pairs, ParamBoolean, &loom.JSONShape{Kind: "boolean"}),
			}
			outer := []FormValueBranch{
				{View: func() (jsontext.Value, bool, error) {
					return FormTextUnionValue("inner", inner, false)
				}, Member: (&loom.JSONShape{Kind: "any"}).Match},
				formTestValueBranch(pairs, ParamString, &loom.JSONShape{Kind: "string", Rules: loom.JSONShapeRules{Enums: [][]jsontext.Value{{jsontext.Value(`"ok"`)}}}}),
			}
			raw, match, err := FormTextUnionValue("outer", outer, false)
			require.NoError(t, err)
			require.Equal(t, tc.match, match)
			require.Equal(t, tc.want, string(raw))
		})
	}
	t.Run("schema ambiguity independent of decoder widths", func(t *testing.T) {
		pairs, err := ParseRawPairs("n=999999999999999999999999")
		require.NoError(t, err)
		branches := []FormValueBranch{
			formTestValueBranch(pairs, ParamInteger, &loom.JSONShape{Kind: "integer"}),
			formTestValueBranch(pairs, ParamNumber, &loom.JSONShape{Kind: "number"}),
		}
		raw, match, err := FormTextUnionValue("n", branches, false)
		require.NoError(t, err)
		require.False(t, match)
		require.Nil(t, raw)
	})
}

func TestFormTextUnionValueClosure(t *testing.T) {
	shape := &loom.JSONShape{Kind: "object", Fields: []loom.JSONShapeField{{Name: "x", Shape: &loom.JSONShape{Kind: "string"}}}}
	for _, tc := range []struct {
		raw   string
		match bool
	}{{`{"x":"ok"}`, true}, {`{"x":"ok","extra":1}`, false}} {
		t.Run(tc.raw, func(t *testing.T) {
			calls := 0
			branches := []FormValueBranch{
				{View: func() (jsontext.Value, bool, error) {
					calls++
					return jsontext.Value(tc.raw), true, nil
				}, Member: shape.Match, Evaluate: shape.Evaluated},
				{View: func() (jsontext.Value, bool, error) {
					return jsontext.Value(tc.raw), true, nil
				}, Member: (&loom.JSONShape{Kind: "string"}).Match},
			}
			_, match, err := FormTextUnionValue("n", branches, true)
			require.NoError(t, err)
			require.Equal(t, tc.match, match)
			require.Equal(t, 1, calls)
		})
	}
}

func TestFormMapJSON(t *testing.T) {
	for _, tc := range []struct {
		input, want string
		match       bool
	}{{"", `{}`, true}, {"=null&numbers=%5B1%2C2%5D", `{"":null,"numbers":[1,2]}`, true}, {"x=1&x=2", `{"x":1}`, true}, {"x=invalid", "", false}, {"%FF=1", "", false}} {
		t.Run(tc.input, func(t *testing.T) {
			pairs, err := ParseRawPairs(tc.input)
			require.NoError(t, err)
			calls := make(map[string]int)
			projection, match, err := FormMapJSON(pairs, func(name string) (jsontext.Value, bool, error) {
				calls[name]++
				return FormJSONValue(pairs, name, FieldValue)
			})
			require.NoError(t, err)
			require.Equal(t, tc.match, match)
			require.Equal(t, tc.want, string(projection.Value))
			for _, count := range calls {
				require.Equal(t, 1, count)
			}
		})
	}
}

func TestFormProjectionEvaluatorErrors(t *testing.T) {
	sentinel := errors.New("evaluator failed")
	view := func() (jsontext.Value, bool, error) {
		return jsontext.Value(`1`), true, nil
	}
	member := func(jsontext.Value) (bool, error) {
		return false, sentinel
	}
	_, _, err := FormTextUnionValue("n", []FormValueBranch{{View: view, Member: member}, {View: view, Member: member}}, false)
	require.ErrorIs(t, err, sentinel)
	_, _, err = FormObjectJSON(RawPairs{}, []FormField{{Name: "x"}})
	require.Error(t, err)
	_, _, err = FormTextValue(TextBranch{})
	require.Error(t, err)
}

func formTestTextBranch(pairs RawPairs, name string, shape ParamBranchShape, member *loom.JSONShape) TextBranch {
	return TextBranch{View: func() (ParamValue, error) {
		return DecodePairsParam(pairs, name, ParamCodec{Style: ParamForm}, shape.Kind, ParamNamespace{})
	}, Shape: shape, Member: member.Match, Assign: func(ParamValue) (bool, error) {
		panic("schema projection must not invoke typed conversion")
	}}
}

func formTestValueBranch(pairs RawPairs, kind ParamScalarKind, shape *loom.JSONShape) FormValueBranch {
	branch := formTestTextBranch(pairs, "n", ParamBranchShape{Kind: ParamScalar, Element: kind}, shape)
	return FormValueBranch{View: func() (jsontext.Value, bool, error) {
		return FormTextValue(branch)
	}, Member: shape.Match, Evaluate: shape.Evaluated}
}

func TestFormObjectJSONNamespaces(t *testing.T) {
	for _, tc := range []struct {
		name, input      string
		keys             FormKeySet
		claimed, unknown []string
	}{
		{"deepObject claims", "options%5Bx%5D=ignored&outside=%FF", FormKeySet{Prefixes: []string{"options"}}, []string{"options[x]"}, []string{"outside"}},
		{"residual field claims", "=ignored&dynamic=ignored", FormKeySet{Residual: true}, []string{"", "dynamic"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pairs, err := ParseRawPairs(tc.input)
			require.NoError(t, err)
			projection, match, err := FormObjectJSON(pairs, []FormField{{Name: "wire_name", Keys: tc.keys, View: func() (jsontext.Value, bool, error) {
				return jsontext.Value("null"), true, nil
			}}})
			require.NoError(t, err)
			require.True(t, match)
			require.Equal(t, `{"wire_name":null}`, string(projection.Value))
			require.ElementsMatch(t, tc.claimed, projection.FieldClaims["wire_name"])
			require.ElementsMatch(t, tc.unknown, projection.Unclaimed)
		})
	}
	t.Run("unknown values do not affect open membership", func(t *testing.T) {
		pairs, err := ParseRawPairs("%E2%98%83=%FF")
		require.NoError(t, err)
		projection, match, err := FormObjectJSON(pairs, nil)
		require.NoError(t, err)
		require.True(t, match)
		require.Equal(t, `{}`, string(projection.Value))
		require.Equal(t, []string{"☃"}, projection.Unclaimed)
		for _, closed := range []bool{false, true} {
			shape := &loom.JSONShape{Kind: "object", Closed: closed}
			match, err := projection.Schema(shape.Match, closed)
			require.NoError(t, err)
			require.Equal(t, !closed, match)
		}
	})
}

func TestFormMapJSONCapturedCodecs(t *testing.T) {
	t.Run("exploded list", func(t *testing.T) {
		pairs, err := ParseRawPairs("=one&=two&other=three")
		require.NoError(t, err)
		projection, match, err := FormMapJSON(pairs, func(name string) (jsontext.Value, bool, error) {
			return FormTextValue(TextBranch{View: func() (ParamValue, error) {
				return DecodePairsParam(pairs, name, ParamCodec{Style: ParamForm, Explode: true}, ParamList, ParamNamespace{})
			}, Shape: ParamBranchShape{Kind: ParamList, Element: ParamString}})
		})
		require.NoError(t, err)
		require.True(t, match)
		require.Equal(t, `{"":["one","two"],"other":["three"]}`, string(projection.Value))
	})
	t.Run("union dynamic values", func(t *testing.T) {
		pairs, err := ParseRawPairs("=12&other=true")
		require.NoError(t, err)
		projection, match, err := FormMapJSON(pairs, func(name string) (jsontext.Value, bool, error) {
			branches := make([]FormValueBranch, 0, 2)
			for _, spec := range []struct {
				kind ParamScalarKind
				name string
			}{{ParamInteger, "integer"}, {ParamBoolean, "boolean"}} {
				shape := &loom.JSONShape{Kind: spec.name}
				branch := formTestTextBranch(pairs, name, ParamBranchShape{Kind: ParamScalar, Element: spec.kind}, shape)
				branches = append(branches, FormValueBranch{View: func() (jsontext.Value, bool, error) {
					return FormTextValue(branch)
				}, Member: shape.Match})
			}
			return FormTextUnionValue(name, branches, false)
		})
		require.NoError(t, err)
		require.True(t, match)
		require.Equal(t, `{"":12,"other":true}`, string(projection.Value))
	})
}

func TestFormProjectionConfiguration(t *testing.T) {
	_, _, err := FormTextValue(TextBranch{View: func() (ParamValue, error) {
		return ParamValue{}, nil
	}, Shape: ParamBranchShape{Kind: ParamKind(255)}})
	require.Error(t, err)
	_, _, err = FormMapJSON(RawPairs{}, nil)
	require.Error(t, err)
	_, _, err = FormTextUnionValue("n", nil, false)
	require.Error(t, err)
	view := func() (jsontext.Value, bool, error) {
		return jsontext.Value(`{}`), true, nil
	}
	_, _, err = FormObjectJSON(RawPairs{}, []FormField{{Name: "x", View: view}, {Name: "x", View: view}})
	require.Error(t, err)
	_, _, err = FormObjectJSON(RawPairs{}, []FormField{{Name: string([]byte{255}), View: view}})
	require.Error(t, err)
	_, match, err := FormMapJSON(RawPairs{pairs: []rawParamPair{{name: "x", value: "1"}}}, func(string) (jsontext.Value, bool, error) {
		return nil, true, nil
	})
	require.NoError(t, err)
	require.False(t, match)
}

func TestFormProjectionConfigurationBeforeInput(t *testing.T) {
	mismatch := func() (jsontext.Value, bool, error) {
		return nil, false, nil
	}
	valid := func() (jsontext.Value, bool, error) {
		return jsontext.Value(`1`), true, nil
	}
	for _, fields := range [][]FormField{
		{{Name: "first", View: mismatch}, {Name: "broken"}},
		{{Name: "first", View: mismatch}, {Name: "first", View: valid}},
		{{Name: "first", View: mismatch}, {Name: string([]byte{255}), View: valid}},
		{{Name: "first", View: mismatch}, {Name: "second", Keys: FormKeySet{Literals: []string{string([]byte{255})}}, View: valid}},
		{{Name: "first", Keys: FormKeySet{Literals: []string{"x"}}, View: mismatch}, {Name: "second", Keys: FormKeySet{Literals: []string{"x"}}, View: valid}},
	} {
		_, _, err := FormObjectJSON(RawPairs{}, fields)
		require.Error(t, err, "invalid configuration must not become ordinary branch mismatch")
	}
	invalid := TextBranch{View: func() (ParamValue, error) {
		return ParamValue{}, paramError("n", "repeated delimited binding")
	}, Shape: ParamBranchShape{Kind: ParamKind(255)}, Member: func(jsontext.Value) (bool, error) {
		return true, nil
	}}
	_, _, err := FormTextValue(invalid)
	require.ErrorContains(t, err, "invalid parameter branch shape")
	_, err = invalid.Schema()
	require.ErrorContains(t, err, "invalid parameter branch shape")
	_, err = invalid.Annotations()
	require.ErrorContains(t, err, "invalid parameter branch shape")
}

func TestFormProjectionPhysicalClosure(t *testing.T) {
	pairs, err := ParseRawPairs("x=ok&filter=%FF")
	require.NoError(t, err)
	field := FormField{Name: "filter", Keys: FormKeySet{Literals: []string{"x"}}, View: func() (jsontext.Value, bool, error) {
		return FormTextValue(TextBranch{View: func() (ParamValue, error) {
			return DecodePairsParam(pairs, "filter", ParamCodec{Style: ParamForm, Explode: true}, ParamPairs, ParamNamespace{OwnedLiterals: []string{"x"}})
		}, Shape: ParamBranchShape{Kind: ParamPairs, Properties: map[string]ParamScalarKind{"x": ParamString}}})
	}}
	projection, match, err := FormObjectJSON(pairs, []FormField{field})
	require.NoError(t, err)
	require.True(t, match)
	require.Equal(t, `{"filter":{"x":"ok"}}`, string(projection.Value))
	require.ElementsMatch(t, []string{"x", "filter"}, projection.Members)
	require.Equal(t, []string{"filter"}, projection.Unclaimed)
	shape := &loom.JSONShape{Kind: "object", Fields: []loom.JSONShapeField{{Name: "filter", Required: true, Shape: &loom.JSONShape{Kind: "any"}}}}
	for _, closed := range []bool{false, true} {
		match, err := projection.Schema(shape.Match, closed)
		require.NoError(t, err)
		require.Equal(t, !closed, match)
	}
	for _, explicitOpen := range []bool{false, true} {
		annotations, err := projection.Annotations(shape.Evaluated, explicitOpen)
		require.NoError(t, err)
		require.Contains(t, annotations.Evaluated, "x")
		require.Equal(t, explicitOpen, len(annotations.Evaluated) == 2)
		branch := TextUnionBranch{Schema: func() (bool, error) {
			return projection.Schema(shape.Match, false)
		}, Decode: func() (bool, error) {
			return true, nil
		}, Annotations: func() (TextUnionAnnotations, error) {
			return projection.Annotations(shape.Evaluated, explicitOpen)
		}}
		miss := TextUnionBranch{Schema: func() (bool, error) {
			return false, nil
		}, Decode: func() (bool, error) {
			return false, nil
		}}
		_, err = MatchUntaggedText("root", []TextUnionBranch{branch, miss}, true)
		if explicitOpen {
			require.NoError(t, err)
		} else {
			require.ErrorContains(t, err, "unevaluated")
		}
		_, err = MatchUntaggedText("root", []TextUnionBranch{branch, branch}, true)
		require.ErrorContains(t, err, "matched 2 branches in schema")
	}
}

func TestFormProjectionResidualExcludesExplicitClaims(t *testing.T) {
	pairs, err := ParseRawPairs("named=value&dynamic=other&=empty")
	require.NoError(t, err)
	projection, match, err := FormObjectJSON(pairs, []FormField{
		{Name: "named", Keys: FormKeySet{Literals: []string{"named"}}, View: func() (jsontext.Value, bool, error) {
			return jsontext.Value(`"value"`), true, nil
		}},
		{Name: "rest", Keys: FormKeySet{Residual: true}, View: func() (jsontext.Value, bool, error) {
			return jsontext.Value(`{"dynamic":"other","":"empty"}`), true, nil
		}},
	})
	require.NoError(t, err)
	require.True(t, match)
	require.Empty(t, projection.Unclaimed)
	require.Equal(t, []string{"named"}, projection.FieldClaims["named"])
	require.ElementsMatch(t, []string{"dynamic", ""}, projection.FieldClaims["rest"])
}

func TestFormProjectionEvaluatorsBeforeClosure(t *testing.T) {
	pairs, err := ParseRawPairs("unknown=value")
	require.NoError(t, err)
	projection, match, err := FormObjectJSON(pairs, nil)
	require.NoError(t, err)
	require.True(t, match)
	_, err = projection.Schema(nil, true)
	require.Error(t, err)
	_, err = projection.Schema((&loom.JSONShape{Kind: "invalid"}).Match, true)
	require.Error(t, err)
	_, err = projection.Annotations(nil, false)
	require.Error(t, err)
	sentinel := errors.New("bad annotation evaluator")
	_, err = projection.Annotations(func(jsontext.Value) ([]string, error) {
		return nil, sentinel
	}, false)
	require.ErrorIs(t, err, sentinel)
	_, _, err = FormTextUnionValue("n", []FormValueBranch{
		{View: func() (jsontext.Value, bool, error) {
			return nil, false, nil
		}, Member: (&loom.JSONShape{Kind: "integer"}).Match},
		{},
	}, false)
	require.Error(t, err)
}
