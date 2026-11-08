package http

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loom "github.com/CaliLuke/loom/pkg"
)

func TestNormalizeParamDecodeError(t *testing.T) {
	fault := loom.Fault("%s", "parameter tags: invalid default")
	for _, tc := range []struct {
		name      string
		err       error
		wantName  string
		wantFault bool
	}{
		{"server fault keeps its classification", fault, "fault", true},
		{"wrapped server fault keeps its classification", fmt.Errorf("decoder candidate 0: %w", fault), "fault", true},
		{"plain error is a decode error", errors.New("parameter tags: invalid value"), loom.DecodePayload, false},
		{"client service error is a decode error", loom.MissingFieldError("name", "query string"), loom.DecodePayload, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got *loom.ServiceError
			require.ErrorAs(t, NormalizeParamDecodeError(tc.err), &got)
			assert.Equal(t, tc.wantName, got.Name)
			assert.Equal(t, tc.wantFault, got.Fault)
		})
	}
}

// TestTextBranchDecodePropagatesAssignErrors keeps the public TextBranch
// contract: Assign reports a mismatch as (false, nil), and every error it
// returns reaches the caller unchanged, whatever its classification.
func TestTextBranchDecodePropagatesAssignErrors(t *testing.T) {
	broken := errors.New("broken codec configuration")
	fault := loom.Fault("%s", "parameter choice: invalid default")
	for _, tc := range []struct {
		name        string
		matched     bool
		err         error
		wantMatched bool
	}{
		{"match", true, nil, true},
		{"mismatch", false, nil, false},
		{"unknown error", false, broken, false},
		{"wrapped unknown error", false, fmt.Errorf("assign candidate: %w", broken), false},
		{"validation error", false, loom.InvalidLengthError("choice", 1, 2, true), false},
		{"server fault", false, fault, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			branch := TextBranch{
				View: func() (ParamValue, error) {
					return ParamValue{Present: true, Text: "x"}, nil
				},
				Assign: func(ParamValue) (bool, error) {
					return tc.matched, tc.err
				},
			}
			matched, err := branch.Decode()
			assert.Equal(t, tc.wantMatched, matched)
			if tc.err == nil {
				assert.NoError(t, err)
				return
			}
			assert.Equal(t, tc.err, err, "Assign errors are returned unchanged")
		})
	}
}

// TestTextUnionCandidateErrorFailsMatch keeps exact-one selection's callback
// contract: every candidate's decoder runs, and an error from any candidate,
// including one that is not the schema match, fails the match with its cause.
// A faulty authored default is such an error; ordinary mismatches are not.
func TestTextUnionCandidateErrorFailsMatch(t *testing.T) {
	broken := errors.New("broken codec configuration")
	fault := loom.Fault("%s", "parameter choice: invalid default")
	for _, tc := range []struct {
		name      string
		err       error
		wantIndex int
	}{
		{"mismatch in the unselected candidate", nil, 1},
		{"fault in the unselected candidate", fault, -1},
		{"unknown error in the unselected candidate", broken, -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			unselected := TextBranch{
				View: func() (ParamValue, error) {
					return ParamValue{Present: true, Text: "x"}, nil
				},
				Assign: func(ParamValue) (bool, error) {
					return false, tc.err
				},
			}
			branches := []TextUnionBranch{
				{Schema: func() (bool, error) { return false, nil }, Decode: unselected.Decode},
				{Schema: func() (bool, error) { return true, nil }, Decode: func() (bool, error) { return true, nil }},
			}
			index, err := MatchUntaggedText("choice", branches, false)
			assert.Equal(t, tc.wantIndex, index)
			if tc.err == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tc.err, "the candidate's cause is preserved")
		})
	}
}
