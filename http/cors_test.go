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
		Pattern: `.*\.example\.com`,
		Regex:   true,
		Headers: []string{"Authorization", "Content-Type"},
		MaxAge:  600,
	}}}
	req := httptest.NewRequest(stdhttp.MethodOptions, "/", nil)
	req.Header.Set("Origin", "api.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()

	HandleCORSPreflight(rec, req, policy, []string{"GET", "POST"})

	require.Equal(t, stdhttp.StatusNoContent, rec.Code)
	require.Equal(t, "api.example.com", rec.Header().Get("Access-Control-Allow-Origin"))
	require.Equal(t, "GET, POST", rec.Header().Get("Access-Control-Allow-Methods"))
	require.Equal(t, "Authorization, Content-Type", rec.Header().Get("Access-Control-Allow-Headers"))
	require.Equal(t, "600", rec.Header().Get("Access-Control-Max-Age"))
}
