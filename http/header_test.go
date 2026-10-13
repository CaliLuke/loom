package http

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHeaderCollectionPresence(t *testing.T) {
	h := make(http.Header)
	require.Nil(t, HeaderValues(h, "x-tags"))
	require.NoError(t, SetHeaderValues(h, "x-tags", nil))
	require.Empty(t, h)
	require.NoError(t, SetHeaderValues(h, "x-tags", []string{}))
	require.Equal(t, []string{}, HeaderValues(h, "X-Tags"))
	require.Equal(t, []string{""}, h.Values("X-Tags"))
	require.NoError(t, SetHeaderValues(h, "x-tags", []string{"a", "b"}))
	require.Equal(t, []string{"a", "b"}, HeaderValues(h, "x-tags"))
	before := h.Clone()
	err := SetHeaderValues(h, "x-tags", []string{"secret,comma"})
	require.ErrorContains(t, err, `header "x-tags"`)
	require.NotContains(t, err.Error(), "secret")
	require.Equal(t, before, h)
}
