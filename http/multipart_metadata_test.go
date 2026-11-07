package http

import (
	"encoding/json/jsontext"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMultipartDerivedMetadataPrecedence(t *testing.T) {
	for _, tc := range []struct {
		name     string
		explicit bool
		count    int
		want     bool
	}{
		{"one file", false, 1, true}, {"explicit empty wins", true, 1, false},
		{"duplicate files", false, 2, false}, {"no file", false, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			form := &MultipartForm{}
			for range tc.count {
				form.Parts = append(form.Parts, MultipartPart{Name: "file", Filename: "name.txt", ContentType: "text/plain"})
			}
			if tc.explicit {
				form.Parts = append(form.Parts, MultipartPart{Name: "filename", Data: []byte{}})
			}
			raw, present, err := MultipartMetadataJSON(form, "filename", "file", "filename")
			require.NoError(t, err)
			require.Equal(t, tc.want, present)
			if present {
				require.Equal(t, `"name.txt"`, string(raw))
			}
			projection, matched, err := MultipartObjectJSON(form, []FormField{
				{Name: "file", Keys: FormKeySet{Literals: []string{"file"}}, View: func() (jsontext.Value, bool, error) { return nil, true, nil }},
				{Name: "filename", Keys: FormKeySet{Literals: []string{"filename"}}, View: func() (jsontext.Value, bool, error) { return raw, true, nil }},
			})
			require.NoError(t, err)
			require.True(t, matched)
			if !tc.explicit {
				require.Empty(t, projection.FieldClaims["filename"])
			}
		})
	}
}
