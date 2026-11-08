package dsl

import (
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

// Response describes a HTTP, JSON-RPC or gRPC response. Response describes both
// success and error responses. When describing an error response the first
// argument is the name of the error.
//
// While a service method may only define a single result type, Response may
// appear multiple times to define multiple success HTTP responses. In this case
// the Tag expression makes it possible to identify a result type attribute and
// a corresponding string value used to select the proper success response.
// Without Tags, multiple HTTP success responses select by a shared StatusField.
// HTTP status classes and StatusFallback also require StatusField. JSON-RPC and
// gRPC responses may only define one success response.
//
// Response may appear in a service expression to define error responses common
// to all the service methods. Response may also appear in a method expression
// to define both success and error responses specific to the method. In both
// cases Response must appear in the transport specific DSL (i.e. in a HTTP,
// JSON-RPC or gRPC subexpression).
//
// Response accepts a selector and an optional mapping callback. HTTP SSE may
// also pass a handshake type between the selector and callback, separate from
// StreamingResult event values. Other transports do not accept handshake types.
// HTTP selectors are exact status integers, Status2XX through Status5XX, or
// StatusFallback. Use Informational for interim statuses.
//
// Success response accepts a status
// code as first argument. If the first argument is a status code then a
// function may be given as the second argument. This function may provide a
// description and describes how to map the result type attributes to transport
// specific constructs (e.g. HTTP headers and body, JSON-RPC ID and result,
// gRPC metadata and message).
//
// JSON-RPC Responses:
//
// For JSON-RPC, Response configures how the method result is mapped to the
// JSON-RPC response structure. The JSON-RPC protocol defines a fixed envelope
// with "id", "result" and "error" fields.
//
// - Success responses: The entire method result is used as the "result" field
// - Error responses: Map errors to JSON-RPC error codes using Response
// - The response "id" automatically matches the request "id" if present
//
// The valid invocations for successful response are thus:
//
// * Response(status)
//
// * Response(func)
//
// * Response(status, func)
//
// Error responses additionally accept the name of the error as first or second argument.
//
// * Response(error_name, status)
//
// * Response(error_name, func)
//
// * Response(error_name, status, func)
//
// * Response(status, error_name)
//
// * Response(status, error_name, func)
//
// By default:
//
//   - success HTTP responses use code 200 (OK) and error HTTP responses use code 400 (Bad Request)
//   - error JSON-RPC responses use code -32603 (Internal error)
//   - success gRPC responses use code 0 (OK) and error gRPC responses use code 2 (Unknown)
//   - The result type attributes are all mapped to the HTTP response body,
//     JSON-RPC result, or gRPC response message.
//
// Example:
//
//	Method("create", func() {
//	    Payload(CreatePayload)
//	    Result(CreateResult)
//	    Error("an_error")
//
//	    HTTP(func() {
//	        Response(StatusAccepted, func() { // HTTP status code set using argument
//	            Description("Response used for async creations")
//	            Tag("outcome", "accepted") // Tag identifies a result type attribute and corresponding
//	                                       // value for this response to be selected.
//	            Header("taskHref")         // map "taskHref" attribute to header, all others to body
//	        })
//
//	        Response(StatusCreated, func () {
//	            Tag("outcome", "created")  // CreateResult type to describe body
//	        })
//
//	        Response(func() {
//	            Description("Response used when item already exists")
//	            Code(StatusNoContent) // HTTP status code set using Code
//	            Body(Empty)           // Override method result type
//	        })
//
//	        Response("an_error", StatusConflict) // Override default of 400
//	    })
//
//	    JSONRPC(func() {
//	        Response("an_error", RPCInvalidParams, func() {
//	            Description("Invalid parameters provided")
//	        })
//	    })
//
//	    GRPC(func() {
//	        Response(CodeOK, func() {
//	            Metadata("taskHref") // map "taskHref" attribute to metadata, all others to message
//	        })
//
//	        Response("an_error", CodeInternal, func() {
//	            Description("Error returned for internal errors")
//	        })
//	    })
//	})
func Response(val any, args ...any) {
	name, ok, val, args := responseNameArgs(val, args)
	switch t := eval.Current().(type) {
	case *expr.RootExpr:
		appendHTTPOrJSONRPCError(name, ok, val, args, t, &t.API.HTTP.Errors)
	case *expr.HTTPExpr:
		appendHTTPOrJSONRPCError(name, ok, val, args, t, &t.Errors)
	case *expr.GRPCExpr:
		appendGRPCError(name, ok, val, args, t, &t.Errors)
	case *expr.JSONRPCExpr:
		appendHTTPOrJSONRPCError(name, ok, val, args, t, &t.Errors)
	case *expr.HTTPServiceExpr:
		appendHTTPOrJSONRPCError(name, ok, val, args, t, &t.HTTPErrors)
	case *expr.GRPCServiceExpr:
		appendGRPCError(name, ok, val, args, t, &t.GRPCErrors)
	case *expr.HTTPEndpointExpr:
		if ok {
			appendHTTPOrJSONRPCError(name, true, val, args, t, &t.HTTPErrors)
			return
		}
		if response := httpEndpointResponse(t, val, args...); response != nil {
			t.Responses = append(t.Responses, response)
		}
	case *expr.GRPCEndpointExpr:
		if ok {
			appendGRPCError(name, true, val, args, t, &t.GRPCErrors)
			return
		}
		t.Response = grpcEndpointResponse(t, val, args...)
	default:
		eval.IncompatibleDSL()
	}
}

// Code sets the Response status code.
//
// Code must appear in a Response expression.
//
// Code accepts one argument: the HTTP, JSON-RPC or gRPC status code.
func Code(code int) {
	switch t := eval.Current().(type) {
	case *expr.HTTPResponseExpr:
		if t.JSONRPCError != nil {
			t.JSONRPCError.Code = code
		} else {
			t.Status = expr.HTTPStatusSelector{Code: code}
		}
	case *expr.GRPCResponseExpr:
		t.StatusCode = code
	default:
		eval.IncompatibleDSL()
	}
}

func appendHTTPOrJSONRPCError(
	name string,
	ok bool,
	val any,
	args []any,
	parent eval.Expression,
	errors *[]*expr.HTTPErrorExpr,
) {
	if !ok {
		eval.InvalidArgError("name of error", val)
		return
	}
	if hasHTTPErrorName(*errors, name) {
		eval.ReportError("error response %q is defined multiple times", name)
		return
	}
	if e := httpOrJSONRPCError(name, parent, args...); e != nil {
		*errors = append(*errors, e)
	}
}

func appendGRPCError(
	name string,
	ok bool,
	val any,
	args []any,
	parent eval.Expression,
	errors *[]*expr.GRPCErrorExpr,
) {
	if !ok {
		eval.InvalidArgError("name of error", val)
		return
	}
	if hasGRPCErrorName(*errors, name) {
		eval.ReportError("error response %q is defined multiple times", name)
		return
	}
	if e := grpcError(name, parent, args...); e != nil {
		*errors = append(*errors, e)
	}
}

func hasHTTPErrorName(errors []*expr.HTTPErrorExpr, name string) bool {
	for _, err := range errors {
		if err.Name == name {
			return true
		}
	}
	return false
}

func hasGRPCErrorName(errors []*expr.GRPCErrorExpr, name string) bool {
	for _, err := range errors {
		if err.Name == name {
			return true
		}
	}
	return false
}

func responseNameArgs(val any, args []any) (string, bool, any, []any) {
	name, ok := val.(string)
	if ok || len(args) == 0 {
		return name, ok, val, args
	}
	name, ok = args[0].(string)
	if !ok {
		return "", false, val, args
	}
	return name, true, args[0], append([]any{val}, args[1:]...)
}

func httpEndpointResponse(parent *expr.HTTPEndpointExpr, val any, args ...any) *expr.HTTPResponseExpr {
	if parent.IsJSONRPC() {
		if _, class := val.(expr.HTTPStatusClass); class {
			eval.ReportError("JSON-RPC responses do not accept HTTP status classes")
			return &expr.HTTPResponseExpr{Status: expr.HTTPStatusSelector{Code: expr.StatusOK}, Parent: parent}
		}
	}
	status, handshake, fn, ok := parseHTTPResponseArgs(val, expr.StatusOK, true, args...)
	if !ok {
		return nil
	}
	resp := &expr.HTTPResponseExpr{Status: status, Parent: parent, Handshake: handshake}
	executeResponseDSL(fn, resp)
	return resp
}

func grpcEndpointResponse(parent *expr.GRPCEndpointExpr, val any, args ...any) *expr.GRPCResponseExpr {
	code, fn, _ := parseResponseArgs(val, args...)
	resp := &expr.GRPCResponseExpr{
		StatusCode: code,
		Parent:     parent,
	}
	executeResponseDSL(fn, resp)
	return resp
}

func grpcError(n string, p eval.Expression, args ...any) *expr.GRPCErrorExpr {
	if len(args) == 0 {
		eval.TooFewArgError()
		return nil
	}
	var (
		code int
		fn   func()
		val  any
	)
	val = args[0]
	args = args[1:]
	code, fn, _ = parseResponseArgs(val, args...)
	if code == 0 {
		code = CodeUnknown
	}
	resp := &expr.GRPCResponseExpr{
		StatusCode: code,
		Parent:     p,
	}
	executeResponseDSL(fn, resp)
	return &expr.GRPCErrorExpr{
		Name:     n,
		Response: resp,
	}
}

func parseResponseArgs(val any, args ...any) (code int, fn func(), supplied bool) {
	switch t := val.(type) {
	case int:
		supplied = true
		code = t
		if len(args) > 1 {
			eval.TooManyArgError()
			return
		}
		if len(args) == 1 {
			if d, ok := args[0].(func()); ok {
				fn = d
			} else {
				eval.InvalidArgError("function", args[0])
				return
			}
		}
	case func():
		if len(args) > 0 {
			eval.TooManyArgError()
			return
		}
		fn = t
	default:
		eval.InvalidArgError("int (HTTP status code) or function", val)
		return
	}
	return
}

func httpOrJSONRPCError(n string, p eval.Expression, args ...any) *expr.HTTPErrorExpr {
	if len(args) == 0 {
		eval.TooFewArgError()
		return nil
	}
	var (
		code int
		fn   func()
		val  any
	)
	val = args[0]
	args = args[1:]
	resp := &expr.HTTPResponseExpr{Parent: p, ErrorName: n}
	if expr.IsJSONRPCScope(p) {
		if _, class := val.(expr.HTTPStatusClass); class {
			eval.ReportError("JSON-RPC error responses do not accept HTTP status classes")
			return nil
		}
		var supplied bool
		code, fn, supplied = parseResponseArgs(val, args...)
		if !supplied {
			code = expr.RPCInternalError
		}
		resp.JSONRPCError = &expr.JSONRPCErrorMapping{Code: code}
	} else {
		var ok bool
		resp.Status, _, fn, ok = parseHTTPResponseArgs(val, expr.StatusBadRequest, false, args...)
		if !ok {
			return nil
		}
	}
	executeResponseDSL(fn, resp)
	return &expr.HTTPErrorExpr{
		Name:     n,
		Response: resp,
	}
}

func executeResponseDSL(fn func(), target eval.Expression) {
	if fn != nil {
		eval.Execute(fn, target)
	}
}

func parseHTTPResponseArgs(
	val any,
	defaultCode int,
	allowValue bool,
	args ...any,
) (expr.HTTPStatusSelector, *expr.AttributeExpr, func(), bool) {
	status := expr.HTTPStatusSelector{Code: defaultCode}
	maximum := 1
	if allowValue {
		maximum = 2
	}
	if len(args) > maximum {
		eval.TooManyArgError()
		return status, nil, nil, false
	}
	switch v := val.(type) {
	case int:
		status = expr.HTTPStatusSelector{Code: v}
	case expr.HTTPStatusClass:
		status = expr.HTTPStatusSelector{Class: v}
	case func():
		if len(args) != 0 {
			eval.TooManyArgError()
			return status, nil, nil, false
		}
		return status, nil, v, true
	default:
		eval.InvalidArgError("HTTP status code, class or function", val)
		return status, nil, nil, false
	}
	if len(args) == 0 {
		return status, nil, nil, true
	}
	switch v := args[0].(type) {
	case func():
		if len(args) != 1 {
			eval.TooManyArgError()
			return status, nil, nil, false
		}
		return status, nil, v, true
	case expr.DataType:
		if !allowValue {
			eval.ReportError("typed handshake is only valid on an SSE success Response")
			return status, nil, nil, false
		}
		var fn func()
		if len(args) == 2 {
			var ok bool
			fn, ok = args[1].(func())
			if !ok {
				eval.InvalidArgError("function", args[1])
				return status, nil, nil, false
			}
		}
		return status, &expr.AttributeExpr{Type: v}, fn, true
	default:
		expected := "function"
		if allowValue {
			expected = "handshake type or function"
		}
		eval.InvalidArgError(expected, args[0])
		return status, nil, nil, false
	}
}
