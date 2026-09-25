package service

import "github.com/CaliLuke/loom/expr"

type (
	// methodSignature lists the parameters that a service method takes after
	// its context and the results that it returns before its error. The
	// service interface and the example stubs render the same signature, so
	// the stubs implement the interface.
	methodSignature struct {
		// Payload is true when the method takes the payload.
		Payload bool
		// Stream is true when the method takes the server stream.
		Stream bool
		// RequestBody is true when the method takes the raw request body.
		RequestBody bool
		// Result is true when the method returns the result.
		Result bool
		// ResponseBody is true when the method returns the raw response body.
		ResponseBody bool
		// FileResponse is true when the method returns a file response.
		FileResponse bool
		// View is true when the method returns the view of the result.
		View bool
	}
)

// serviceMethodSignature returns the signature of the service method that
// method describes. A method with a server stream returns only an error,
// except when it also has a result that differs from the streamed type. The
// methods of a JSON-RPC WebSocket service that receive a streaming payload
// and send no stream handle one message each: they take the message as
// their payload and return the result.
func serviceMethodSignature(method *MethodData) methodSignature {
	sig := methodSignature{Payload: method.Payload != ""}
	view := method.Result != "" && method.ViewedResult != nil && method.ViewedResult.ViewName == ""
	if method.ServerStream == nil {
		sig.RequestBody = method.SkipRequestBodyEncodeDecode
		sig.Result = method.Result != ""
		sig.ResponseBody = method.SkipResponseBodyEncodeDecode
		sig.FileResponse = !method.SkipResponseBodyEncodeDecode && method.FileResponse
		sig.View = view
		return sig
	}
	switch {
	case method.IsJSONRPC && !method.IsJSONRPCSSE && method.ServerStream.Kind == expr.ClientStreamKind:
		sig.Result = method.Result != ""
	case method.HasMixedResults:
		sig.Stream = true
		sig.Result = method.Result != ""
		sig.View = view
	default:
		sig.Stream = true
	}
	return sig
}
