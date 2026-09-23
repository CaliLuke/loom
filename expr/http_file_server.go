package expr

import (
	"fmt"
	"path"
	"strings"

	"github.com/CaliLuke/loom/eval"
)

type (
	// HTTPFileServerExpr defines an endpoint that serves static assets
	// through HTTP.
	HTTPFileServerExpr struct {
		// Service is the parent service.
		Service *HTTPServiceExpr
		// Description for docs
		Description string
		// Docs points to the service external documentation
		Docs *DocsExpr
		// FilePath is the file path to the static asset(s)
		FilePath string
		// RequestPaths is the list of HTTP paths that serve the assets.
		RequestPaths []string
		// Redirect defines a redirect for the endpoint.
		Redirect *HTTPRedirectExpr
		// Meta is a list of key/value pairs
		Meta MetaExpr
	}
)

// EvalName returns the generic definition name used in error messages.
func (f *HTTPFileServerExpr) EvalName() string {
	suffix := fmt.Sprintf("file server %s", f.FilePath)
	var prefix string
	if f.Service != nil {
		prefix = f.Service.EvalName() + " "
	}
	return prefix + suffix
}

// Validate makes sure every full request path is one the muxer can register:
// a "{*name}" catch-all must terminate it, no wildcard may repeat, and no bare
// "*" may appear outside the "{*name}" syntax.
func (f *HTTPFileServerExpr) Validate() error {
	verr := new(eval.ValidationErrors)
	for _, p := range f.fullRequestPaths() {
		validateHTTPPathWildcards(verr, f, p)
	}
	if len(verr.Errors) == 0 {
		return nil
	}
	return verr
}

// Finalize normalizes the request path.
func (f *HTTPFileServerExpr) Finalize() {
	f.RequestPaths = f.fullRequestPaths()
}

// IsDir reports whether the request path uses a wildcard and therefore needs
// directory-style route mounting. The configured target may resolve to either
// a directory or a single file at runtime.
func (f *HTTPFileServerExpr) IsDir() bool {
	return HTTPWildcardRegex.MatchString(f.RequestPaths[0])
}

// fullRequestPaths joins the declared request path with the API and service
// base paths. A declared path that starts with "//" is absolute: it ignores
// the API and service prefixes and mounts at the server root.
func (f *HTTPFileServerExpr) fullRequestPaths() []string {
	current := f.RequestPaths[0]
	isAbs := strings.HasPrefix(current, "//")
	if isAbs {
		current = "/" + strings.TrimPrefix(current, "//")
	}

	paths := f.Service.Paths
	if len(paths) == 0 {
		paths = []string{"/"}
	}
	res := make([]string, len(paths))
	for i, sp := range paths {
		var p string
		if isAbs {
			p = current
		} else {
			p = path.Join(Root.API.HTTP.Path, sp, current)
		}
		// Make sure request path starts with a "/" so codegen can rely on it.
		if !strings.HasPrefix(p, "/") {
			p = "/" + p
		}
		res[i] = p
	}
	return res
}
