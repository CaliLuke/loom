package unionmatch

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSelectionNonemptyCardinality(t *testing.T) {
	yes := func() (bool, error) {
		return true, nil
	}
	no := func() (bool, error) {
		return false, nil
	}
	fatal := errors.New("broken evaluator")
	fail := func() (bool, error) {
		return false, fatal
	}
	for _, tc := range []struct {
		name       string
		candidates []Candidate
		selected   int
		message    string
		mismatch   bool
		cause      error
	}{
		{"empty", nil, -1, "at least one candidate", false, nil},
		{"singleton", []Candidate{{Schema: yes, Decode: yes}}, 0, "", false, nil},
		{"singleton schema mismatch", []Candidate{{Schema: no, Decode: yes}}, -1, "matched 0 branches in schema", true, nil},
		{"singleton decoder mismatch", []Candidate{{Schema: yes, Decode: no}}, -1, "matched 0 branches in decoder", true, nil},
		{"singleton missing schema", []Candidate{{Decode: yes}}, -1, "incomplete", false, nil},
		{"singleton missing decoder", []Candidate{{Schema: yes}}, -1, "incomplete", false, nil},
		{"singleton schema fatal", []Candidate{{Schema: fail, Decode: yes}}, -1, "schema candidate 0", false, fatal},
		{"singleton decoder fatal", []Candidate{{Schema: yes, Decode: fail}}, -1, "decoder candidate 0", false, fatal},
		{"ordinary unique", []Candidate{{Schema: no, Decode: no}, {Schema: yes, Decode: yes}}, 1, "", false, nil},
		{"ordinary ambiguous", []Candidate{{Schema: yes, Decode: yes}, {Schema: yes, Decode: yes}}, -1, "matched 2 branches in schema", true, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			selected, err := Select("Value", tc.candidates)
			require.Equal(t, tc.selected, selected)
			if tc.message == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.message)
				var mismatch *Mismatch
				require.Equal(t, tc.mismatch, errors.As(err, &mismatch))
				if tc.cause != nil {
					require.ErrorIs(t, err, tc.cause)
				}
			}
		})
	}
	for _, tc := range []struct {
		name       string
		candidates []Candidate
		selected   int
		message    string
		cause      error
	}{
		{"empty", nil, -1, "at least one candidate", nil},
		{"singleton", []Candidate{{Schema: yes}}, 0, "", nil},
		{"mismatch", []Candidate{{Schema: no}}, -1, "matched 0 branches", nil},
		{"incomplete", []Candidate{{}}, -1, "has no schema", nil},
		{"fatal", []Candidate{{Schema: fail}}, -1, "schema candidate 0", fatal},
	} {
		t.Run("schema "+tc.name, func(t *testing.T) {
			selected, err := SelectSchema("Value", tc.candidates)
			require.Equal(t, tc.selected, selected)
			if tc.message == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.message)
			}
			if tc.cause != nil {
				require.ErrorIs(t, err, tc.cause)
			}
		})
	}
}
