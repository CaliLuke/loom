package dsl

import (
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

// GET defines a route using the GET HTTP method. The route may use wildcards to
// define path parameters. Wildcards start with '{' or with '{*' and end with
// '}'. They must appear after a '/'.
//
// A wildcard that starts with '{' matches a section of the path (the value in
// between two slashes).
//
// A wildcard that starts with '{*' matches the rest of the path. Such wildcards
// must terminate the path.
//
// GET must appear in a method HTTP function.
//
// GET accepts one argument which is the request path.
//
// Example:
//
//	var _ = Service("Manager", func() {
//	    Method("GetAccount", func() {
//	        Payload(GetAccount)
//	        Result(Account)
//	        HTTP(func() {
//	            GET("/{accountID}/details")
//	            GET("/{*accountPath}")
//	        })
//	    })
//	})
func GET(path string) *expr.RouteExpr {
	return route("GET", path)
}

// HEAD creates a route using the HEAD HTTP method. See GET.
func HEAD(path string) *expr.RouteExpr {
	return route("HEAD", path)
}

// POST creates a route using the POST HTTP method. See GET.
func POST(path string) *expr.RouteExpr {
	return route("POST", path)
}

// PUT creates a route using the PUT HTTP method. See GET.
func PUT(path string) *expr.RouteExpr {
	return route("PUT", path)
}

// DELETE creates a route using the DELETE HTTP method. See GET.
func DELETE(path string) *expr.RouteExpr {
	return route("DELETE", path)
}

// OPTIONS creates a route using the OPTIONS HTTP method. See GET.
func OPTIONS(path string) *expr.RouteExpr {
	return route("OPTIONS", path)
}

// TRACE creates a route using the TRACE HTTP method. See GET.
func TRACE(path string) *expr.RouteExpr {
	return route("TRACE", path)
}

// CONNECT creates a route using the CONNECT HTTP method. See GET.
func CONNECT(path string) *expr.RouteExpr {
	return route("CONNECT", path)
}

// PATCH creates a route using the PATCH HTTP method. See GET.
func PATCH(path string) *expr.RouteExpr {
	return route("PATCH", path)
}

func route(method, path string) *expr.RouteExpr {
	r := &expr.RouteExpr{Method: method, Path: path}

	switch e := eval.Current().(type) {
	case *expr.HTTPServiceExpr:
		// Service-level route - only allowed for JSON-RPC services
		if e.ServiceExpr.Meta == nil || e.ServiceExpr.Meta["jsonrpc:service"] == nil {
			eval.ReportError("routes at the service level are only allowed for JSON-RPC services. Use method-level routes instead.")
			return r
		}
		// For JSON-RPC services, store the route in the service
		e.JSONRPCRoute = r
		return r

	case *expr.HTTPEndpointExpr:
		// Method-level route - not allowed for JSON-RPC endpoints
		if e.IsJSONRPC() {
			eval.ReportError("JSON-RPC endpoints cannot define routes at the method level. Define routes at the service level using JSONRPC(func() { GET(\"/path\") })")
			return r
		}
		r.Endpoint = e
		e.Routes = append(e.Routes, r)
		return r

	default:
		eval.IncompatibleDSL()
		return r
	}
}
