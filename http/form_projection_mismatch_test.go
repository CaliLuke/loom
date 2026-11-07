package http

import (
	"encoding/json/jsontext"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormProjectionKeepsPhysicalClaimsOnMismatch(t *testing.T) {
	for _, failure := range []string{"mismatch", "invalid JSON", "view error"} {
		var viewError error
		if failure == "view error" {
			viewError = errors.New("projection failed")
		}
		view := func() (jsontext.Value, bool, error) {
			if failure == "invalid JSON" {
				return jsontext.Value("{"), true, nil
			}
			return nil, false, viewError
		}
		pairs, err := ParseRawPairs("x=bad&x=again&extra=unknown")
		require.NoError(t, err)
		projection, match, err := FormObjectJSON(pairs, []FormField{{Name: "logical", Keys: FormKeySet{Literals: []string{"x"}}, View: view}})
		require.ErrorIs(t, err, viewError)
		require.False(t, match)
		require.Nil(t, projection.Value)
		require.Equal(t, []string{"extra", "x"}, projection.Members)
		require.Equal(t, []string{"extra"}, projection.Unclaimed)
		require.Equal(t, []string{"x"}, projection.FieldClaims["logical"])
		// The typed domain can accept a value whose schema projection fails. Its
		// closure check must still see all physical fields, independent of that failure.
		for _, closed := range []bool{false, true} {
			decoded, err := projection.Schema(func(jsontext.Value) (bool, error) {
				return true, nil
			}, closed)
			require.NoError(t, err)
			require.Equal(t, !closed, decoded)
		}

		calls := 0
		mapped, match, err := FormMapJSON(pairs, func(string) (jsontext.Value, bool, error) {
			calls++
			return view()
		})
		require.ErrorIs(t, err, viewError)
		require.False(t, match)
		require.Nil(t, mapped.Value)
		require.Equal(t, 1, calls)
		require.Equal(t, []string{"extra", "x"}, mapped.Members)
		require.Empty(t, mapped.Unclaimed)
		require.Equal(t, map[string][]string{"x": {"x"}, "extra": {"extra"}}, mapped.FieldClaims)
	}
}
