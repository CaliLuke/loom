package expr

import (
	"cmp"
	"fmt"
	pathpkg "path"
	"sort"
	"strings"

	"github.com/CaliLuke/loom/eval"
)

type (
	// HTTPMountPlan contains the physical registrations for one transport service.
	// Logical JSON-RPC methods share a dispatcher; preflights are separate mounts.
	HTTPMountPlan struct {
		// Routes contains endpoint, dispatcher and static-file registrations.
		Routes []*HTTPMountRoute
		// Preflights contains CORS OPTIONS registrations, including local overrides.
		Preflights []*HTTPMountRoute
		// SSE reports whether the JSON-RPC dispatcher serves streaming SSE methods.
		SSE bool
		// Mixed reports whether that dispatcher also serves ordinary JSON-RPC calls.
		Mixed bool
	}

	// HTTPMountRoute binds a physical method/path to its actual handler owner.
	HTTPMountRoute struct {
		// Method is the registered HTTP verb.
		Method string
		// Path is the effective registration path, retaining parameter names.
		Path string
		// Endpoint identifies an HTTP handler or an explicit local OPTIONS handler.
		Endpoint *HTTPEndpointExpr
		// FileServer identifies a static-file handler.
		FileServer *HTTPFileServerExpr
		// FilePrefix is stripped from a static-file request before filesystem lookup.
		FilePrefix string
		// Dispatcher reports that the service's JSON-RPC dispatcher owns this route.
		Dispatcher bool
		// Methods lists the CORS preflight's allowed physical verbs.
		Methods []string
	}
)

// MountPlan resolves the prepared service's physical routes for both design
// admission and code generation. Invalid graphs and conflicting local owners
// return diagnostics; callers must not emit a partial plan.
func (svc *HTTPServiceExpr) MountPlan() (*HTTPMountPlan, error) {
	if _, err := serviceHTTPPaths(svc, make(map[*HTTPServiceExpr]bool)); err != nil {
		return nil, err
	}
	plan := &HTTPMountPlan{}
	ordinary := false
	for _, endpoint := range svc.HTTPEndpoints {
		if endpoint.IsJSONRPC() {
			plan.SSE = plan.SSE || endpoint.MethodExpr.IsStreaming() && endpoint.SSE != nil
			ordinary = ordinary || !endpoint.MethodExpr.IsStreaming()
		}
	}
	plan.Mixed = plan.SSE && ordinary
	if err := plan.addEndpointRoutes(svc.HTTPEndpoints); err != nil {
		return nil, err
	}
	if err := plan.addFileRoutes(svc.FileServers); err != nil {
		return nil, err
	}
	if svc.CORS != nil || svc.Root.CORS != nil {
		plan.resolvePreflights()
	}
	return plan, nil
}

func httpRouteKey(method, path string) string {
	canonical := HTTPWildcardRegex.ReplaceAllStringFunc(path, func(wildcard string) string {
		if strings.HasPrefix(wildcard, "/{*") {
			return "/{*}"
		}
		return "/{}"
	})
	return method + " " + canonical
}

func sameHTTPMountOwner(left, right *HTTPMountRoute) bool {
	return left.Endpoint == right.Endpoint && left.FileServer == right.FileServer && left.Dispatcher == right.Dispatcher
}

func httpMountSource(route *HTTPMountRoute) string {
	switch {
	case route.Endpoint != nil:
		return route.Endpoint.EvalName()
	case route.FileServer != nil:
		return route.FileServer.EvalName()
	case route.Dispatcher:
		return "JSON-RPC dispatcher"
	default:
		return "CORS preflight"
	}
}

func (r *RootExpr) validateHTTPMounts() *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	if r.API == nil {
		return verr
	}
	type claim struct {
		service *HTTPServiceExpr
		route   *HTTPMountRoute
	}
	var claims []claim
	transports := []*HTTPExpr{r.API.HTTP}
	if r.API.JSONRPC != nil {
		transports = append(transports, &r.API.JSONRPC.HTTPExpr)
	}
	for _, transport := range transports {
		if transport == nil {
			continue
		}
		for _, service := range transport.Services {
			plan, err := service.MountPlan()
			if err != nil {
				verr.Add(service, "%s", err)
				continue
			}
			for _, route := range append(plan.Routes, plan.Preflights...) {
				// Explicit OPTIONS is emitted once through its preflight wrapper.
				if len(route.Methods) == 0 && route.Method == "OPTIONS" && len(plan.Preflights) > 0 {
					continue
				}
				claims = append(claims, claim{service: service, route: route})
			}
		}
	}
	sort.SliceStable(claims, func(i, j int) bool {
		left, right := claims[i], claims[j]
		return cmp.Or(
			cmp.Compare(left.service.Name(), right.service.Name()),
			cmp.Compare(httpRouteKey(left.route.Method, left.route.Path), httpRouteKey(right.route.Method, right.route.Path)),
			cmp.Compare(httpMountSource(left.route), httpMountSource(right.route)),
		) < 0
	})
	for i, left := range claims {
		for _, right := range claims[i+1:] {
			if !r.httpServicesCohosted(left.service.Name(), right.service.Name()) || httpRouteKey(left.route.Method, left.route.Path) != httpRouteKey(right.route.Method, right.route.Path) {
				continue
			}
			verr.Add(left.service, "HTTP route %s %s is owned by both %s (%s) and %s (%s)", left.route.Method, left.route.Path, left.service.Name(), httpMountSource(left.route), right.service.Name(), httpMountSource(right.route))
		}
	}
	return verr
}

func (r *RootExpr) httpServicesCohosted(left, right string) bool {
	if left == right || len(r.API.Servers) == 0 {
		return true
	}
	for _, server := range r.API.Servers {
		if len(server.Services) == 0 {
			return true
		}
		var hasLeft, hasRight bool
		for _, service := range server.Services {
			hasLeft = hasLeft || service == left
			hasRight = hasRight || service == right
		}
		if hasLeft && hasRight {
			return true
		}
	}
	return false
}

func (plan *HTTPMountPlan) addRoute(route *HTTPMountRoute) error {
	for _, previous := range plan.Routes {
		if httpRouteKey(previous.Method, previous.Path) != httpRouteKey(route.Method, route.Path) {
			continue
		}
		if previous.Path == route.Path && sameHTTPMountOwner(previous, route) {
			return nil
		}
		return fmt.Errorf("HTTP route %s %s conflicts between %s and %s (path %s)", route.Method, route.Path, httpMountSource(previous), httpMountSource(route), previous.Path)
	}
	plan.Routes = append(plan.Routes, route)
	return nil
}

func (plan *HTTPMountPlan) resolvePreflights() {
	byPath := make(map[string]*HTTPMountRoute)
	for _, route := range plan.Routes {
		if route.FileServer != nil {
			continue
		}
		key := httpRouteKey("OPTIONS", route.Path)
		preflight := byPath[key]
		if preflight == nil {
			preflight = &HTTPMountRoute{Method: "OPTIONS", Path: route.Path}
			byPath[key] = preflight
		}
		preflight.Methods = append(preflight.Methods, route.Method)
		if route.Method == "OPTIONS" {
			preflight.Endpoint = route.Endpoint
			preflight.Path = route.Path
		}
	}
	for _, route := range byPath {
		sort.Strings(route.Methods)
		plan.Preflights = append(plan.Preflights, route)
	}
	sort.Slice(plan.Preflights, func(i, j int) bool {
		return plan.Preflights[i].Path < plan.Preflights[j].Path
	})
}

func (plan *HTTPMountPlan) addEndpointRoutes(endpoints []*HTTPEndpointExpr) error {
	for _, endpoint := range endpoints {
		for _, route := range endpoint.Routes {
			paths, err := routeHTTPPaths(route, make(map[*HTTPServiceExpr]bool))
			if err != nil {
				return err
			}
			for _, path := range paths {
				mount := &HTTPMountRoute{Method: route.Method, Path: path, Endpoint: endpoint}
				if endpoint.IsJSONRPC() {
					mount.Endpoint = nil
					mount.Dispatcher = true
				}
				if err := plan.addRoute(mount); err != nil {
					return err
				}
				if mount.Dispatcher && (plan.Mixed || plan.SSE && endpoint.Name() == "events/stream") {
					if err := plan.addRoute(&HTTPMountRoute{Method: "GET", Path: path, Dispatcher: true}); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func (plan *HTTPMountPlan) addFileRoutes(servers []*HTTPFileServerExpr) error {
	for _, server := range servers {
		for _, path := range server.RequestPaths {
			paths := []string{path}
			prefix := pathpkg.Dir(path)
			if server.IsDir() {
				var found bool
				prefix, _, found = strings.CutLast(path, "/{")
				if !found {
					return fmt.Errorf("file route %q has no directory wildcard", path)
				}
				root := prefix + "/"
				paths = []string{root, root + "{*" + ExtractHTTPWildcards(path)[0] + "}"}
			}
			for _, path := range paths {
				if err := plan.addRoute(&HTTPMountRoute{Method: "GET", Path: path, FileServer: server, FilePrefix: prefix}); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
