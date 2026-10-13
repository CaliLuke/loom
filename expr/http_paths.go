package expr

import (
	"fmt"
	"path"
	"strings"

	"github.com/dimfeld/httppath"
)

// serviceHTTPPaths is shared by early parameter inheritance and mount admission.
// Invalid graphs have no paths; admission reports the returned diagnostic.
func serviceHTTPPaths(svc *HTTPServiceExpr, visiting map[*HTTPServiceExpr]bool) ([]string, error) {
	if svc == nil || svc.Root == nil {
		return nil, fmt.Errorf("HTTP service has no route root")
	}
	if len(visiting) == 0 {
		if err := validateHTTPParentChain(svc); err != nil {
			return nil, err
		}
	}
	if visiting[svc] {
		return nil, fmt.Errorf("HTTP parent cycle at service %q", svc.Name())
	}
	visiting[svc] = true
	defer delete(visiting, svc)
	bases := []string{svc.Root.Path}
	if svc.ParentName != "" {
		parent := svc.Parent()
		canonical := parent.CanonicalEndpoint()
		var err error
		bases, err = routeHTTPPaths(canonical.Routes[0], visiting)
		if err != nil {
			return nil, err
		}
	}
	declared := svc.Paths
	if len(declared) == 0 {
		declared = []string{"/"}
	}
	var paths []string
	for _, p := range declared {
		if strings.HasPrefix(p, "//") {
			paths = append(paths, httppath.Clean(p))
			continue
		}
		for _, base := range bases {
			v := httppath.Clean(path.Join(base, p))
			if strings.HasSuffix(p, "/") && v != "/" {
				v += "/"
			}
			paths = append(paths, v)
		}
	}
	return paths, nil
}

func routeHTTPPaths(route *RouteExpr, visiting map[*HTTPServiceExpr]bool) ([]string, error) {
	if route == nil {
		return nil, fmt.Errorf("HTTP route is missing")
	}
	if route.IsAbsolute() {
		return []string{httppath.Clean(route.Path[1:])}, nil
	}
	if route.Endpoint == nil {
		return nil, fmt.Errorf("HTTP route %q has no endpoint", route.Path)
	}
	bases, err := serviceHTTPPaths(route.Endpoint.Service, visiting)
	if err != nil {
		return nil, err
	}
	paths := make([]string, len(bases))
	for i, base := range bases {
		paths[i] = httppath.Clean(path.Join(base, route.Path))
		if paths[i] == "/" {
			continue
		}
		if route.Path == "/" && strings.HasSuffix(base, "/") || route.Path != "/" && strings.HasSuffix(route.Path, "/") {
			paths[i] += "/"
		}
	}
	return paths, nil
}

func validateHTTPParentChain(service *HTTPServiceExpr) error {
	seen := make(map[*HTTPServiceExpr]bool)
	for service != nil {
		if seen[service] {
			return fmt.Errorf("HTTP parent cycle at service %q", service.Name())
		}
		seen[service] = true
		if service.ParentName == "" {
			return nil
		}
		parent := service.Parent()
		if parent == nil {
			return fmt.Errorf("Parent service %s not found", service.ParentName)
		}
		canonical := parent.CanonicalEndpoint()
		if canonical == nil || len(canonical.Routes) == 0 {
			return fmt.Errorf("Parent service %s has no canonical route", service.ParentName)
		}
		service = parent
	}
	return nil
}
