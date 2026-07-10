package transportir

import "github.com/CaliLuke/loom/expr"

func buildStream(endpoint *expr.HTTPEndpointExpr) *Stream {
	if endpoint == nil || endpoint.MethodExpr == nil {
		return nil
	}
	stream := &Stream{
		Kind:            endpoint.MethodExpr.Stream,
		Direction:       streamDirection(endpoint.MethodExpr.Stream),
		IsStreaming:     endpoint.MethodExpr.IsStreaming(),
		HasMixedResults: endpoint.MethodExpr.HasMixedResults(),
		RequestHasBody:  endpoint.StreamingBody != nil && endpoint.StreamingBody.Type != expr.Empty,
		RequestPayload:  endpoint.MethodExpr.StreamingPayload,
	}
	if !stream.IsStreaming {
		return stream
	}
	stream.RequestMessage = endpoint.MethodExpr.StreamingPayload
	if endpoint.MethodExpr.IsPayloadStreaming() && endpoint.StreamingBody != nil {
		stream.RequestMessage = endpoint.StreamingBody
	}
	stream.ResponseMessage = endpoint.MethodExpr.StreamingResult
	switch {
	case endpoint.SSE != nil:
		stream.Transport = "sse"
		stream.IsSSE = true
		stream.HandshakeContent = "text/event-stream"
		if len(endpoint.Routes) > 0 {
			stream.HandshakeMethod = endpoint.Routes[0].Method
		}
		if len(endpoint.Responses) > 0 {
			stream.HandshakeStatus = endpoint.Responses[0].StatusCode
		} else {
			stream.HandshakeStatus = expr.StatusOK
		}
		requestIDPointer := false
		if endpoint.SSE.RequestIDField != "" && endpoint.MethodExpr.Payload != nil {
			requestIDPointer = endpoint.MethodExpr.Payload.IsPrimitivePointer(endpoint.SSE.RequestIDField, true)
		}
		stream.SSE = &SSE{
			RequestIDField:     endpoint.SSE.RequestIDField,
			RequestIDPointer:   requestIDPointer,
			NotificationMethod: endpoint.SSE.NotificationMethod,
			DataField:          endpoint.SSE.DataField,
			IDField:            endpoint.SSE.IDField,
			EventField:         endpoint.SSE.EventField,
			RetryField:         endpoint.SSE.RetryField,
		}
	default:
		stream.Transport = "websocket"
		stream.IsWebSocket = true
		if len(endpoint.Routes) > 0 {
			stream.HandshakeMethod = endpoint.Routes[0].Method
		}
		stream.HandshakeStatus = expr.StatusSwitchingProtocols
	}
	return stream
}

func streamDirection(kind expr.StreamKind) string {
	switch kind {
	case expr.ServerStreamKind:
		return "server"
	case expr.ClientStreamKind:
		return "client"
	default:
		return "bidirectional"
	}
}
