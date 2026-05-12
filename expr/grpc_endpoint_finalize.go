package expr

// Finalize ensures the request and response attributes are initialized.
func (e *GRPCEndpointExpr) Finalize() {
	if pobj := AsObject(e.MethodExpr.Payload.Type); pobj != nil {
		addToMetadata := func(field string, tName string) {
			e.addPayloadFieldToMetadata(pobj, field, tName)
		}
		e.finalizeSecurityRequirements(addToMetadata)
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

func (e *GRPCEndpointExpr) addPayloadFieldToMetadata(pobj *Object, field string, transportName string) {
	attr := pobj.Attribute(field)
	e.Metadata.Type.(*Object).Set(field, attr)
	if transportName != "" {
		e.Metadata.Map(transportName, field)
	}
	if e.MethodExpr.Payload.IsRequired(field) {
		e.Metadata.Validation.AddRequired(field)
	}
}

func (e *GRPCEndpointExpr) finalizeSecurityRequirements(addToMetadata func(string, string)) {
	reqLen := len(e.MethodExpr.Requirements)
	if reqLen == 0 {
		return
	}
	e.Requirements = make([]*SecurityExpr, 0, reqLen)
	for _, req := range e.MethodExpr.Requirements {
		dupReq := DupRequirement(req)
		for _, sch := range dupReq.Schemes {
			e.finalizeSecurityScheme(sch, addToMetadata)
		}
		e.Requirements = append(e.Requirements, dupReq)
	}
}

func (e *GRPCEndpointExpr) finalizeSecurityScheme(sch *SchemeExpr, addToMetadata func(string, string)) {
	switch sch.Kind {
	case NoKind:
		return
	case BasicAuthKind:
		e.finalizeBasicAuthScheme(sch, addToMetadata)
		return
	case APIKeyKind:
		e.finalizeCredentialScheme(TaggedAttribute(e.MethodExpr.Payload, "security:apikey:"+sch.SchemeName), sch, addToMetadata)
	case JWTKind:
		e.finalizeCredentialScheme(TaggedAttribute(e.MethodExpr.Payload, "security:token"), sch, addToMetadata)
	case OAuth2Kind:
		e.finalizeCredentialScheme(TaggedAttribute(e.MethodExpr.Payload, "security:accesstoken"), sch, addToMetadata)
	}
}

func (e *GRPCEndpointExpr) finalizeBasicAuthScheme(sch *SchemeExpr, addToMetadata func(string, string)) {
	for _, field := range []string{
		TaggedAttribute(e.MethodExpr.Payload, "security:username"),
		TaggedAttribute(e.MethodExpr.Payload, "security:password"),
	} {
		sch.Name, sch.In = findKey(e, field)
		if sch.Name == "" {
			addToMetadata(field, "")
		}
	}
}

func (e *GRPCEndpointExpr) finalizeCredentialScheme(field string, sch *SchemeExpr, addToMetadata func(string, string)) {
	sch.Name, sch.In = findKey(e, field)
	if sch.Name == "" {
		sch.Name = "authorization"
		addToMetadata(field, sch.Name)
	}
}

func (e *GRPCEndpointExpr) finalizeRequestMessageFromPayload(pobj *Object) {
	msgObj := Dup(pobj).(*Object)
	for _, nat := range *AsObject(e.Metadata.Type) {
		initAttrFromDesign(nat.Attribute, pobj.Attribute(nat.Name))
		if e.MethodExpr.Payload.IsRequired(nat.Name) {
			e.Metadata.Validation.AddRequired(nat.Name)
		}
		msgObj.Delete(nat.Name)
	}
	if len(*msgObj) > 0 {
		if e.Request.Type == Empty {
			e.Request.Type = &Object{}
		}
		reqObj := AsObject(e.Request.Type)
		for _, nat := range *msgObj {
			if reqObj.Attribute(nat.Name) == nil {
				reqObj.Set(nat.Name, nat.Attribute)
			}
			if e.MethodExpr.Payload.IsRequired(nat.Name) {
				e.Request.Validation.AddRequired(nat.Name)
			}
		}
	}
	for _, nat := range *AsObject(e.Request.Type) {
		patt := DupAtt(pobj.Attribute(nat.Name))
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
	if msgObj := AsObject(e.StreamingRequest.Type); msgObj != nil {
		for _, nat := range *msgObj {
			if e.MethodExpr.StreamingPayload.IsRequired(nat.Name) {
				e.StreamingRequest.Validation.AddRequired(nat.Name)
			}
		}
	}
}
