package encodingmeta

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRepresentationMetadata(t *testing.T) {
	for _, tc := range []struct {
		name        string
		meta        map[string][]string
		replacement string
		override    bool
	}{
		{"none", nil, "", false},
		{"named", map[string][]string{"openapi:typename": {"Blob"}, "name:original": {"Blob"}}, "", false},
		{"replacement", map[string][]string{"struct:field:type": {"other.Blob", "example.com/other"}}, "other.Blob", false},
		{"empty format", map[string][]string{"openapi:format": {""}}, "", true},
		{"empty list", map[string][]string{"openapi:format": {}}, "", false},
		{"encoding", map[string][]string{"openapi:contentEncoding": {"base64"}}, "", true},
		{"media", map[string][]string{"openapi:contentMediaType": {"image/png"}}, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.replacement, Replacement(tc.meta))
			require.Equal(t, tc.override, SchemaOverride(tc.meta))
		})
	}
}
