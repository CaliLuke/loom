package vet

import (
	"fmt"
	"strconv"

	"github.com/CaliLuke/loom/expr"
)

type (
	routeKey struct {
		method string
		path   string
	}

	manualRoute struct {
		method              string
		path                string
		location            Location
		duplicateSuppressed bool
		conflictSuppressed  bool
	}

	designRoute struct {
		operation string
		location  string
	}
)

func duplicateManualRouteDiagnostics(routes []manualRoute) []Diagnostic {
	groups := make(map[routeKey][]manualRoute)
	for _, route := range routes {
		key := routeKey{method: route.method, path: route.path}
		groups[key] = append(groups[key], route)
	}
	var diagnostics []Diagnostic
	for key, duplicates := range groups {
		if len(duplicates) < 2 {
			continue
		}
		for index, route := range duplicates {
			if route.duplicateSuppressed {
				continue
			}
			other := duplicates[(index+1)%len(duplicates)]
			diagnostics = append(diagnostics, Diagnostic{
				Rule:     RuleDuplicateRouteRegistration,
				Severity: SeverityError,
				Message: fmt.Sprintf(
					"manual route %s %s is registered more than once; other registration at %s",
					key.method,
					key.path,
					formatLocation(other.location),
				),
				Location: route.location,
			})
		}
	}
	return diagnostics
}

func designRouteConflictDiagnostics(routes []manualRoute, root *expr.RootExpr) []Diagnostic {
	designed := collectDesignRoutes(root)
	var diagnostics []Diagnostic
	for _, route := range routes {
		if route.conflictSuppressed {
			continue
		}
		key := routeKey{method: route.method, path: route.path}
		for _, conflict := range designed[key] {
			diagnostics = append(diagnostics, Diagnostic{
				Rule:     RuleRouteConflictWithDesign,
				Severity: SeverityError,
				Message: fmt.Sprintf(
					"manual route %s %s conflicts with designed operation %s (%s)",
					key.method,
					key.path,
					conflict.operation,
					conflict.location,
				),
				Location: route.location,
			})
		}
	}
	return diagnostics
}

func collectDesignRoutes(root *expr.RootExpr) map[routeKey][]designRoute {
	routes := make(map[routeKey][]designRoute)
	if root == nil || root.API == nil {
		return routes
	}
	transports := []*expr.HTTPExpr{root.API.HTTP}
	if root.API.JSONRPC != nil {
		transports = append(transports, &root.API.JSONRPC.HTTPExpr)
	}
	for _, transport := range transports {
		if transport == nil {
			continue
		}
		for _, service := range transport.Services {
			collectServiceDesignRoutes(routes, service)
		}
	}
	return routes
}

func collectServiceDesignRoutes(routes map[routeKey][]designRoute, service *expr.HTTPServiceExpr) {
	if service == nil || service.ServiceExpr == nil {
		return
	}
	for _, endpoint := range service.HTTPEndpoints {
		if endpoint == nil || endpoint.MethodExpr == nil {
			continue
		}
		operation := service.Name() + "." + endpoint.Name()
		location := "service." + service.Name() + ".method." + endpoint.Name()
		for _, route := range endpoint.Routes {
			if route == nil {
				continue
			}
			for _, path := range route.FullPaths() {
				key := routeKey{method: route.Method, path: path}
				routes[key] = append(routes[key], designRoute{operation: operation, location: location})
			}
		}
	}
}

func formatLocation(location Location) string {
	result := location.Path
	if location.Line > 0 {
		result += ":" + strconv.Itoa(location.Line)
	}
	if location.Column > 0 {
		result += ":" + strconv.Itoa(location.Column)
	}
	return result
}
