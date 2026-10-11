package transportir

import "github.com/CaliLuke/loom/expr"

func buildRoutes(endpoint *expr.HTTPEndpointExpr) []*Route {
	if endpoint == nil {
		return nil
	}
	var routes []*Route
	operationIndex := 0
	for index, route := range endpoint.Routes {
		for _, fullPath := range route.FullPaths() {
			routes = append(routes, &Route{
				Index:          index,
				OperationIndex: operationIndex,
				Method:         route.Method,
				Path:           fullPath,
				SourcePath:     route.Path,
				Wildcards:      expr.ExtractHTTPWildcards(fullPath),
			})
			operationIndex++
		}
	}
	return routes
}

func RouteForExpr(endpoint *Endpoint, route *expr.RouteExpr, renderedPath string) *Route {
	if endpoint == nil || route == nil || len(endpoint.Routes) == 0 {
		return nil
	}
	routeIndex, sourcePath, ok := routeIdentity(route)
	if !ok {
		return nil
	}
	for _, routeIR := range endpoint.Routes {
		if routeIR.Method == route.Method && routeIR.matches(routeIndex, sourcePath, renderedPath) {
			return routeIR
		}
	}
	return nil
}

func routeIdentity(route *expr.RouteExpr) (int, string, bool) {
	if route == nil || route.Endpoint == nil {
		return 0, "", false
	}
	for index, current := range route.Endpoint.Routes {
		if current == route {
			return index, route.Path, true
		}
	}
	return 0, route.Path, false
}

func (r *Route) matches(index int, sourcePath string, renderedPath string) bool {
	if r == nil {
		return false
	}
	return r.Index == index && r.SourcePath == sourcePath && routePathMatches(r.Path, renderedPath)
}

func routePathMatches(path string, requested string) bool {
	return path == requested || expr.HTTPWildcardRegex.ReplaceAllString(path, "/{$1}") == requested
}
