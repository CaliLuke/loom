package transportir

import (
	"strings"

	"github.com/CaliLuke/loom/expr"
)

func buildRoutes(endpoint *expr.HTTPEndpointExpr) []*Route {
	if endpoint == nil {
		return nil
	}
	var routes []*Route
	for index, route := range endpoint.Routes {
		for _, fullPath := range route.FullPaths() {
			routes = append(routes, &Route{
				Index:      index,
				Method:     strings.ToUpper(route.Method),
				Path:       fullPath,
				SourcePath: route.Path,
				Wildcards:  expr.ExtractHTTPWildcards(fullPath),
			})
		}
	}
	return routes
}
