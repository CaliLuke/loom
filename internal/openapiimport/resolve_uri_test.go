package openapiimport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolverFragmentPointer(t *testing.T) {
	for _, tc := range []struct{ fragment, pointer string }{
		{"", ""}, {"/", "/"}, {"%2F", "/"}, {"/a%2Fb", "/a/b"},
		{"/a~1b", "/a~1b"}, {"/a%7E1b", "/a~1b"}, {"/a%7e1b", "/a~1b"},
		{"/%61", "/a"}, {"/~01", "/~01"}, {"/%252F", "/%2F"},
		{"/a%20b/%23/%25", "/a b/#/%"},
	} {
		t.Run(tc.fragment, func(t *testing.T) {
			pointer, err := decodePointer(tc.fragment)
			require.NoError(t, err)
			require.Equal(t, tc.pointer, pointer)
			again, err := decodePointer(emitPointer(pointer))
			require.NoError(t, err)
			require.Equal(t, pointer, again)
		})
	}
	for _, invalid := range []string{"anchor", "/~", "/~2", "/%Q0", "/%"} {
		_, err := decodePointer(invalid)
		require.Error(t, err, invalid)
	}
}

func TestResolverNormalization(t *testing.T) {
	for _, tc := range []struct{ value, want string }{
		{"HTTP://EXAMPLE.COM:80", "http://example.com/"},
		{"https://EXAMPLE.COM:443/a/./b/../c#top", "https://example.com/a/c"},
		{"https://e.test/%7e/%2f/%61?x=%7e#fragment", "https://e.test/~/%2F/a?x=~"},
		{"https://e.test/a/%2e%2e/b", "https://e.test/b"},
		{"https://e.test/a//b/", "https://e.test/a//b/"},
		{"https://e.test:8443/", "https://e.test:8443/"},
		{"file:///a/../b", "file:///b"},
	} {
		t.Run(tc.value, func(t *testing.T) {
			got, err := normalizeURI(tc.value)
			require.NoError(t, err)
			require.Equal(t, docURI(tc.want), got)
		})
	}
	for _, tc := range []struct{ reference, want string }{
		{"g", "http://a/b/c/g"}, {"./g", "http://a/b/c/g"},
		{"g/", "http://a/b/c/g/"}, {"/g", "http://a/g"},
		{"//g", "http://g/"}, {"?y", "http://a/b/c/d;p?y"},
		{"../g", "http://a/b/g"}, {"../../g", "http://a/g"},
		{"../../../g", "http://a/g"}, {"g/./h", "http://a/b/c/g/h"},
		{"g/../h", "http://a/b/c/h"}, {"g?y/../x", "http://a/b/c/g?y/../x"},
	} {
		got, pointer, err := resolveReference(tc.reference, "http://a/b/c/d;p?q")
		require.NoError(t, err, tc.reference)
		require.Equal(t, docURI(tc.want), got, tc.reference)
		require.Empty(t, pointer)
	}
	for _, invalid := range []string{"https://user:secret@e.test/x", "https://e.test/%QQ", "https://e.test/has space", "https://e.test/\\x"} {
		_, err := normalizeURI(invalid)
		require.Error(t, err)
		require.NotContains(t, err.Error(), "secret")
	}
}
