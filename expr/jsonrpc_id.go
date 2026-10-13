package expr

import "github.com/CaliLuke/loom/eval"

// JSONRPCIDBinding maps one service field exclusively to the request envelope.
// Attribute retains its authored constraints and representation. Protocol
// presence and correlation remain properties of the original request envelope.
type JSONRPCIDBinding struct {
	// Name is the actual service object key, including an authored element suffix.
	Name string
	// Attribute is the effective service field declaration.
	Attribute *AttributeExpr
	// Required requires the envelope ID to be present before defaults apply.
	Required bool
}

// jsonrpcIDBodyConflict checks actual selected fields, not the Reference graph
// that supplies their definitions. References can also contain excluded fields.
func jsonrpcIDBodyConflict(body *AttributeExpr, name string) bool {
	if body == nil {
		return false
	}
	selected, _ := body.Meta.Last("origin:attribute")
	if selected != "" {
		return AttributeName(selected) == AttributeName(name)
	}
	object := AsObject(body.Type)
	if object == nil {
		return false
	}
	for _, field := range *object {
		if AttributeName(field.Name) == AttributeName(name) {
			return true
		}
	}
	return false
}

func (e *HTTPEndpointExpr) resolveJSONRPCID(plan *httpWireBindingPlan) {
	if !e.IsJSONRPC() {
		return
	}
	payload := e.jsonrpcRequestPayload()
	name := jsonrpcIDAttributeName(payload)
	if name == "" {
		return
	}
	_, attribute := payload.FindAttribute(name)
	e.JSONRPCID = &JSONRPCIDBinding{Name: name, Attribute: attribute, Required: payload.IsRequired(name)}
	plan.bodyExclusions[AttributeName(name)] = true
}

func (e *HTTPEndpointExpr) validateJSONRPCID(verr *eval.ValidationErrors) {
	payload := e.jsonrpcRequestPayload()
	if payload == nil || !hasJSONRPCIDField(payload) {
		return
	}
	object := AsObject(payload.Type)
	if object == nil {
		verr.Add(e, "JSON-RPC ID must designate a top-level payload field")
		return
	}
	count := 0
	for _, field := range *object {
		if _, marked := field.Attribute.Meta["jsonrpc:id"]; !marked {
			if hasJSONRPCIDField(field.Attribute) {
				verr.Add(e, "JSON-RPC ID %q must be a top-level payload field", field.Name)
			}
			continue
		}
		count++
		if unalias(field.Attribute.Type) != String {
			verr.Add(e, "JSON-RPC ID %q must use String or a named String type", field.Name)
		}
		for _, mapping := range []*MappedAttributeExpr{e.Headers, e.Params, e.Cookies} {
			if mapping != nil && mapping.Find(AttributeName(field.Name)) != nil {
				verr.Add(e, "JSON-RPC ID %q belongs to the envelope and cannot also use an HTTP mapping", field.Name)
			}
		}
		if jsonrpcIDBodyConflict(e.Body, field.Name) {
			verr.Add(e, "JSON-RPC ID %q belongs to the envelope and cannot also appear in an explicit params body", field.Name)
		}
	}
	if count > 1 {
		verr.Add(e, "JSON-RPC request has multiple ID fields; the envelope has one ID")
	}
}
