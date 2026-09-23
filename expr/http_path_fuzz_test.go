package expr

import (
	"slices"
	"strings"
	"testing"
)

// httpPathFuzzSeeds are realistic route and base paths, including the
// malformed wildcard shapes designs sometimes contain.
var httpPathFuzzSeeds = []string{
	"",
	"/",
	"/users",
	"/users/",
	"/users/{id}",
	"/users/{userID}/posts/{post_id}",
	"/files/{*path}",
	"/{*path}/tail",
	"/{a}/{a}",
	"/{id",
	"/id}",
	"/{}",
	"/{*}",
	"/{a-b}",
	"/foo{id}",
	"/{id}.json",
	"//absolute/{id}",
	"///",
	"/a/./b/../c",
	"/{日本}",
	"/\xff/{x}",
}

// FuzzExtractHTTPWildcards compares ExtractHTTPWildcards with an independent
// scanner implementing the documented wildcard grammar: a "/" followed by
// "{", an optional "*", a non-empty run of [a-zA-Z0-9_], and "}".
func FuzzExtractHTTPWildcards(f *testing.F) {
	for _, s := range httpPathFuzzSeeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, path string) {
		got := ExtractHTTPWildcards(path)
		want := referenceHTTPWildcards(path)
		if !slices.Equal(got, want) {
			t.Errorf("ExtractHTTPWildcards(%q) = %q, want %q", path, got, want)
		}
	})
}

// FuzzRouteFullPaths checks that joining API, service, and route paths yields
// normalized absolute paths that keep the route's trailing slash semantics
// and its wildcards.
func FuzzRouteFullPaths(f *testing.F) {
	for _, s := range httpPathFuzzSeeds {
		f.Add("/api", "/svc", s)
		f.Add("", "", s)
		f.Add("/", s, "/{id}")
	}
	f.Fuzz(func(t *testing.T, apiPath, servicePath, routePath string) {
		root := &HTTPExpr{Path: apiPath}
		svc := &HTTPServiceExpr{ServiceExpr: &ServiceExpr{Name: "svc"}, Root: root}
		if servicePath != "" {
			svc.Paths = []string{servicePath}
		}
		root.Services = []*HTTPServiceExpr{svc}
		route := &RouteExpr{Method: "GET", Path: routePath, Endpoint: &HTTPEndpointExpr{Service: svc}}
		for _, full := range route.FullPaths() {
			if !strings.HasPrefix(full, "/") {
				t.Errorf("FullPaths(%q, %q, %q) = %q is not absolute", apiPath, servicePath, routePath, full)
			}
			if strings.Contains(full, "//") {
				t.Errorf("FullPaths(%q, %q, %q) = %q contains an empty segment", apiPath, servicePath, routePath, full)
			}
			for _, segment := range strings.Split(full, "/") {
				if segment == "." || segment == ".." {
					t.Errorf("FullPaths(%q, %q, %q) = %q contains a dot segment", apiPath, servicePath, routePath, full)
				}
			}
			if full != "/" && routePath != "/" && routePath != "" && !hasDotSegment(routePath) {
				if strings.HasSuffix(routePath, "/") != strings.HasSuffix(full, "/") {
					t.Errorf("FullPaths(%q, %q, %q) = %q does not preserve the route trailing slash", apiPath, servicePath, routePath, full)
				}
			}
			if !hasDotSegment(routePath) {
				for _, wc := range ExtractHTTPWildcards(routePath) {
					if !slices.Contains(ExtractHTTPWildcards(full), wc) {
						t.Errorf("FullPaths(%q, %q, %q) = %q drops wildcard %q", apiPath, servicePath, routePath, full, wc)
					}
				}
			}
		}
	})
}

// FuzzRouteCatchAllValidation checks that route validation accepts a
// catch-all wildcard only when it terminates every full path.
func FuzzRouteCatchAllValidation(f *testing.F) {
	for _, s := range httpPathFuzzSeeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, routePath string) {
		root := &HTTPExpr{}
		svc := &HTTPServiceExpr{ServiceExpr: &ServiceExpr{Name: "svc"}, Root: root}
		root.Services = []*HTTPServiceExpr{svc}
		endpoint := &HTTPEndpointExpr{Service: svc, MethodExpr: &MethodExpr{Name: "m", Service: svc.ServiceExpr}}
		route := &RouteExpr{Method: "GET", Path: routePath, Endpoint: endpoint}
		verr := route.Validate()
		reported := strings.Contains(verr.Error(), "Catch-all wildcard")
		want := false
		for _, full := range route.FullPaths() {
			if end := referenceCatchAllIndex(full); end >= 0 && end != len(full) {
				want = true
			}
		}
		if reported != want {
			t.Errorf("Validate(%q) catch-all error = %v, want %v: %v", routePath, reported, want, verr)
		}
	})
}

// referenceHTTPWildcards scans path left to right for non-overlapping
// wildcards.
func referenceHTTPWildcards(path string) []string {
	res := []string{}
	for i := 0; i < len(path); {
		name, end := scanHTTPWildcard(path, i)
		if end < 0 {
			i++
			continue
		}
		res = append(res, name)
		i = end
	}
	return res
}

// referenceCatchAllIndex returns the end offset of the first non-terminal
// catch-all wildcard in path, len(path) when every catch-all terminates it,
// or -1 when path has no catch-all wildcard.
func referenceCatchAllIndex(path string) int {
	found := -1
	for i := 0; i < len(path); {
		_, end := scanHTTPWildcard(path, i)
		if end < 0 {
			i++
			continue
		}
		if strings.HasPrefix(path[i:], "/{*") {
			if end != len(path) {
				return end
			}
			found = end
		}
		i = end
	}
	return found
}

// scanHTTPWildcard parses a wildcard starting at offset i and returns its
// name and end offset, or -1 when no wildcard starts at i.
func scanHTTPWildcard(path string, i int) (string, int) {
	if !strings.HasPrefix(path[i:], "/{") {
		return "", -1
	}
	j := i + 2
	if j < len(path) && path[j] == '*' {
		j++
	}
	start := j
	for j < len(path) && isWildcardNameByte(path[j]) {
		j++
	}
	if j == start || j >= len(path) || path[j] != '}' {
		return "", -1
	}
	return path[start:j], j + 1
}

// hasDotSegment reports whether path contains a "." or ".." segment. Path
// cleaning resolves those segments, which may remove wildcards or add a
// directory trailing slash, so the trailing-slash and wildcard properties do
// not apply to them.
func hasDotSegment(path string) bool {
	for _, segment := range strings.Split(path, "/") {
		if segment == "." || segment == ".." {
			return true
		}
	}
	return false
}

func isWildcardNameByte(c byte) bool {
	return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' || c == '_'
}
