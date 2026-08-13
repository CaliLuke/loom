package openapiversion

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	tests := map[string]struct {
		value   string
		want    Target
		wantErr string
	}{
		"default":       {want: Target32},
		"whitespace":    {value: "  ", want: Target32},
		"3.1 alias":     {value: "3.1", want: Target31},
		"3.1.0":         {value: "3.1.0", want: Target31},
		"3.1.1":         {value: "3.1.1", want: Target31},
		"3.1.2 trimmed": {value: " 3.1.2 ", want: Target31},
		"3.2 alias":     {value: "3.2", want: Target32},
		"3.2.0":         {value: "3.2.0", want: Target32},
		"unsupported": {
			value:   "3.3",
			wantErr: `unsupported OpenAPI version "3.3"; supported values are 3.1, 3.1.0, 3.1.1, 3.1.2, 3.2, and 3.2.0`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := Parse(test.value)
			if test.wantErr != "" {
				require.EqualError(t, err, test.wantErr)
				require.Zero(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}
