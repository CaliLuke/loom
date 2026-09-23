package http

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type (
	// muxFuzzRoute is one generated route registration.
	muxFuzzRoute struct {
		method  string
		pattern string
	}

	// muxObservation records what the muxer exposed while serving a request.
	muxObservation struct {
		// Route is the registered "METHOD pattern" whose handler ran, or "".
		Route string
		// Pattern is r.Pattern as seen by the handler.
		Pattern string
		// Resolved is ResolvePattern as seen by the handler.
		Resolved string
		// Vars is Vars as seen by the handler.
		Vars map[string]string
		// MiddlewarePattern is r.Pattern as seen by mux middleware.
		MiddlewarePattern string
		// MiddlewareResolved is ResolvePattern as seen by mux middleware.
		MiddlewareResolved string
		// MiddlewareVars is Vars as seen by mux middleware.
		MiddlewareVars map[string]string
	}

	// muxMiddlewareMode selects which mux-level middleware is installed.
	muxMiddlewareMode int
)

const (
	muxNoMiddleware muxMiddlewareMode = iota
	muxPatternMiddleware
	muxResolvingMiddleware
)

var (
	muxFuzzSegments = []string{"a", "b", "ab", "{p}"}
	muxFuzzTokens   = []string{"a", "b", "ab", "z", "%2F", "%252F", "%20", "%40", "~", ".", "", "/", "/", "/"}
)

// FuzzMuxRouting registers generated route sets and checks that, for any
// request path, the route chi dispatches is the one an independent segment
// matcher selects, and that r.Pattern, ResolvePattern, and Vars agree with it
// in handlers and in mux middleware, including for escaped paths that set
// URL.RawPath.
func FuzzMuxRouting(f *testing.F) {
	f.Add([]byte{0, 3, 1, 2, 0}, []byte{0, 13, 3}, uint8(0))
	f.Add([]byte{1, 2, 0, 3, 1, 1, 3, 0}, []byte{0, 13, 4, 13, 1}, uint8(1))
	f.Add([]byte{2, 1, 3, 2, 3, 3, 0}, []byte{0, 13, 5, 13, 6}, uint8(2))
	f.Add([]byte{3, 0, 1, 1, 2, 3, 2, 0, 3, 1}, []byte{1, 13, 7, 8, 13, 13, 0}, uint8(4))
	f.Add([]byte{1, 3, 1, 3, 0}, []byte{0, 4, 13, 0}, uint8(2))

	f.Fuzz(func(t *testing.T, spec, pathSpec []byte, flags uint8) {
		routes := muxFuzzRoutes(spec)
		method := http.MethodGet
		if flags&1 == 1 {
			method = http.MethodPost
		}
		target := muxFuzzTarget(pathSpec)
		u, err := url.ParseRequestURI(target)
		require.NoError(t, err, "target %q", target)

		want := referenceMuxMatch(routes, method, u)
		base := serveMuxFuzz(t, routes, method, target, muxNoMiddleware)
		require.Equal(t, want.Route, base.Route, "dispatched route for %s %q (raw %q)", method, u.Path, u.RawPath)
		if want.Route != "" {
			require.Equal(t, want.Route, base.Pattern, "r.Pattern")
			require.Equal(t, strings.TrimPrefix(want.Route, method+" "), base.Resolved, "ResolvePattern")
			require.Equal(t, want.Vars, base.Vars, "Vars")
		}

		for _, mode := range []muxMiddlewareMode{muxPatternMiddleware, muxResolvingMiddleware} {
			got := serveMuxFuzz(t, routes, method, target, mode)
			require.Equal(t, base.Route, got.Route, "middleware mode %d changed dispatch", mode)
			require.Equal(t, base.Pattern, got.Pattern, "middleware mode %d changed r.Pattern", mode)
			require.Equal(t, base.Resolved, got.Resolved, "middleware mode %d changed ResolvePattern", mode)
			require.Equal(t, base.Vars, got.Vars, "middleware mode %d changed Vars", mode)
			require.Equal(t, want.Route, got.MiddlewarePattern, "middleware mode %d r.Pattern", mode)
			if mode == muxResolvingMiddleware {
				require.Equal(t, strings.TrimPrefix(want.Route, method+" "), got.MiddlewareResolved, "middleware ResolvePattern")
				require.Equal(t, want.Vars, got.MiddlewareVars, "middleware Vars")
			}
		}
	})
}

// muxFuzzRoutes decodes up to four distinct routes from spec. Each route has
// up to three segments drawn from muxFuzzSegments and an optional trailing
// named wildcard.
func muxFuzzRoutes(spec []byte) []muxFuzzRoute {
	next := func() int {
		if len(spec) == 0 {
			return 0
		}
		b := int(spec[0])
		spec = spec[1:]
		return b
	}
	count := next()%4 + 1
	seen := make(map[muxFuzzRoute]bool)
	routes := make([]muxFuzzRoute, 0, count)
	for range count {
		header := next()
		method := http.MethodGet
		if header&1 == 1 {
			method = http.MethodPost
		}
		segments := make([]string, 0, 4)
		for i := range (header >> 1) % 4 {
			segment := muxFuzzSegments[next()%len(muxFuzzSegments)]
			if segment == "{p}" {
				segment = fmt.Sprintf("{p%d}", i)
			}
			segments = append(segments, segment)
		}
		if (header>>3)%3 == 0 {
			segments = append(segments, "{*w}")
		}
		route := muxFuzzRoute{method: method, pattern: "/" + strings.Join(segments, "/")}
		if seen[route] {
			continue
		}
		seen[route] = true
		routes = append(routes, route)
	}
	return routes
}

func muxFuzzTarget(pathSpec []byte) string {
	var b strings.Builder
	b.WriteByte('/')
	for _, c := range pathSpec {
		b.WriteString(muxFuzzTokens[int(c)%len(muxFuzzTokens)])
	}
	return b.String()
}

func serveMuxFuzz(t *testing.T, routes []muxFuzzRoute, method, target string, mode muxMiddlewareMode) muxObservation {
	t.Helper()
	var obs muxObservation
	mux := NewMuxer()
	switch mode {
	case muxPatternMiddleware:
		mux.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				obs.MiddlewarePattern = r.Pattern
				next.ServeHTTP(w, r)
			})
		})
	case muxResolvingMiddleware:
		mux.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				obs.MiddlewarePattern = r.Pattern
				obs.MiddlewareResolved = mux.ResolvePattern(r)
				obs.MiddlewareVars = mux.Vars(r)
				next.ServeHTTP(w, r)
			})
		})
	case muxNoMiddleware:
	}
	for _, route := range routes {
		registered := route.method + " " + route.pattern
		mux.Handle(route.method, route.pattern, func(_ http.ResponseWriter, r *http.Request) {
			obs.Route = registered
			obs.Pattern = r.Pattern
			obs.Resolved = mux.ResolvePattern(r)
			obs.Vars = mux.Vars(r)
		})
	}
	mux.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(method, target, nil))
	return obs
}

// referenceMuxMatch selects the route chi should dispatch to for u. Routing
// uses URL.RawPath when set, otherwise URL.Path. Segments are compared left to
// right with backtracking, preferring a literal segment, then a {param}, then
// a trailing {*wildcard}. A {param} matches one segment without '/'; it may
// be empty except as the final segment of the path. A {*wildcard} matches the
// remaining path, including an empty one. Captured values are unescaped when
// URL.RawPath is set.
func referenceMuxMatch(routes []muxFuzzRoute, method string, u *url.URL) muxObservation {
	routePath := u.Path
	if u.RawPath != "" {
		routePath = u.RawPath
	}
	var (
		best     muxObservation
		bestRank []int
	)
	for _, route := range routes {
		if route.method != method {
			continue
		}
		vars, rank, ok := referenceMuxRoute(route.pattern, routePath)
		if !ok {
			continue
		}
		if bestRank != nil && !muxRankLess(rank, bestRank) {
			continue
		}
		bestRank = rank
		if u.RawPath != "" {
			for k, v := range vars {
				if unescaped, err := url.PathUnescape(v); err == nil {
					vars[k] = unescaped
				}
			}
		}
		if len(vars) == 0 {
			vars = nil
		}
		best = muxObservation{Route: method + " " + route.pattern, Vars: vars}
	}
	return best
}

// referenceMuxRoute matches pattern against path and returns the captured
// values and the per-segment kinds (0 literal, 1 param, 2 wildcard) used to
// rank competing matches.
func referenceMuxRoute(pattern, path string) (map[string]string, []int, bool) {
	patternSegments := strings.Split(strings.TrimPrefix(pattern, "/"), "/")
	if pattern == "/" {
		patternSegments = nil
	}
	if !strings.HasPrefix(path, "/") {
		return nil, nil, false
	}
	rest := path[1:]
	vars := make(map[string]string)
	rank := make([]int, 0, len(patternSegments))
	for i, segment := range patternSegments {
		last := i == len(patternSegments)-1
		switch {
		case strings.HasPrefix(segment, "{*"):
			vars[strings.TrimSuffix(strings.TrimPrefix(segment, "{*"), "}")] = rest
			return vars, append(rank, 2), true
		case strings.HasPrefix(segment, "{"):
			value, remainder, more := strings.Cut(rest, "/")
			if last && (more || value == "") {
				return nil, nil, false
			}
			if !last && !more {
				return nil, nil, false
			}
			vars[strings.TrimSuffix(strings.TrimPrefix(segment, "{"), "}")] = value
			rank = append(rank, 1)
			rest = remainder
		default:
			value, remainder, more := strings.Cut(rest, "/")
			if value != segment || last == more {
				return nil, nil, false
			}
			rank = append(rank, 0)
			rest = remainder
		}
	}
	if len(patternSegments) == 0 && rest != "" {
		return nil, nil, false
	}
	return vars, rank, true
}

func muxRankLess(a, b []int) bool {
	for i := range min(len(a), len(b)) {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return len(a) < len(b)
}
