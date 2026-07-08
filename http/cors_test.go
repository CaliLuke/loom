package http

import (
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCORSActualHeaders(t *testing.T) {
	policy := CORSPolicy{Origins: []CORSOrigin{{
		Pattern:     "https://app.example.com",
		Expose:      []string{"X-Request-Id"},
		Credentials: true,
	}}}
	req := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rec := httptest.NewRecorder()

	ok := WriteCORSActualHeaders(rec, req, policy)

	require.True(t, ok)
	require.Equal(t, "https://app.example.com", rec.Header().Get("Access-Control-Allow-Origin"))
	require.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
	require.Equal(t, "X-Request-Id", rec.Header().Get("Access-Control-Expose-Headers"))
	require.Equal(t, []string{"Origin"}, rec.Header().Values("Vary"))
}

func TestCORSPreflight(t *testing.T) {
	policy := CORSPolicy{Origins: []CORSOrigin{{
		Pattern: `https://.*\.example\.com`,
		Regex:   true,
		Headers: []string{"Authorization", "Content-Type"},
		MaxAge:  600,
	}}}
	req := httptest.NewRequest(stdhttp.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://api.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()

	HandleCORSPreflight(rec, req, policy, []string{"GET", "POST"})

	require.Equal(t, stdhttp.StatusNoContent, rec.Code)
	require.Equal(t, "https://api.example.com", rec.Header().Get("Access-Control-Allow-Origin"))
	require.Equal(t, "GET, POST", rec.Header().Get("Access-Control-Allow-Methods"))
	require.Equal(t, "Authorization, Content-Type", rec.Header().Get("Access-Control-Allow-Headers"))
	require.Equal(t, "600", rec.Header().Get("Access-Control-Max-Age"))
}

// TestCORSRegexAnchoring proves origin regexes match the full origin string so a
// partial match on an unanchored pattern cannot allow an attacker-controlled
// origin (issue #126).
func TestCORSRegexAnchoring(t *testing.T) {
	policy := CORSPolicy{Origins: []CORSOrigin{{
		Pattern: `https://.*\.example\.com`,
		Regex:   true,
	}}}

	tests := []struct {
		name   string
		origin string
		want   bool
	}{
		{name: "legitimate subdomain matches", origin: "https://api.example.com", want: true},
		{name: "suffix attack does not match", origin: "https://api.example.com.evil.io", want: false},
		{name: "trailing-path suffix attack does not match", origin: "https://api.example.com.evil.io/callback", want: false},
		{name: "prefix attack does not match", origin: "nothttps://api.example.com", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
			req.Header.Set("Origin", tt.origin)
			rec := httptest.NewRecorder()

			ok := WriteCORSActualHeaders(rec, req, policy)

			require.Equal(t, tt.want, ok)
			if tt.want {
				require.Equal(t, tt.origin, rec.Header().Get("Access-Control-Allow-Origin"))
			} else {
				require.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
			}
		})
	}
}

// TestCORSExactAndWildcard confirms exact (non-regex) origins and the "*"
// wildcard still behave correctly alongside the anchored regex matching.
func TestCORSExactAndWildcard(t *testing.T) {
	t.Run("exact origin matches only itself", func(t *testing.T) {
		policy := CORSPolicy{Origins: []CORSOrigin{{Pattern: "https://app.example.com"}}}

		reqOK := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
		reqOK.Header.Set("Origin", "https://app.example.com")
		recOK := httptest.NewRecorder()
		require.True(t, WriteCORSActualHeaders(recOK, reqOK, policy))
		require.Equal(t, "https://app.example.com", recOK.Header().Get("Access-Control-Allow-Origin"))

		reqBad := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
		reqBad.Header.Set("Origin", "https://app.example.com.evil.io")
		recBad := httptest.NewRecorder()
		require.False(t, WriteCORSActualHeaders(recBad, reqBad, policy))
		require.Empty(t, recBad.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("wildcard matches any origin", func(t *testing.T) {
		policy := CORSPolicy{Origins: []CORSOrigin{{Pattern: "*"}}}

		req := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
		req.Header.Set("Origin", "https://anything.example")
		rec := httptest.NewRecorder()
		require.True(t, WriteCORSActualHeaders(rec, req, policy))
		require.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
	})
}
