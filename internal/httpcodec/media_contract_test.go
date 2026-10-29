package httpcodec

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSelectMediaContract(t *testing.T) {
	contracts := []string{"*/*", "text/*", "text/plain", "application/json", "application/json; profile=v2"}
	for _, tc := range []struct {
		name, header string
		want         int
		rejected     bool
	}{
		{"exact", "text/plain; charset=utf-8", 2, false},
		{"subtype", "text/html", 1, false},
		{"fallback", "image/png", 0, false},
		{"json", "APPLICATION/JSON", 3, false},
		{"parameters", "application/json; profile=v2; charset=utf-8", 4, false},
		{"absent", "", 0, true},
		{"wildcard", "text/*", 0, true},
		{"malformed", "invalid", 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := SelectMediaContract(contracts, tc.header)
			if tc.rejected {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
	_, err := SelectMediaContract([]string{"text/plain"}, "application/json")
	require.Error(t, err)
	_, err = SelectMediaContract([]string{"text/plain; a=1", "text/plain; b=2"}, "text/plain; a=1; b=2")
	require.Error(t, err)
}

func TestValidateMediaContracts(t *testing.T) {
	for _, tc := range []struct {
		name      string
		contracts []string
		rejected  bool
	}{
		{"overlap ordered by specificity", []string{"*/*", "text/*", "text/plain"}, false},
		{"parameter specificity", []string{"application/json", "application/json; profile=v2"}, false},
		{"duplicates", []string{"text/plain", "TEXT/PLAIN"}, true},
		{"ambiguous parameters", []string{"text/plain; a=1", "text/plain; b=2"}, true},
		{"different parameters", []string{"text/plain; profile=v1", "text/plain; profile=v2"}, false},
		{"bad range", []string{"*/json"}, true},
		{"partial wildcard", []string{"application/*+json"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateMediaContracts(tc.contracts)
			if tc.rejected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMediaContractEmptyParameterRequiresPresence(t *testing.T) {
	_, err := SelectMediaContract([]string{`application/json; profile=""`}, "application/json")
	require.Error(t, err)
	index, err := SelectMediaContract([]string{`application/json; charset=UTF-8`}, "application/json; charset=utf-8")
	require.NoError(t, err)
	require.Zero(t, index)
}
