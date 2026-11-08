package dsl

import (
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

// Informational declares a named interim response with its own value type.
// It accepts a status, type and optional Header/StatusField/Description callback.
// It must appear in a method HTTP expression. The type supplies header values,
// independently of the method result. Exact statuses 100 and 102..199 may omit
// StatusField; Status1XX requires it. Exact 100 declares observation and OpenAPI
// documentation only: its generated Send always rejects. The transport owns
// automatic 100; upgrade 101 cannot be declared. Informational values have no body.
func Informational(name string, args ...any) {
	e, ok := eval.Current().(*expr.HTTPEndpointExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	if e.IsJSONRPC() {
		eval.ReportError("Informational is HTTP only, not JSON-RPC")
		return
	}
	if len(args) < 2 {
		eval.TooFewArgError()
		return
	}
	status, value, fn, ok := parseHTTPResponseArgs(args[0], 0, true, args[1:]...)
	if !ok {
		return
	}
	if value == nil {
		eval.ReportError("Informational requires a value type")
		return
	}
	r := &expr.HTTPInformationalExpr{Name: name, Value: value, HTTPResponseExpr: &expr.HTTPResponseExpr{Status: status, Parent: e}}
	r.Prepare()
	e.Informational = append(e.Informational, r)
	executeResponseDSL(fn, r)
}

// StatusField binds a required native Int member to an HTTP response's status.
// It appears in Response or Informational and excludes that member from an
// inferred body. The field must be non-nullable and have no default. It cannot
// also be mapped to a header, cookie or explicit body. All status-selected
// success branches bind the same field, which must survive every admitted view.
func StatusField(name string) {
	var response *expr.HTTPResponseExpr
	switch r := eval.Current().(type) {
	case *expr.HTTPResponseExpr:
		response = r
	case *expr.HTTPInformationalExpr:
		response = r.HTTPResponseExpr
	default:
		eval.IncompatibleDSL()
		return
	}
	if response.StatusField != "" {
		eval.ReportError("StatusField is defined multiple times")
		return
	}
	response.StatusField = name
}

// NoSuccessResponse declares an HTTP operation with an empty result and only
// named error responses. It must appear in a non-streaming method HTTP
// expression and suppresses the implicit success response.
func NoSuccessResponse() {
	e, ok := eval.Current().(*expr.HTTPEndpointExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	if e.IsJSONRPC() {
		eval.ReportError("NoSuccessResponse is HTTP only, not JSON-RPC")
		return
	}
	e.NoSuccess = true
}
