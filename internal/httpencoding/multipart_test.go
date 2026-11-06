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

func TestMultipartSerialization(t *testing.T) {
	no, yes := false, true
	for _, tc := range []struct {
		name     string
		shape    Shape
		source   Serialization
		accepted bool
	}{
		{"plain scalar", Scalar, Serialization{}, true},
		{"scalar form", Scalar, Serialization{Style: "form", Explode: &no}, true},
		{"plain list", Sequence, Serialization{}, true},
		{"repeated list", Sequence, Serialization{Style: "form", Explode: &yes}, true},
		{"comma list", Sequence, Serialization{Style: "form", Explode: &no}, false},
		{"plain object", Object, Serialization{}, true},
		{"plain map", ScalarMap, Serialization{}, true},
		{"flattened object", Object, Serialization{Style: "form"}, false},
		{"flag object", Object, Serialization{AllowReserved: &no}, false},
		{"nested JSON", Nested, Serialization{}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := MatchMultipart(BranchShape{Shape: tc.shape}, tc.source)
			if tc.accepted {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
