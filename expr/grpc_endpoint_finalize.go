package expr

// Finalize ensures the request and response attributes are initialized.
func (e *GRPCEndpointExpr) Finalize() {
	if errors := e.ensureMetadataBindings(); len(errors.Errors) > 0 {
		return
	}
	if pobj := AsObject(e.MethodExpr.Payload.Type); pobj != nil {
		e.finalizeRequestMessageFromPayload(pobj)
		e.propagatePayloadProtoStructName()
	} else {
		e.finalizeNonObjectPayload()
	}

	e.finalizeStreamingRequest()
	e.Response.Finalize(e, e.MethodExpr.Result)
	for _, gerr := range e.GRPCErrors {
		gerr.Finalize(e)
	}
}

func (e *GRPCEndpointExpr) finalizeRequestMessageFromPayload(pobj *Object) {
	msgObj := Dup(pobj).(*Object)
	for _, nat := range *AsObject(e.Metadata.Type) {
		key, attr := objectAttribute(pobj, nat.Name)
		initAttrFromDesign(nat.Attribute, attr)
		if e.MethodExpr.Payload.IsRequired(key) {
			e.Metadata.Validation.AddRequired(nat.Name)
		}
		msgObj.Delete(key)
	}
	if len(*msgObj) > 0 {
		if e.Request.Type == Empty {
			e.Request.Type = &Object{}
		}
		reqObj := AsObject(e.Request.Type)
		for _, nat := range *msgObj {
			key, attr := objectAttribute(reqObj, nat.Name)
			if attr == nil {
				key = nat.Name
				reqObj.Set(key, nat.Attribute)
			}
			if e.MethodExpr.Payload.IsRequired(nat.Name) {
				e.Request.Validation.AddRequired(key)
			}
		}
	}
	for _, nat := range *AsObject(e.Request.Type) {
		_, attr := objectAttribute(pobj, nat.Name)
		patt := DupAtt(attr)
		initAttrFromDesign(nat.Attribute, patt)
		if nat.Attribute.Meta == nil {
			nat.Attribute.Meta = patt.Meta
		} else {
			nat.Attribute.Meta.Merge(patt.Meta)
		}
	}
}

func (e *GRPCEndpointExpr) propagatePayloadProtoStructName() {
	ut, ok := e.MethodExpr.Payload.Type.(UserType)
	if !ok {
		return
	}
	proto, ok := ut.Attribute().Meta.Last("struct:name:proto")
	if !ok {
		return
	}
	if e.Request.Meta == nil {
		e.Request.Meta = ut.Attribute().Meta
	} else {
		e.Request.Meta["struct:name:proto"] = []string{proto}
	}
}

func (e *GRPCEndpointExpr) finalizeNonObjectPayload() {
	initAttrFromDesign(e.Request, e.MethodExpr.Payload)
}

func (e *GRPCEndpointExpr) finalizeStreamingRequest() {
	if e.MethodExpr.StreamingPayload.Type == Empty {
		return
	}
	attr := e.MethodExpr.StreamingPayload
	if ut, ok := attr.Type.(UserType); ok {
		attr = ut.Attribute()
	}
	initAttrFromDesign(e.StreamingRequest, attr)
	// Prepare creates a non-nil message validation before finalization, so
	// initAttrFromDesign cannot inherit the constraints of the unwrapped type.
	if attr.Validation != nil {
		e.StreamingRequest.Validation.Merge(attr.Validation)
	}
	if msgObj := AsObject(e.StreamingRequest.Type); msgObj != nil {
		for _, nat := range *msgObj {
			if e.MethodExpr.StreamingPayload.IsRequired(nat.Name) {
				e.StreamingRequest.Validation.AddRequired(nat.Name)
			}
		}
	}
}
