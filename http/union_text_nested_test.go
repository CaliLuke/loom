package http

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComposeTextUnion(t *testing.T) {
	for _, tc := range []struct {
		name   string
		schema [2]bool
		decode [2]bool
		outer  bool
		want   int
	}{
		{"nested ambiguity leaves outer alternative", [2]bool{true, true}, [2]bool{true, true}, true, 1},
		{"nested width split leaves outer alternative", [2]bool{true, true}, [2]bool{true, false}, true, 1},
		{"nested unique", [2]bool{true, false}, [2]bool{true, false}, false, 0},
		{"nested identity disagreement", [2]bool{true, false}, [2]bool{false, true}, false, -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var schemaCalls, decodeCalls [2]int
			inner := make([]TextUnionBranch, 2)
			for index := range inner {
				inner[index] = TextUnionBranch{
					Schema: func() (bool, error) {
						schemaCalls[index]++
						return tc.schema[index], nil
					},
					Decode: func() (bool, error) {
						decodeCalls[index]++
						return tc.decode[index], nil
					},
				}
			}
			selected := -1
			nested := ComposeTextUnion("inner", inner, false, func(index int) (bool, error) {
				selected = index
				return true, nil
			})
			outer := []TextUnionBranch{nested, {
				Schema: func() (bool, error) {
					return tc.outer, nil
				},
				Decode: func() (bool, error) {
					return tc.outer, nil
				},
			}}
			index, err := MatchUntaggedText("outer", outer, false)
			if tc.want < 0 {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.want, index)
			require.Equal(t, [2]int{1, 1}, schemaCalls)
			require.Equal(t, [2]int{1, 1}, decodeCalls)
			if tc.want == 0 {
				require.Equal(t, 0, selected)
			} else {
				require.Equal(t, -1, selected)
			}
		})
	}
}

func TestComposeTextUnionUnexpectedError(t *testing.T) {
	broken := errors.New("broken evaluator")
	nested := ComposeTextUnion("inner", []TextUnionBranch{
		{Schema: func() (bool, error) {
			return false, broken
		}, Decode: func() (bool, error) {
			return false, nil
		}},
		{Schema: func() (bool, error) {
			return true, nil
		}, Decode: func() (bool, error) {
			return true, nil
		}},
	}, false, func(int) (bool, error) {
		t.Error("unexpected assignment")
		return false, nil
	})
	_, err := nested.Schema()
	require.ErrorIs(t, err, broken)
	_, err = nested.Decode()
	require.ErrorIs(t, err, broken)
}

func TestComposeTextUnionRetainsPredicateResults(t *testing.T) {
	var schemaCalls, decodeCalls, annotationCalls int
	branch := TextUnionBranch{
		Schema: func() (bool, error) {
			schemaCalls++
			return true, nil
		},
		Decode: func() (bool, error) {
			decodeCalls++
			return true, nil
		},
		Annotations: func() (TextUnionAnnotations, error) {
			annotationCalls++
			return TextUnionAnnotations{Members: []string{"x"}, Evaluated: []string{"x"}}, nil
		},
	}
	miss := TextUnionBranch{
		Schema: func() (bool, error) {
			return false, nil
		},
		Decode: func() (bool, error) {
			return false, nil
		},
	}
	for range 32 {
		branch = ComposeTextUnion("nested", []TextUnionBranch{branch, miss}, true, func(index int) (bool, error) {
			require.Equal(t, 0, index)
			return true, nil
		})
	}
	for range 2 {
		matched, err := branch.Schema()
		require.NoError(t, err)
		require.True(t, matched)
		matched, err = branch.Decode()
		require.NoError(t, err)
		require.True(t, matched)
	}
	require.Equal(t, 1, schemaCalls)
	require.Equal(t, 1, decodeCalls)
	require.Equal(t, 1, annotationCalls)
}
