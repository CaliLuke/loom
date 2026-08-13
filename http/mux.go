package http

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sync"

	chi "github.com/go-chi/chi/v5"
)

type (
	// Muxer is the HTTP request multiplexer interface used by the generated
	// code. ServeHTTP must match the HTTP method and URL of each incoming
	// request against the list of registered patterns and call the handler
	// for the corresponding method and the pattern that most closely
	// matches the URL.
	//
	// The patterns may include wildcards that identify URL segments that
	// must be captured.
	//
	// There are two forms of wildcards the implementation must support:
	//
	//   - "{name}" wildcards capture a single path segment, for example the
	//     pattern "/images/{name}" captures "/images/favicon.ico" and adds
	//     the key "name" with the value "favicon.ico" to the map returned
	//     by Vars.
	//
	//   - "{*name}" wildcards must appear at the end of the pattern and
	//     captures the entire path starting where the wildcard matches. For
	//     example the pattern "/images/{*filename}" captures
	//     "/images/public/thumbnail.jpg" and associates the key key
	//     "filename" with "public/thumbnail.jpg" in the map returned by
	//     Vars.
	//
	// The names of wildcards must match the regular expression
	// "[a-zA-Z0-9_]+".
	Muxer interface {
		// Handle registers the handler function for the given method
		// and pattern.
		Handle(method, pattern string, handler http.HandlerFunc)

		// ServeHTTP dispatches the request to the handler whose method
		// matches the request method and whose pattern most closely
		// matches the request URL.
		ServeHTTP(http.ResponseWriter, *http.Request)

		// Vars returns the path variables captured for the given
		// request.
		Vars(*http.Request) map[string]string
	}

	// MiddlewareMuxer makes it possible to mount middlewares downstream of the
	// Muxer.
	MiddlewareMuxer interface {
		Muxer
		// Use appends a middleware to the list of middlewares to be applied
		// to the Muxer. Use must be called before Handle or before mounting a
		// generated server.
		Use(func(http.Handler) http.Handler)
	}

	// ResolverMuxer is a MiddlewareMuxer that can resolve the route pattern used
	// to register the handler for the given request.
	ResolverMuxer interface {
		MiddlewareMuxer
		ResolvePattern(*http.Request) string
	}

	// mux is the default Muxer implementation.
	mux struct {
		chi.Router
		// protect access to middlewares and handlers
		mu sync.Mutex
		// middlewares to be registered before handlers
		middlewares []func(http.Handler) http.Handler
		// routesRegistered reports whether Handle has mounted the middleware chain.
		routesRegistered bool
		// wildcards maps a method and a pattern to the name of the wildcard
		// this is needed because chi does not expose the name of the wildcard
		wildcards map[string]string
		// patternBeforeMiddleware reports whether ServeHTTP must pre-match the
		// route so mux-level middleware can read r.Pattern before chi dispatches
		// the matched handler.
		patternBeforeMiddleware bool
	}
)

// NewMuxer returns a Muxer implementation based on a Chi router.
//
// The returned muxer sets r.Pattern (Go 1.22+) on every matched request. When
// mux middleware is registered, r.Pattern is set before the middleware runs so
// observability middleware such as otelhttp can read the matched route for span
// attributes and metrics. To take advantage of this, register otelhttp as a mux
// middleware rather than wrapping the mux externally:
//
//	mux := loomhttp.NewMuxer()
//	mux.Use(otelhttp.NewMiddleware("service"))
func NewMuxer() ResolverMuxer {
	return &mux{
		Router:      chi.NewRouter(),
		middlewares: make([]func(http.Handler) http.Handler, 0),
		wildcards:   make(map[string]string),
	}
}

// wildPath matches a wildcard path segment.
var wildPath = regexp.MustCompile(`/{\*([a-zA-Z0-9_]+)}`)

// Handle registers the handler function for the given method and pattern.
// It sets r.Pattern on every matched request to "METHOD /path" (matching the Go
// 1.22+ convention used by http.ServeMux).
func (m *mux) Handle(method, pattern string, handler http.HandlerFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.routesRegistered {
		for _, middleware := range m.middlewares {
			m.Router.Use(middleware)
		}
		m.NotFound(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), AcceptTypeKey, req.Header.Get("Accept"))
			enc := ResponseEncoder(ctx, w)
			w.WriteHeader(http.StatusNotFound)
			enc.Encode(NewErrorResponse(ctx, fmt.Errorf("404 page not found"))) // nolint:errcheck
		}))
		m.routesRegistered = true
	}
	// Capture the registered pattern before wildcard rewriting so we can
	// populate r.Pattern for downstream consumers.
	reqPattern := method + " " + pattern
	if wildcards := wildPath.FindStringSubmatch(pattern); len(wildcards) > 0 {
		if len(wildcards) > 2 {
			panic("too many wildcards")
		}
		pattern = wildPath.ReplaceAllString(pattern, "/*")
		m.wildcards[method+"::"+pattern] = wildcards[1]
	}
	m.Method(method, pattern, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Pattern = reqPattern
		handler(w, r)
	}))
}

// ServeHTTP dispatches the request through chi's middleware chain and handler.
// When mux middleware is registered, it pre-resolves the matched route so the
// middleware can read r.Pattern before the matched handler runs.
func (m *mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if m.patternBeforeMiddleware {
		rctx := chi.NewRouteContext()
		if m.Match(rctx, r.Method, r.URL.Path) {
			r.Pattern = r.Method + " " + m.resolveWildcard(r.Method, rctx.RoutePattern())
		}
	}
	m.Router.ServeHTTP(w, r)
}

// Vars extracts the path variables from the request context.
func (m *mux) Vars(r *http.Request) map[string]string {
	ctx := m.ensureContext(r)
	if ctx == nil {
		return nil
	}
	params := ctx.URLParams
	if len(params.Keys) == 0 {
		return nil
	}
	vars := make(map[string]string, len(params.Keys))
	for i, k := range params.Keys {
		value := params.Values[i]
		if r.URL.RawPath != "" {
			value = unescapePathParam(value)
		}
		if k == "*" {
			wildcard := m.wildcards[r.Method+"::"+ctx.RoutePattern()]
			vars[wildcard] = value
			continue
		}
		vars[k] = value
	}
	return vars
}

func unescapePathParam(value string) string {
	unescaped, err := url.PathUnescape(value)
	if err != nil {
		return value
	}
	return unescaped
}

// Use appends a middleware to the list of middlewares applied downstream of
// the Muxer. It panics if a route has already been registered; call Use before
// mounting generated servers.
func (m *mux) Use(f func(http.Handler) http.Handler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.routesRegistered {
		panic("loom: register muxer middleware before mounting generated servers")
	}
	m.middlewares = append(m.middlewares, f)
	m.patternBeforeMiddleware = true
}

// ResolvePattern returns the route pattern used to register the handler for the
// given method and path.
func (m *mux) ResolvePattern(r *http.Request) string {
	ctx := m.ensureContext(r)
	if ctx == nil {
		return ""
	}
	return m.resolveWildcard(r.Method, ctx.RoutePattern())
}

// resolveWildcard returns the route pattern with the wildcard replaced by the
// name of the wildcard.
func (m *mux) resolveWildcard(method, pattern string) string {
	if wildcard, ok := m.wildcards[method+"::"+pattern]; ok {
		return pattern[:len(pattern)-2] + "/{*" + wildcard + "}"
	}
	return pattern
}

// ensureContext makes sure chi has initialized the request context if it
// handles it, otherwise it returns nil.
func (m *mux) ensureContext(r *http.Request) *chi.Context {
	ctx := chi.RouteContext(r.Context())
	if ctx == nil {
		return nil // request not handled by chi
	}
	if ctx.RoutePattern() != "" {
		return ctx // already initialized
	}
	if !m.Match(ctx, r.Method, r.URL.Path) {
		return nil // route not handled by chi
	}
	return ctx
}
