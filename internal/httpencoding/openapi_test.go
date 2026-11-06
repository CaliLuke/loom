package httpencoding

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeOpenAPIParameter(t *testing.T) {
	flag := false
	for _, tc := range []struct {
		loc   Location
		style string
	}{{Path, "simple"}, {Header, "simple"}, {NamedQuery, "form"}, {Cookie, "form"}} {
		source := Serialization{Explode: &flag, AllowReserved: &flag}
		got := NormalizeOpenAPIParameter(tc.loc, source)
		require.Equal(t, tc.style, got.Style)
		require.Same(t, source.Explode, got.Explode)
		require.Same(t, source.AllowReserved, got.AllowReserved)
		source.Style = "deepObject"
		require.Equal(t, source, NormalizeOpenAPIParameter(tc.loc, source))
		source = Serialization{MediaType: "application/json", Explode: &flag}
		require.Equal(t, source, NormalizeOpenAPIParameter(tc.loc, source))
	}
}
