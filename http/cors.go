package http

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type (
	// CORSPolicy defines the Cross-Origin Resource Sharing headers generated
	// handlers may write for actual and preflight requests.
	CORSPolicy struct {
		Origins []CORSOrigin
	}

	// CORSOrigin defines one allowed origin and the response policy attached
	// to it. Pattern is an exact origin, "*", or a regular expression when
	// Regex is true.
	CORSOrigin struct {
		Pattern     string
		Regex       bool
		Methods     []string
		Headers     []string
		Expose      []string
		MaxAge      int
		Credentials bool
	}

	// RuntimeCORSPolicy is an immutable, validated startup snapshot of a CORS
	// policy supplied by application configuration.
	RuntimeCORSPolicy struct {
		policy CORSPolicy
	}
)

// corsRegexCache memoizes compiled, anchored origin regexps keyed by the raw
// user pattern. Generated code rebuilds CORSPolicy struct literals per request,
// so caching on the struct value would not persist; a package-level cache keeps
// compilation to once per distinct pattern across all requests.
var corsRegexCache sync.Map // map[string]*regexp.Regexp

// NewRuntimeCORSPolicy validates policy and returns an immutable startup
// snapshot suitable for generated servers declared with RuntimeCORS.
func NewRuntimeCORSPolicy(policy CORSPolicy) (RuntimeCORSPolicy, error) {
	if len(policy.Origins) == 0 {
		return RuntimeCORSPolicy{}, fmt.Errorf("CORS policy must define at least one origin")
	}
	snapshot := CORSPolicy{Origins: make([]CORSOrigin, len(policy.Origins))}
	for i, origin := range policy.Origins {
		if origin.Pattern == "" {
			return RuntimeCORSPolicy{}, fmt.Errorf("CORS origin %d cannot be empty", i)
		}
		if origin.Credentials && origin.Pattern == "*" {
			return RuntimeCORSPolicy{}, fmt.Errorf("CORS credentials are incompatible with wildcard origin")
		}
		if origin.Regex {
			if _, err := regexp.Compile(origin.Pattern); err != nil {
				return RuntimeCORSPolicy{}, fmt.Errorf("CORS origin regex %q is invalid: %w", origin.Pattern, err)
			}
		}
		if origin.MaxAge < 0 {
			return RuntimeCORSPolicy{}, fmt.Errorf("CORS max age cannot be negative")
		}
		origin.Methods = append([]string(nil), origin.Methods...)
		origin.Headers = append([]string(nil), origin.Headers...)
		origin.Expose = append([]string(nil), origin.Expose...)
		snapshot.Origins[i] = origin
	}
	return RuntimeCORSPolicy{policy: snapshot}, nil
}

// Handler wraps next and applies the runtime CORS policy to actual requests.
func (p RuntimeCORSPolicy) Handler(next http.HandlerFunc) http.HandlerFunc {
	return CORSHandler(p.policy, next)
}

// HandlePreflight writes a preflight response using the runtime CORS policy.
func (p RuntimeCORSPolicy) HandlePreflight(w http.ResponseWriter, r *http.Request, allowedMethods []string) {
	HandleCORSPreflight(w, r, p.policy, allowedMethods)
}

// CORSHandler wraps next and writes CORS response headers for matching actual
// browser requests. Non-CORS requests and disallowed origins pass through
// unchanged.
func CORSHandler(policy CORSPolicy, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		WriteCORSActualHeaders(w, r, policy)
		next(w, r)
	}
}

// WriteCORSActualHeaders writes CORS headers for a matching actual request.
func WriteCORSActualHeaders(w http.ResponseWriter, r *http.Request, policy CORSPolicy) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}
	match, ok := policy.match(origin)
	if !ok {
		return false
	}
	writeVary(w.Header(), "Origin")
	writeCORSOriginHeaders(w.Header(), origin, match)
	if len(match.Expose) > 0 {
		w.Header().Set("Access-Control-Expose-Headers", strings.Join(match.Expose, ", "))
	}
	return true
}

// HandleCORSPreflight writes a CORS preflight response for a matching origin.
// allowedMethods is the route-local method set generated for the matched path.
func HandleCORSPreflight(w http.ResponseWriter, r *http.Request, policy CORSPolicy, allowedMethods []string) {
	origin := r.Header.Get("Origin")
	requestMethod := r.Header.Get("Access-Control-Request-Method")
	match, ok := policy.match(origin)
	if origin == "" || requestMethod == "" || !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	methods := match.Methods
	if len(methods) == 0 {
		methods = allowedMethods
	}
	if len(methods) > 0 && !containsFold(methods, requestMethod) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	header := w.Header()
	writeVary(header, "Origin")
	writeVary(header, "Access-Control-Request-Method")
	writeVary(header, "Access-Control-Request-Headers")
	writeCORSOriginHeaders(header, origin, match)
	if len(methods) > 0 {
		header.Set("Access-Control-Allow-Methods", strings.Join(methods, ", "))
	}
	if len(match.Headers) > 0 {
		header.Set("Access-Control-Allow-Headers", strings.Join(match.Headers, ", "))
	} else if requested := r.Header.Get("Access-Control-Request-Headers"); requested != "" {
		header.Set("Access-Control-Allow-Headers", requested)
	}
	if match.MaxAge > 0 {
		header.Set("Access-Control-Max-Age", strconv.Itoa(match.MaxAge))
	}
	w.WriteHeader(http.StatusNoContent)
}

func (p CORSPolicy) match(origin string) (CORSOrigin, bool) {
	for _, allowed := range p.Origins {
		switch {
		case allowed.Pattern == "*":
			return allowed, true
		case allowed.Regex:
			if re := corsRegex(allowed.Pattern); re != nil && re.MatchString(origin) {
				return allowed, true
			}
		case allowed.Pattern == origin:
			return allowed, true
		}
	}
	return CORSOrigin{}, false
}

// corsRegex returns the compiled, full-string-anchored regexp for a CORS origin
// pattern. Origin patterns are wrapped as \A(?:pattern)\z so a partial match can
// never allow an unintended origin (e.g. "https://.*\.example\.com" must not
// match "https://api.example.com.evil.io"). Compiled expressions are cached by
// pattern so repeated requests do not recompile. It returns nil when the pattern
// fails to compile, in which case the origin is treated as not matching.
func corsRegex(pattern string) *regexp.Regexp {
	if cached, ok := corsRegexCache.Load(pattern); ok {
		return cached.(*regexp.Regexp)
	}
	re, err := regexp.Compile(`\A(?:` + pattern + `)\z`)
	if err != nil {
		re = nil
	}
	corsRegexCache.Store(pattern, re)
	return re
}

func writeCORSOriginHeaders(header http.Header, requestOrigin string, origin CORSOrigin) {
	if origin.Pattern == "*" && !origin.Credentials {
		header.Set("Access-Control-Allow-Origin", "*")
	} else {
		header.Set("Access-Control-Allow-Origin", requestOrigin)
	}
	if origin.Credentials {
		header.Set("Access-Control-Allow-Credentials", "true")
	}
}

func writeVary(header http.Header, value string) {
	existing := header.Values("Vary")
	for _, line := range existing {
		for _, part := range strings.Split(line, ",") {
			if strings.EqualFold(strings.TrimSpace(part), value) {
				return
			}
		}
	}
	header.Add("Vary", value)
}

func containsFold(values []string, want string) bool {
	for _, value := range values {
		if strings.EqualFold(value, want) {
			return true
		}
	}
	return false
}
