package service

import "github.com/CaliLuke/loom/codegen"

type (
	// serverStreamPlan owns the emitted server operations and the carrier used
	// when a transport dispatches an incoming message through an endpoint.
	serverStreamPlan struct {
		operations []serverStreamOperation
		endpoint   string
		incoming   bool
	}
	serverStreamOperation struct {
		name        string
		description string
		parameters  []streamOperationParameter
		results     []string
		action      streamOperationAction
		// contextual names the same operation with an explicit context. Empty
		// means this operation already carries context or has no contextual form.
		contextual string
	}
	streamOperationParameter struct {
		name  string
		ref   string
		named bool
	}
	streamOperationAction int
)

const (
	streamDelegate streamOperationAction = iota
	streamSend
	streamReceive
)

func newServerStreamPlan(method *MethodData) *serverStreamPlan {
	s := method.ServerStream
	p := &serverStreamPlan{
		endpoint: s.EndpointStruct,
		incoming: method.IsJSONRPCWebSocket && method.StreamingPayloadRef != "",
	}
	ctx := streamOperationParameter{name: "ctx", ref: "context.Context"}
	p.addSendOperations(method)
	if s.RecvTypeRef != "" && !method.IsJSONRPCWebSocket {
		p.operations = append(p.operations,
			serverStreamOperation{
				name:        s.RecvName,
				description: s.RecvDesc,
				action:      streamReceive,
				results:     []string{s.RecvTypeRef, "error"},
				contextual:  s.RecvWithContextName,
			},
			serverStreamOperation{
				name:        s.RecvWithContextName,
				description: s.RecvWithContextDesc,
				action:      streamReceive,
				parameters:  []streamOperationParameter{ctx},
				results:     []string{s.RecvTypeRef, "error"},
			},
		)
	}
	if !method.IsJSONRPCSSE && (method.IsJSONRPCWebSocket || s.MustClose) {
		p.addOperation("Close", "Close closes the stream.", streamDelegate)
	}
	if method.ViewedResult != nil && method.ViewedResult.ViewName == "" {
		doc := "SetView sets the view used to render the result before streaming."
		if method.IsJSONRPCSSE {
			doc = "SetView sets the view used to render subsequent stream results."
		}
		p.operations = append(p.operations, serverStreamOperation{
			name:        "SetView",
			description: doc,
			parameters:  []streamOperationParameter{{name: "view", ref: "string", named: true}},
		})
	}
	if method.IsJSONRPCSSE {
		p.addOperation("SendError", "SendError sends a JSON-RPC error response.", streamDelegate, ctx, streamOperationParameter{name: "id", ref: "any"}, streamOperationParameter{name: "err", ref: "error"})
	}
	return p
}

func (p *serverStreamPlan) addSendOperations(method *MethodData) {
	s := method.ServerStream
	ctx := streamOperationParameter{name: "ctx", ref: "context.Context"}
	value := streamOperationParameter{name: "v", ref: s.SendTypeRef}
	if s.SendTypeRef != "" {
		switch {
		case method.IsJSONRPCSSE:
			value.ref = method.EventName
			p.addOperation("Send", s.SendDesc+"\nIMPORTANT: Send only sends JSON-RPC notifications. Use SendAndClose to send a final response.", streamSend, ctx, value)
			if s.SendAndCloseName != "" {
				p.addOperation(s.SendAndCloseName, s.SendAndCloseDesc+"\nThe result will be sent as a JSON-RPC response with the original request ID.\nIf the result has an ID field populated, that ID will be used instead of the request ID.", streamSend, ctx, value)
			}
		case method.IsJSONRPCWebSocket:
			p.addOperation("SendNotification", "SendNotification sends a JSON-RPC notification (no response expected).", streamSend, ctx, value)
			p.addOperation("SendResponse", "SendResponse sends a JSON-RPC response with the original request ID.", streamSend, ctx, value)
			p.addOperation("SendError", "SendError sends a JSON-RPC error response.", streamDelegate, ctx, streamOperationParameter{name: "err", ref: "error"})
		default:
			p.addOperation(s.SendName, s.SendDesc, streamSend, value)
			p.operations[len(p.operations)-1].contextual = s.SendWithContextName
			p.addOperation(s.SendWithContextName, s.SendWithContextDesc, streamSend, ctx, value)
		}
	}
}

func (p *serverStreamPlan) addOperation(name, doc string, action streamOperationAction, params ...streamOperationParameter) {
	p.operations = append(p.operations, serverStreamOperation{
		name:        name,
		description: doc,
		action:      action,
		parameters:  params,
		results:     []string{"error"},
	})
}

func (op serverStreamOperation) callback() string {
	return codegen.Goify(op.name, false)
}
