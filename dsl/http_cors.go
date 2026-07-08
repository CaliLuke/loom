package dsl

import (
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

// CORS defines the Cross-Origin Resource Sharing policy for an API or HTTP
// service. Service-level CORS overrides the API-level policy for that service.
func CORS(fn func()) {
	cors := new(expr.HTTPCORSExpr)
	if !eval.Execute(fn, cors) {
		return
	}
	switch def := eval.Current().(type) {
	case *expr.RootExpr:
		def.API.HTTP.CORS = cors
	case *expr.HTTPServiceExpr:
		def.CORS = cors
	default:
		eval.IncompatibleDSL()
	}
}

// Origin adds an allowed CORS origin. Use "*" for a wildcard origin.
func Origin(pattern string, fn ...func()) {
	origin(pattern, false, fn...)
}

// OriginRegex adds an allowed CORS origin regular expression.
func OriginRegex(pattern string, fn ...func()) {
	origin(pattern, true, fn...)
}

// Methods sets the HTTP methods allowed for the current CORS origin.
func Methods(methods ...string) {
	if o, ok := eval.Current().(*expr.HTTPCORSOriginExpr); ok {
		o.Methods = append(o.Methods, methods...)
		return
	}
	eval.IncompatibleDSL()
}

// Expose sets the response headers exposed to browser clients for the current
// CORS origin.
func Expose(headers ...string) {
	if o, ok := eval.Current().(*expr.HTTPCORSOriginExpr); ok {
		o.Expose = append(o.Expose, headers...)
		return
	}
	eval.IncompatibleDSL()
}

// MaxAge sets the Access-Control-Max-Age value, in seconds, for the current
// CORS origin.
func MaxAge(seconds int) {
	if o, ok := eval.Current().(*expr.HTTPCORSOriginExpr); ok {
		o.MaxAge = seconds
		return
	}
	eval.IncompatibleDSL()
}

// Credentials allows credentialed browser requests for the current CORS
// origin.
func Credentials() {
	if o, ok := eval.Current().(*expr.HTTPCORSOriginExpr); ok {
		o.Credentials = true
		return
	}
	eval.IncompatibleDSL()
}

func origin(pattern string, regex bool, fns ...func()) {
	if len(fns) > 1 {
		eval.TooManyArgError()
		return
	}
	cors, ok := eval.Current().(*expr.HTTPCORSExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	o := &expr.HTTPCORSOriginExpr{Pattern: pattern, Regex: regex}
	if len(fns) == 1 && !eval.Execute(fns[0], o) {
		return
	}
	cors.Origins = append(cors.Origins, o)
}
