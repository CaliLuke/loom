package transportir

import "github.com/CaliLuke/loom/expr"

func buildRoutes(endpoint *expr.HTTPEndpointExpr) []*Route {
	if endpoint == nil {
		return nil
	}
	var routes []*Route
	for index, route := range endpoint.Routes {
		for _, fullPath := range route.FullPaths() {
			routes = append(routes, &Route{
				Index:      index,
				Method:     route.Method,
				Path:       fullPath,
				SourcePath: route.Path,
				Wildcards:  expr.ExtractHTTPWildcards(fullPath),
			})
		}
	}
	return routes
}

func RouteForExpr(endpoint *Endpoint, route *expr.RouteExpr, renderedPath string) *Route {
	if endpoint == nil || len(endpoint.Routes) == 0 {
		return nil
	}
	routeIndex, sourcePath := routeIdentity(route)
	for _, routeIR := range endpoint.Routes {
		if routeIR.matches(routeIndex, sourcePath, renderedPath) {
			return routeIR
		}
	}
	for _, routeIR := range endpoint.Routes {
		if routeIR.Path == renderedPath {
			return routeIR
		}
	}
	return endpoint.Routes[0]
}

func routeIdentity(route *expr.RouteExpr) (int, string) {
	if route == nil || route.Endpoint == nil {
		return 0, ""
	}
	for index, current := range route.Endpoint.Routes {
		if current == route {
			return index, route.Path
		}
	}
	return 0, route.Path
}

func (r *Route) matches(index int, sourcePath string, renderedPath string) bool {
	if r == nil {
		return false
	}
	if r.Index == index && r.SourcePath == sourcePath {
		return true
	}
	return r.Path == renderedPath
}
