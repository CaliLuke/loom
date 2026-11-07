package jsonrpc

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	loomhttp "github.com/CaliLuke/loom/http"
)

func TestDecodeNullableErrorIdentity(t *testing.T) {
	for _, tc := range []struct {
		name         string
		data         string
		nullable     bool
		alternatives int
		calls        int
		selected     bool
	}{
		{"null", "null", true, 1, 1, true},
		{"whitespace", " \nnull\t", true, 1, 1, true},
		{"ambiguous null", "null", true, 2, 2, false},
		{"ordinary null", "null", false, 1, 0, false},
		{"missing", "", true, 1, 0, false},
		{"matching concrete", `{"name":"left"}`, true, 1, 1, true},
		{"conflicting concrete", `{"name":"other"}`, true, 1, 0, false},
		{"missing concrete selector", `{}`, true, 1, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			selected := errors.New("selected")
			calls := 0
			candidates := make([]ErrorCandidate, tc.alternatives)
			for i := range candidates {
				candidates[i] = ErrorCandidate{Name: "left", Nullable: tc.nullable, Selector: []string{"name"}, Decode: func(*http.Response) (error, error) {
					calls++
					return selected, nil
				}}
			}
			got := DecodeErrorResponse("svc", "method", &RawErrorResponse{Code: 4001, Data: []byte(tc.data)}, nil, candidates)
			require.Equal(t, tc.calls, calls)
			if tc.selected {
				require.Same(t, selected, got)
			} else {
				var invalid *loomhttp.ClientError
				require.ErrorAs(t, got, &invalid)
				if tc.calls > 1 {
					require.Contains(t, invalid.Message, "ambiguous")
				}
			}
		})
	}
}
