package unionmatch

import (
	"errors"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSelectCountsSchemaAndDecoderIndependently(t *testing.T) {
	for _, tc := range []struct {
		name     string
		schema   []bool
		decode   []bool
		want     int
		mismatch string
	}{
		{"unique", []bool{false, true}, []bool{false, true}, 1, ""},
		{"schema overlap", []bool{true, true}, []bool{true, false}, -1, "matched 2 branches in schema"},
		{"decoder overlap", []bool{true, false}, []bool{true, true}, -1, "matched 2 branches in decoder"},
		{"no schema", []bool{false, false}, []bool{true, false}, -1, "matched 0 branches in schema"},
		{"no decoder", []bool{true, false}, []bool{false, false}, -1, "matched 0 branches in decoder"},
		{"different identity", []bool{true, false}, []bool{false, true}, -1, "identities differ"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			candidates := make([]Candidate, len(tc.schema))
			for i := range candidates {
				candidates[i] = Candidate{
					Schema: func() (bool, error) {
						return tc.schema[i], nil
					},
					Decode: func() (bool, error) {
						return tc.decode[i], nil
					},
				}
			}
			selected, err := Select("Value", candidates)
			require.Equal(t, tc.want, selected)
			if tc.mismatch == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.mismatch)
				var mismatch *Mismatch
				require.ErrorAs(t, err, &mismatch)
			}
		})
	}
}

func TestSelectVisitsEveryCandidateInEitherOrder(t *testing.T) {
	for _, order := range [][]int{{0, 1, 2}, {2, 1, 0}} {
		var calls []int
		candidates := make([]Candidate, 0, len(order))
		for _, index := range order {
			candidates = append(candidates, Candidate{
				Schema: func() (bool, error) {
					calls = append(calls, index*2)
					return true, nil
				},
				Decode: func() (bool, error) {
					calls = append(calls, index*2+1)
					return true, nil
				},
			})
		}
		_, err := Select("Value", candidates)
		require.ErrorContains(t, err, "matched 3 branches in schema")
		want := make([]int, 0, 2*len(order))
		for _, index := range order {
			want = append(want, index*2, index*2+1)
		}
		require.Equal(t, want, calls)
	}
}

func TestSelectPropagatesFatalErrors(t *testing.T) {
	fatal := errors.New("broken evaluator")
	yes := func() (bool, error) {
		return true, nil
	}
	fail := func() (bool, error) {
		return false, fatal
	}
	for _, schema := range []bool{false, true} {
		broken := Candidate{Schema: yes, Decode: fail}
		if schema {
			broken = Candidate{Schema: fail, Decode: yes}
		}
		for _, reverse := range []bool{false, true} {
			candidates := []Candidate{{Schema: yes, Decode: yes}, {Schema: yes, Decode: yes}, broken}
			if reverse {
				slices.Reverse(candidates)
			}
			selected, err := Select("Value", candidates)
			require.Equal(t, -1, selected)
			require.ErrorIs(t, err, fatal)
		}
	}
}
