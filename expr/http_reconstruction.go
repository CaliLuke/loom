package expr

import "github.com/CaliLuke/loom/eval"

// missingReconstructionFields compares service field identities with the actual
// decoder inputs. References supply declarations, not values; a default on a
// field absent from all inputs cannot materialize that field.
func missingReconstructionFields(source, body *AttributeExpr, bindings []string) []string {
	if source == nil || AsObject(source.Type) == nil {
		return nil
	}
	covered := make(map[string]bool, len(bindings))
	for _, name := range bindings {
		covered[AttributeName(name)] = true
	}
	if body != nil {
		if selected, ok := body.Meta.Last("origin:attribute"); ok {
			covered[AttributeName(selected)] = true
		} else if object := AsObject(body.Type); object != nil {
			for _, field := range *object {
				covered[AttributeName(field.Name)] = true
			}
		}
	}
	var missing []string
	for _, name := range source.AllRequired() {
		logical := AttributeName(name)
		if _, field := source.FindAttribute(name); field != nil && !covered[logical] {
			missing = append(missing, name)
			covered[logical] = true
		}
	}
	return missing
}

func validateReconstruction(source, body *AttributeExpr, bindings []string, label string, parent eval.Expression, errors *eval.ValidationErrors) {
	for _, name := range missingReconstructionFields(source, body, bindings) {
		errors.Add(parent, "required %s attribute %q has no reconstruction source in the runtime body or transport bindings", label, name)
	}
}

func responseReconstructionBindings(response *HTTPResponseExpr) []string {
	var bindings []string
	if response.Headers != nil {
		if object := AsObject(response.Headers.Type); object != nil {
			for _, field := range *object {
				bindings = append(bindings, field.Name)
			}
		}
	}
	for _, cookie := range response.Cookies {
		if !cookie.IsEmpty() {
			bindings = append(bindings, cookie.AttributeName())
		}
	}
	if response.TagBinding != nil {
		bindings = append(bindings, response.TagBinding.Key)
	}
	if response.StatusErrorName != "" {
		bindings = append(bindings, response.StatusErrorName)
	}
	return bindings
}

func (e *HTTPEndpointExpr) validateRequestReconstruction(errors *eval.ValidationErrors) {
	source := e.MethodExpr.Payload
	if source == nil || AsObject(source.Type) == nil {
		return
	}
	var bindings []string
	for _, binding := range e.ensureWireBindings().bindings {
		if binding.sourceKind == "payload" && binding.location != "body" {
			bindings = append(bindings, binding.sources...)
		}
	}
	if e.JSONRPCID != nil && e.jsonrpcRequestPayload() == source {
		bindings = append(bindings, e.JSONRPCID.Name)
	}
	if e.MapQueryParams != nil {
		bindings = append(bindings, *e.MapQueryParams)
	}
	validateReconstruction(source, httpRequestBody(e), bindings, "payload", e, errors)
}

func (r *HTTPResponseExpr) validateResultReconstruction(e *HTTPEndpointExpr, errors *eval.ValidationErrors) {
	// JSON-RPC reconstructs the first success representation; it does not select
	// HTTP status branches. Its first representation still applies its own Tag.
	if ordered := e.OrderedResponses(); e.IsJSONRPC() && (len(ordered) == 0 || ordered[0] != r) {
		return
	}
	source := e.MethodExpr.Result
	if source == nil {
		return
	}
	body, _ := responseBodySource(source, r)
	bindings := responseReconstructionBindings(r)
	result, viewed := source.Type.(*ResultTypeExpr)
	if !viewed {
		validateReconstruction(source, body, bindings, "result", r, errors)
		return
	}
	for _, view := range result.Views {
		if selected, fixed := source.Meta.Last(ViewMetaKey); fixed && selected != view.Name {
			continue
		}
		projected, err := Project(result, view.Name)
		if err != nil {
			// The result/view owner reports malformed projections.
			continue
		}
		validateReconstruction(projected.AttributeExpr, body, bindings, "result in view "+view.Name, r, errors)
	}
}
