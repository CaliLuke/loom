package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// adversarialPathValues lists path values that change the route or lose
// their meaning when they are formatted into a path without escaping.
var adversarialPathValues = []string{
	"plain",
	"a/b",
	"/leading",
	"trailing/",
	"100%",
	"%2F",
	"%zz",
	"a b",
	"a+b",
	"日本語",
	".",
	"..",
	"a/../b",
	"./a",
	"?q=1",
	"#frag",
	"a;b,c",
	"@:=&$",
	"a//b",
}

func TestEscapePathSegment(t *testing.T) {
	cases := []struct {
		Value, Expected string
	}{
		{"plain", "plain"},
		{"", ""},
		{"a/b", "a%2Fb"},
		{"100%", "100%25"},
		{"a b", "a%20b"},
		{"a+b", "a+b"},
		{"日本", "%E6%97%A5%E6%9C%AC"},
		{".", "%2E"},
		{"..", "%2E%2E"},
		{"...", "..."},
		{"a/../b", "a%2F..%2Fb"},
		{"?#", "%3F%23"},
		{"a;b,c", "a%3Bb%2Cc"},
	}
	for _, c := range cases {
		t.Run(c.Value, func(t *testing.T) {
			assert.Equal(t, c.Expected, EscapePathSegment(c.Value))
		})
	}
}

func TestEscapePathRemainder(t *testing.T) {
	cases := []struct {
		Value, Expected string
	}{
		{"plain", "plain"},
		{"", ""},
		{"a/b", "a/b"},
		{"a//b", "a//b"},
		{"/a/", "/a/"},
		{"a b/100%", "a%20b/100%25"},
		{"../etc/passwd", "%2E%2E/etc/passwd"},
		{"a/./b", "a/%2E/b"},
		{"a/?#", "a/%3F%23"},
	}
	for _, c := range cases {
		t.Run(c.Value, func(t *testing.T) {
			assert.Equal(t, c.Expected, EscapePathRemainder(c.Value))
		})
	}
}

func TestRequestURL(t *testing.T) {
	cases := []struct {
		Name, Path, WantPath, WantString string
	}{
		{"plain", "/items/plain", "/items/plain", "https://example.com/items/plain"},
		{"escaped slash", "/items/a%2Fb", "/items/a/b", "https://example.com/items/a%2Fb"},
		{"escaped percent", "/items/100%25", "/items/100%", "https://example.com/items/100%25"},
		{"escaped dots", "/items/%2E%2E", "/items/..", "https://example.com/items/%2E%2E"},
		{"double slash", "//a", "//a", "https://example.com//a"},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			u, err := RequestURL("https", "example.com", c.Path)
			require.NoError(t, err)
			assert.Equal(t, c.WantPath, u.Path)
			assert.Equal(t, c.Path, u.EscapedPath())
			assert.Equal(t, c.WantString, u.String())
		})
	}
}

func TestRequestURLRejectsInvalidEscapes(t *testing.T) {
	u, err := RequestURL("http", "example.com", "/items/%zz")
	require.Error(t, err)
	require.NotNil(t, u)
	assert.Equal(t, "example.com", u.Host)
	assert.Equal(t, "/items/%zz", u.Path)
}

// TestPathEscapingRoundTripsThroughMux checks that the muxer decodes to the
// original value what the path builders encode with EscapePathSegment and
// EscapePathRemainder and the clients send with RequestURL.
func TestPathEscapingRoundTripsThroughMux(t *testing.T) {
	for _, value := range adversarialPathValues {
		t.Run(value, func(t *testing.T) {
			var vars map[string]string
			mux := NewMuxer()
			mux.Handle(http.MethodGet, "/items/{id}/files/{*path}", func(_ http.ResponseWriter, r *http.Request) {
				vars = mux.Vars(r)
			})
			path := "/items/" + EscapePathSegment(value) + "/files/" + EscapePathRemainder(value)
			u, err := RequestURL("http", "example.com", path)
			require.NoError(t, err)
			req, err := http.NewRequest(http.MethodGet, u.String(), nil)
			require.NoError(t, err)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code, "request URL %s", u.String())
			assert.Equal(t, value, vars["id"])
			assert.Equal(t, value, vars["path"])
		})
	}
}
