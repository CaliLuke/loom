package http

import (
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewRuntimeCORSPolicy(t *testing.T) {
	policy, err := NewRuntimeCORSPolicy(CORSPolicy{Origins: []CORSOrigin{{
		Pattern: "https://app.example.com",
		Methods: []string{"GET"},
	}}})
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://app.example.com")
	policy.Handler(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusAccepted)
	})(recorder, req)
	require.Equal(t, stdhttp.StatusAccepted, recorder.Code)
	require.Equal(t, "https://app.example.com", recorder.Header().Get("Access-Control-Allow-Origin"))
}

func TestNewRuntimeCORSPolicyRejectsInvalidPolicies(t *testing.T) {
	tests := []struct {
		name   string
		policy CORSPolicy
		want   string
	}{
		{name: "empty", policy: CORSPolicy{}, want: "at least one origin"},
		{name: "empty origin", policy: CORSPolicy{Origins: []CORSOrigin{{}}}, want: "cannot be empty"},
		{name: "wildcard credentials", policy: CORSPolicy{Origins: []CORSOrigin{{Pattern: "*", Credentials: true}}}, want: "wildcard"},
		{name: "invalid regex", policy: CORSPolicy{Origins: []CORSOrigin{{Pattern: "[", Regex: true}}}, want: "invalid"},
		{name: "negative max age", policy: CORSPolicy{Origins: []CORSOrigin{{Pattern: "https://app.example.com", MaxAge: -1}}}, want: "negative"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewRuntimeCORSPolicy(test.policy)
			require.ErrorContains(t, err, test.want)
		})
	}
}

func TestRuntimeCORSPolicySnapshotsInput(t *testing.T) {
	origins := []CORSOrigin{{Pattern: "https://app.example.com", Methods: []string{"GET"}}}
	policy, err := NewRuntimeCORSPolicy(CORSPolicy{Origins: origins})
	require.NoError(t, err)
	origins[0].Pattern = "https://evil.example.com"
	origins[0].Methods[0] = "DELETE"

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(stdhttp.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	policy.HandlePreflight(recorder, req, []string{"GET"})
	require.Equal(t, "https://app.example.com", recorder.Header().Get("Access-Control-Allow-Origin"))
}

func TestRuntimeCORSPolicyRequestBehavior(t *testing.T) {
	policy, err := NewRuntimeCORSPolicy(CORSPolicy{Origins: []CORSOrigin{{
		Pattern:     "https://app.example.com",
		Methods:     []string{"POST"},
		Headers:     []string{"Authorization"},
		Credentials: true,
	}}})
	require.NoError(t, err)

	t.Run("allowed credentialed actual request", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(stdhttp.MethodPost, "/", nil)
		req.Header.Set("Origin", "https://app.example.com")
		policy.Handler(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
			w.WriteHeader(stdhttp.StatusOK)
		})(recorder, req)
		require.Equal(t, "https://app.example.com", recorder.Header().Get("Access-Control-Allow-Origin"))
		require.Equal(t, "true", recorder.Header().Get("Access-Control-Allow-Credentials"))
	})

	t.Run("disallowed actual request", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(stdhttp.MethodPost, "/", nil)
		req.Header.Set("Origin", "https://evil.example.com")
		policy.Handler(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
			w.WriteHeader(stdhttp.StatusOK)
		})(recorder, req)
		require.Empty(t, recorder.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("allowed preflight", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(stdhttp.MethodOptions, "/", nil)
		req.Header.Set("Origin", "https://app.example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		policy.HandlePreflight(recorder, req, []string{"POST"})
		require.Equal(t, stdhttp.StatusNoContent, recorder.Code)
		require.Equal(t, "POST", recorder.Header().Get("Access-Control-Allow-Methods"))
	})

	t.Run("disallowed preflight", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(stdhttp.MethodOptions, "/", nil)
		req.Header.Set("Origin", "https://app.example.com")
		req.Header.Set("Access-Control-Request-Method", "DELETE")
		policy.HandlePreflight(recorder, req, []string{"POST"})
		require.Equal(t, stdhttp.StatusNoContent, recorder.Code)
		require.Empty(t, recorder.Header().Get("Access-Control-Allow-Origin"))
	})
}

func TestRuntimeCORSPolicyOptionsHandlerDispatchesByPreflightHeader(t *testing.T) {
	policy, err := NewRuntimeCORSPolicy(CORSPolicy{Origins: []CORSOrigin{{
		Pattern: "https://app.example.com",
	}}})
	require.NoError(t, err)
	endpointCalls := 0
	handler := policy.OptionsHandler(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		endpointCalls++
		w.WriteHeader(stdhttp.StatusOK)
	}, []string{"GET", "OPTIONS"})

	ordinary := httptest.NewRequest(stdhttp.MethodOptions, "/items", nil)
	ordinary.Header.Set("Origin", "https://app.example.com")
	ordinaryResponse := httptest.NewRecorder()
	handler(ordinaryResponse, ordinary)
	require.Equal(t, stdhttp.StatusOK, ordinaryResponse.Code)
	require.Equal(t, 1, endpointCalls)
	require.Equal(t, "https://app.example.com", ordinaryResponse.Header().Get("Access-Control-Allow-Origin"))

	preflight := httptest.NewRequest(stdhttp.MethodOptions, "/items", nil)
	preflight.Header.Set("Origin", "https://app.example.com")
	preflight.Header.Set("Access-Control-Request-Method", "GET")
	preflightResponse := httptest.NewRecorder()
	handler(preflightResponse, preflight)
	require.Equal(t, stdhttp.StatusNoContent, preflightResponse.Code)
	require.Equal(t, 1, endpointCalls)
	require.Equal(t, "GET, OPTIONS", preflightResponse.Header().Get("Access-Control-Allow-Methods"))

	emptyPreflight := httptest.NewRequest(stdhttp.MethodOptions, "/items", nil)
	emptyPreflight.Header["Access-Control-Request-Method"] = []string{""}
	emptyPreflightResponse := httptest.NewRecorder()
	handler(emptyPreflightResponse, emptyPreflight)
	require.Equal(t, stdhttp.StatusNoContent, emptyPreflightResponse.Code)
	require.Equal(t, 1, endpointCalls)
}

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
