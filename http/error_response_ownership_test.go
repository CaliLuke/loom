package http

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	loom "github.com/CaliLuke/loom/pkg"
)

func TestProblemResponseCompleteErrorOwnership(t *testing.T) {
	branch := loom.WithErrorRemedy(loom.TemporaryError("busy", "branch detail"), &loom.ErrorRemedy{RetryHint: "branch retry"})
	branch.ID = "branch-id"
	other := errors.New("independent cleanup")
	outer := loom.NewServiceError(errors.Join(branch, other), "outer", false, false, true)
	outer.ID = "outer-id"
	for _, tc := range []struct {
		name  string
		err   error
		owned bool
	}{
		{"direct", branch, true},
		{"wrapped", fmt.Errorf("diagnostic: %w", branch), true},
		{"single join", errors.Join(nil, branch), true},
		{"join", errors.Join(branch, other), false},
		{"reverse", errors.Join(other, branch), false},
		{"wrapped join", fmt.Errorf("diagnostic: %w", errors.Join(branch, other)), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := NewProblemResponse(t.Context(), tc.err, 0, "", "")
			if tc.owned {
				require.Equal(t, "busy", got.Code)
				require.Equal(t, ProblemInstanceURI(branch.ID), got.Instance)
				require.True(t, got.Temporary)
				require.Equal(t, new("branch retry"), got.RetryHint)
				return
			}
			require.Equal(t, http.StatusInternalServerError, got.Status)
			require.Equal(t, "fault", got.Code)
			require.NotEqual(t, ProblemInstanceURI(branch.ID), got.Instance)
			require.False(t, got.Temporary)
			require.False(t, got.Timeout)
			require.True(t, got.Fault)
			require.Nil(t, got.RetryHint)
			require.Equal(t, "internal server error", got.Detail)
		})
	}
	got := NewProblemResponse(t.Context(), outer, 409, "urn:outer", "Outer")
	require.Equal(t, "outer", got.Code)
	require.Equal(t, ProblemInstanceURI(outer.ID), got.Instance)
	require.Equal(t, 409, got.Status)
	require.Equal(t, "urn:outer", got.Type)
	require.Equal(t, "Outer", got.Title)
	require.Nil(t, got.RetryHint)
}
