package openapimeta

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTagKeyRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		tagName string
		field   string
		want    string
	}{
		{name: "ordinary", tagName: "billing", want: "openapi:tag:billing"},
		{name: "empty ordinary", tagName: "", want: "openapi:tag:"},
		{name: "literal percent escape", tagName: "billing%3Adesc", want: "openapi:tag:billing%3Adesc"},
		{name: "colon suffix collision", tagName: "billing:desc", field: "desc", want: "openapi:tag-id:YmlsbGluZzpkZXNj:desc"},
		{name: "multiple colons", tagName: "ops:parent", field: "parent", want: "openapi:tag-id:b3BzOnBhcmVudA:parent"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			key := TagIdentityKey(test.tagName)
			if test.field != "" {
				key = TagAnnotationKey(test.tagName, test.field)
			}
			require.Equal(t, test.want, key)

			name, field, ok, err := ParseTagKey(key)
			require.NoError(t, err)
			require.True(t, ok)
			require.Equal(t, test.tagName, name)
			require.Equal(t, test.field, field)
		})
	}
}

func TestValidateTagMetadata(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string][]string
		wantErr  string
	}{
		{name: "identity has no value", metadata: map[string][]string{"openapi:tag:billing": {}}},
		{name: "annotation has one value", metadata: map[string][]string{"openapi:tag:billing:desc": {"description"}}},
		{name: "encoded tag can contain colon extension", metadata: map[string][]string{"openapi:tag-id:YmlsbGluZzpkZXNj:extension:x-a:b": {"true"}}},
		{name: "bad encoding", metadata: map[string][]string{"openapi:tag-id:%%%": {}}, wantErr: "malformed encoded"},
		{name: "noncanonical encoded ordinary name", metadata: map[string][]string{"openapi:tag:billing": {}, "openapi:tag-id:YmlsbGluZw": {}}, wantErr: "must be valid UTF-8 and contain a colon"},
		{name: "ambiguous legacy colon identity", metadata: map[string][]string{"openapi:tag:billing:desc:more": {}}, wantErr: "unknown OpenAPI tag metadata field"},
		{name: "empty annotation value list", metadata: map[string][]string{"openapi:tag:billing:desc": {}}, wantErr: "requires exactly one value"},
		{name: "multiple annotation values", metadata: map[string][]string{"openapi:tag:billing:desc": {"one", "two"}}, wantErr: "requires exactly one value"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateTagMetadata(test.metadata)
			if test.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}
