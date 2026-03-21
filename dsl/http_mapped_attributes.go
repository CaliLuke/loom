package dsl

import (
	"goa.design/goa/v3/eval"
	"goa.design/goa/v3/expr"
)

// headers returns the mapped attribute containing the headers for the given
// expression if it's either the root, a service or an endpoint - nil otherwise.
func headers(exp eval.Expression) *expr.MappedAttributeExpr {
	switch e := exp.(type) {
	case *expr.RootExpr:
		if e.API.HTTP.Headers == nil {
			e.API.HTTP.Headers = expr.NewEmptyMappedAttributeExpr()
		}
		return e.API.HTTP.Headers
	case *expr.HTTPServiceExpr:
		if e.Headers == nil {
			e.Headers = expr.NewEmptyMappedAttributeExpr()
		}
		return e.Headers
	case *expr.HTTPEndpointExpr:
		if e.Headers == nil {
			e.Headers = expr.NewEmptyMappedAttributeExpr()
		}
		return e.Headers
	case *expr.HTTPResponseExpr:
		if e.Headers == nil {
			e.Headers = expr.NewEmptyMappedAttributeExpr()
		}
		return e.Headers
	case *expr.MappedAttributeExpr:
		return e
	default:
		return nil
	}
}

// cookies returns the mapped attribute containing the cookies for the given
// expression if it's either the root, a service or an endpoint - nil otherwise.
func cookies(exp eval.Expression) *expr.MappedAttributeExpr {
	switch e := exp.(type) {
	case *expr.RootExpr:
		if e.API.HTTP.Cookies == nil {
			e.API.HTTP.Cookies = expr.NewEmptyMappedAttributeExpr()
		}
		return e.API.HTTP.Cookies
	case *expr.HTTPServiceExpr:
		if e.Cookies == nil {
			e.Cookies = expr.NewEmptyMappedAttributeExpr()
		}
		return e.Cookies
	case *expr.HTTPEndpointExpr:
		if e.Cookies == nil {
			e.Cookies = expr.NewEmptyMappedAttributeExpr()
		}
		return e.Cookies
	case *expr.MappedAttributeExpr:
		return e
	default:
		return nil
	}
}

// params returns the mapped attribute containing the path and query params for
// the given expression if it's either the root, an API server, a service or an
// endpoint - nil otherwise.
func params(exp eval.Expression) *expr.MappedAttributeExpr {
	switch e := exp.(type) {
	case *expr.RootExpr:
		if e.API.HTTP.Params == nil {
			e.API.HTTP.Params = expr.NewEmptyMappedAttributeExpr()
		}
		return e.API.HTTP.Params
	case *expr.HTTPServiceExpr:
		if e.Params == nil {
			e.Params = expr.NewEmptyMappedAttributeExpr()
		}
		return e.Params
	case *expr.HTTPEndpointExpr:
		if e.Params == nil {
			e.Params = expr.NewEmptyMappedAttributeExpr()
		}
		return e.Params
	case *expr.MappedAttributeExpr:
		return e
	default:
		return nil
	}
}
