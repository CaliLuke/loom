package http

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNestedTextUnionClosureOrdering(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		innerClosed, outerClosed bool
		extra, innerAmbiguous    bool
		fallback                 bool
		want                     int
	}{
		{"nested accepts", true, true, false, false, false, 0},
		{"inner closure leaves outer alternative", true, false, true, false, true, 1},
		{"outer closure consumes selected annotations", false, true, true, false, false, -1},
		{"ambiguity precedes closure", true, true, true, true, true, 1},
		{"open union preserves acceptance", false, false, true, false, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			members := []string{"x"}
			if tc.extra {
				members = append(members, "extra")
			}
			annotations := TextUnionAnnotations{Members: members, Evaluated: []string{"x"}}
			calls := 0
			selected := -1
			inner := ComposeTextUnion("inner", []TextUnionBranch{
				{
					Schema: func() (bool, error) {
						return true, nil
					},
					Decode: func() (bool, error) {
						return true, nil
					},
					Annotations: func() (TextUnionAnnotations, error) {
						calls++
						return annotations, nil
					},
				},
				{
					Schema: func() (bool, error) {
						return tc.innerAmbiguous, nil
					},
					Decode: func() (bool, error) {
						return tc.innerAmbiguous, nil
					},
				},
			}, tc.innerClosed, func(index int) (bool, error) {
				selected = index
				return true, nil
			})
			index, err := MatchUntaggedText("outer", []TextUnionBranch{inner, {
				Schema: func() (bool, error) {
					return tc.fallback, nil
				},
				Decode: func() (bool, error) {
					return tc.fallback, nil
				},
				Annotations: func() (TextUnionAnnotations, error) {
					return TextUnionAnnotations{}, nil
				},
			}}, tc.outerClosed)
			if tc.want < 0 {
				require.ErrorContains(t, err, "unevaluated member")
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.want, index)
			require.LessOrEqual(t, calls, 1, "annotations are retained across schema, decoder and enclosing closure")
			if tc.innerClosed && (tc.extra || tc.innerAmbiguous) {
				require.Equal(t, -1, selected, "rejected inner candidate is never assigned")
			}
		})
	}
}

func TestNestedTextUnionAnnotationError(t *testing.T) {
	broken := errors.New("broken annotations")
	inner := ComposeTextUnion("inner", []TextUnionBranch{
		{
			Schema: func() (bool, error) {
				return true, nil
			},
			Decode: func() (bool, error) {
				return true, nil
			},
			Annotations: func() (TextUnionAnnotations, error) {
				return TextUnionAnnotations{}, broken
			},
		},
		{
			Schema: func() (bool, error) {
				return false, nil
			},
			Decode: func() (bool, error) {
				return false, nil
			},
		},
	}, true, func(int) (bool, error) {
		t.Error("unexpected assignment")
		return false, nil
	})
	_, err := inner.Schema()
	require.ErrorIs(t, err, broken)
	_, err = inner.Decode()
	require.ErrorIs(t, err, broken)
}
