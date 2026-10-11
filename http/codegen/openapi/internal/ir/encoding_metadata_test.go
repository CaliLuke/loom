package ir

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodingMetadataFamilies(t *testing.T) {
	for _, tc := range []struct {
		name    string
		meta    map[string][]string
		invalid bool
	}{
		{"empty", nil, false},
		{"named", map[string][]string{"openapi:encoding:id:style": {"form"}}, false},
		{"positional pair", map[string][]string{
			"openapi:prefixEncoding:0:style": {"form"}, "openapi:itemEncoding:style": {"form"},
		}, false},
		{"root conflict", map[string][]string{
			"openapi:encoding:id:style": {"form"}, "openapi:itemEncoding:style": {"form"},
		}, true},
		{"nested conflict with false leaf", map[string][]string{
			"openapi:encoding:id:encoding:name:explode": {"false"}, "openapi:encoding:id:itemEncoding:explode": {"false"},
		}, true},
		{"equivalent index spellings", map[string][]string{
			"openapi:prefixEncoding:00:encoding:id:style": {"form"}, "openapi:prefixEncoding:0:itemEncoding:style": {"form"},
		}, true},
		{"distinct prefix objects", map[string][]string{
			"openapi:prefixEncoding:0:encoding:id:style": {"form"}, "openapi:prefixEncoding:1:itemEncoding:style": {"form"},
		}, false},
		{"empty annotation emits nothing", map[string][]string{
			"openapi:encoding:id:style": {"form"}, "openapi:itemEncoding:style": nil,
		}, false},
		{"invalid prefix emits nothing", map[string][]string{
			"openapi:encoding:id:style": {"form"}, "openapi:prefixEncoding:-1:style": {"form"},
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateEncodingMetadata(tc.meta)
			if tc.invalid {
				require.ErrorContains(t, err, "encoding cannot coexist with prefixEncoding or itemEncoding")
			} else {
				require.NoError(t, err)
			}
		})
	}
}
