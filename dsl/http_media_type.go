package dsl

import (
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

// MediaType binds an HTTP media type or range to a branch of the selected Body
// union. Use it inside an endpoint HTTP block or Response. Every union branch
// must have exactly one mapping. Requests and client responses select the most
// specific mapping from Content-Type; clients and services select outgoing
// representations through the ordinary union constructors. A range requires a
// mapped Content-Type header supplying a concrete outgoing type. MediaType does
// not perform Accept negotiation or change the union's JSON representation
// outside this HTTP body occurrence.
func MediaType(media, branch string) {
	mapping := &expr.HTTPMediaTypeExpr{ContentType: media, Branch: branch}
	switch current := eval.Current().(type) {
	case *expr.HTTPEndpointExpr:
		current.MediaTypes = append(current.MediaTypes, mapping)
	case *expr.HTTPResponseExpr:
		current.MediaTypes = append(current.MediaTypes, mapping)
	default:
		eval.IncompatibleDSL()
	}
}
