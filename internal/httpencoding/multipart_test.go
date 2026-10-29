package httpencoding

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMultipartContentTypeRequiresConcreteMedia(t *testing.T) {
	for _, tc := range []struct {
		media string
		valid bool
	}{
		{"image/png", true},
		{`image/png; profile="*"`, true},
		{"image/*", false},
		{"*/*", false},
		{"image", false},
	} {
		t.Run(tc.media, func(t *testing.T) {
			_, err := MultipartContentType(Scalar, true, tc.media)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "concrete")
			}
		})
	}
}
