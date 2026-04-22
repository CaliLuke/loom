package expr

import (
	"fmt"
	"slices"

	"github.com/CaliLuke/loom/eval"
)

type (
	// GRPCEndpointExpr describes a gRPC endpoint. It embeds a MethodExpr
	// and adds gRPC specific properties.
	GRPCEndpointExpr struct {
		eval.DSLFunc
		// MethodExpr is the underlying method expression.
		MethodExpr *MethodExpr
		// Service is the parent service.
		Service *GRPCServiceExpr
		// Request is the message passed to the gRPC method.
		Request *AttributeExpr
		// StreamingRequest is the message passed to the gRPC method through a
		// stream.
		StreamingRequest *AttributeExpr
		// Responses is the success gRPC response from the method.
		Response *GRPCResponseExpr
		// GRPCErrors is the list of all the possible error gRPC responses.
		GRPCErrors []*GRPCErrorExpr
		// Metadata is the metadata to be sent in a gRPC request.
		Metadata *MappedAttributeExpr
		// Requirements is the list of security requirements for the gRPC endpoint.
		Requirements []*SecurityExpr
		// Meta is a set of key/value pairs with semantic that is
		// specific to each generator, see dsl.Meta.
		Meta MetaExpr
	}
)

// Name of gRPC endpoint
func (e *GRPCEndpointExpr) Name() string {
	return e.MethodExpr.Name
}

// Description of gRPC endpoint
func (e *GRPCEndpointExpr) Description() string {
	return e.MethodExpr.Description
}

// EvalName returns the generic expression name used in error messages.
func (e *GRPCEndpointExpr) EvalName() string {
	var prefix, suffix string
	if e.Name() != "" {
		suffix = fmt.Sprintf("gRPC endpoint %#v", e.Name())
	} else {
		suffix = "unnamed gRPC endpoint"
	}
	if e.Service != nil {
		prefix = e.Service.EvalName() + " "
	}
	return prefix + suffix
}

// Prepare initializes the Request and Response if nil.
func (e *GRPCEndpointExpr) Prepare() {
	e.Request = ensureValidatedAttribute(e.Request)
	e.StreamingRequest = ensureValidatedAttribute(e.StreamingRequest)
	e.Metadata = ensureValidatedMappedAttribute(e.Metadata)
	e.Response = ensureGRPCResponse(e.Response)
	e.Response.Prepare()
	methodErrors := grpcMethodErrorNames(e.GRPCErrors)
	e.appendMethodGRPCErrors(methodErrors)
	e.appendServiceGRPCErrors(methodErrors)
	e.prepareGRPCErrorResponses()
}

// Validate validates the endpoint expression by checking if the request
// and responses contains the "rpc:tag" in the meta. It also makes sure
// that there is only one response per status code.
func (e *GRPCEndpointExpr) Validate() error {
	verr := new(eval.ValidationErrors)
	if e.Name() == "" {
		verr.Add(e, "Endpoint name cannot be empty")
	}
	e.validateGRPCUnionShapes(verr)
	verr.Merge(e.validateRequestShape())
	verr.Merge(e.Response.Validate(e))
	verr.Merge(e.validateGRPCErrors())
	return verr
}

func ensureValidatedAttribute(att *AttributeExpr) *AttributeExpr {
	if att == nil {
		att = &AttributeExpr{Type: Empty}
	}
	if att.Validation == nil {
		att.Validation = &ValidationExpr{}
	}
	return att
}

func ensureValidatedMappedAttribute(att *MappedAttributeExpr) *MappedAttributeExpr {
	if att == nil {
		att = NewEmptyMappedAttributeExpr()
	}
	if att.Validation == nil {
		att.Validation = &ValidationExpr{}
	}
	return att
}

func ensureGRPCResponse(resp *GRPCResponseExpr) *GRPCResponseExpr {
	if resp == nil {
		return &GRPCResponseExpr{StatusCode: 0}
	}
	return resp
}

func grpcMethodErrorNames(errors []*GRPCErrorExpr) map[string]struct{} {
	names := map[string]struct{}{}
	for _, err := range errors {
		names[err.Name] = struct{}{}
	}
	return names
}

func (e *GRPCEndpointExpr) appendMethodGRPCErrors(methodErrors map[string]struct{}) {
	for _, methodErr := range e.MethodExpr.Errors {
		if _, ok := methodErrors[methodErr.Name]; ok {
			continue
		}
		methodErrors[methodErr.Name] = struct{}{}
		if resp := dupNamedGRPCError(methodErr.Name, e.Service.GRPCErrors); resp != nil {
			e.GRPCErrors = append(e.GRPCErrors, resp)
			continue
		}
		if resp := dupNamedGRPCError(methodErr.Name, Root.API.GRPC.Errors); resp != nil {
			e.GRPCErrors = append(e.GRPCErrors, resp)
		}
	}
}

func (e *GRPCEndpointExpr) appendServiceGRPCErrors(methodErrors map[string]struct{}) {
	for _, serviceErr := range e.Service.ServiceExpr.Errors {
		if _, ok := methodErrors[serviceErr.Name]; ok {
			continue
		}
		if resp := dupNamedGRPCError(serviceErr.Name, e.Service.GRPCErrors); resp != nil {
			e.GRPCErrors = append(e.GRPCErrors, resp)
			continue
		}
		if resp := dupNamedGRPCError(serviceErr.Name, Root.API.GRPC.Errors); resp != nil {
			e.GRPCErrors = append(e.GRPCErrors, resp)
		}
	}
}

func dupNamedGRPCError(name string, errors []*GRPCErrorExpr) *GRPCErrorExpr {
	for _, err := range errors {
		if err.Name == name {
			return err.Dup()
		}
	}
	return nil
}

func (e *GRPCEndpointExpr) prepareGRPCErrorResponses() {
	for _, err := range e.GRPCErrors {
		err.Response.Prepare()
	}
}

func (e *GRPCEndpointExpr) validateGRPCUnionShapes(verr *eval.ValidationErrors) {
	seenUnions := make(map[*Union]struct{})
	seenAttrs := make(map[*AttributeExpr]struct{})
	validateGRPCUnionShapes(e.MethodExpr.Payload, e.MethodExpr, verr, seenUnions, seenAttrs)
	validateGRPCUnionShapes(e.MethodExpr.StreamingPayload, e.MethodExpr, verr, seenUnions, seenAttrs)
	validateGRPCUnionShapes(e.MethodExpr.Result, e.MethodExpr, verr, seenUnions, seenAttrs)
	validateGRPCUnionShapes(e.MethodExpr.StreamingResult, e.MethodExpr, verr, seenUnions, seenAttrs)
	for _, err := range e.MethodExpr.Errors {
		validateGRPCUnionShapes(err.AttributeExpr, e.MethodExpr, verr, seenUnions, seenAttrs)
	}
}

func (e *GRPCEndpointExpr) validateRequestShape() *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	hasMessage, hasMetadata := e.validateRequestComponents(verr)
	payloadObj := AsObject(e.MethodExpr.Payload.Type)
	if payloadObj == nil {
		if hasMessage && hasMetadata {
			verr.Add(e, "Both request message and metadata are defined, but payload is not an object. Define either metadata or message or make payload an object type.")
		}
		return verr
	}
	verr.Merge(e.validateRequestObjectUsage(payloadObj, hasMessage, hasMetadata))
	return verr
}

func (e *GRPCEndpointExpr) validateRequestComponents(verr *eval.ValidationErrors) (bool, bool) {
	hasMessage := e.Request.Type != Empty
	if hasMessage {
		verr.Merge(e.Request.Validate("gRPC request message", e))
		verr.Merge(validateMessage(e.Request, e.MethodExpr.Payload, e, true))
	}
	hasMetadata := !e.Metadata.IsEmpty()
	if hasMetadata {
		verr.Merge(e.Metadata.Validate("gRPC request metadata", e))
		verr.Merge(validateMetadata(e.Metadata, e.MethodExpr.Payload, e, true))
	}
	return hasMessage, hasMetadata
}

func (e *GRPCEndpointExpr) validateRequestObjectUsage(payloadObj *Object, hasMessage, hasMetadata bool) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	switch {
	case hasMessage && hasMetadata:
		verr.Merge(e.validateDistinctRequestMessageAndMetadata())
	case !hasMessage && !hasMetadata:
		verr.Merge(e.validateImplicitRequestRPCTags(payloadObj))
	}
	return verr
}

func (e *GRPCEndpointExpr) validateDistinctRequestMessageAndMetadata() *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	msgObj := AsObject(e.Request.Type)
	metObj := AsObject(e.Metadata.Type)
	for _, msgnat := range *msgObj {
		for _, metnat := range *metObj {
			if metnat.Name == msgnat.Name {
				verr.Add(e, "Attribute %q defined in both request message and metadata. Define the attribute in either message or metadata.", metnat.Name)
				break
			}
		}
	}
	return verr
}

func (e *GRPCEndpointExpr) validateImplicitRequestRPCTags(payloadObj *Object) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	msgFields := nonSecurityRPCFields(payloadObj, getSecurityAttributes(e.MethodExpr))
	if len(*msgFields) > 0 {
		verr.Merge(validateRPCTags(msgFields, e))
	}
	return verr
}

func nonSecurityRPCFields(payloadObj *Object, securityAttrs []string) *Object {
	if len(securityAttrs) == 0 {
		return payloadObj
	}
	msgFields := &Object{}
	for _, nat := range *payloadObj {
		if !slices.Contains(securityAttrs, nat.Name) {
			msgFields.Set(nat.Name, nat.Attribute)
		}
	}
	return msgFields
}

func (e *GRPCEndpointExpr) validateGRPCErrors() *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	for _, err := range e.GRPCErrors {
		verr.Merge(err.Validate())
	}
	return verr
}

func validateGRPCUnionShapes(att *AttributeExpr, parent eval.Expression, verr *eval.ValidationErrors, seenUnions map[*Union]struct{}, seenAttrs map[*AttributeExpr]struct{}) {
	if att == nil || att.Type == nil {
		return
	}
	if _, ok := seenAttrs[att]; ok {
		return
	}
	seenAttrs[att] = struct{}{}

	if u := AsUnion(att.Type); u != nil {
		if _, ok := seenUnions[u]; ok {
			return
		}
		seenUnions[u] = struct{}{}
		for _, ut := range u.Values {
			switch {
			case IsArray(ut.Attribute.Type):
				verr.Add(parent, "union type %s has array elements, not supported by gRPC", u.Name())
			case IsMap(ut.Attribute.Type):
				verr.Add(parent, "union type %s has map elements, not supported by gRPC", u.Name())
			}
			validateGRPCUnionShapes(ut.Attribute, parent, verr, seenUnions, seenAttrs)
		}
		return
	}

	if o := AsObject(att.Type); o != nil {
		for _, nat := range *o {
			validateGRPCUnionShapes(nat.Attribute, parent, verr, seenUnions, seenAttrs)
		}
		return
	}

	if ar := AsArray(att.Type); ar != nil {
		validateGRPCUnionShapes(ar.ElemType, parent, verr, seenUnions, seenAttrs)
		return
	}

	if m := AsMap(att.Type); m != nil {
		validateGRPCUnionShapes(m.KeyType, parent, verr, seenUnions, seenAttrs)
		validateGRPCUnionShapes(m.ElemType, parent, verr, seenUnions, seenAttrs)
		return
	}
}

// validateMessage validates the gRPC message. It compares the given message
// with the service type (Payload or Result) and ensures all the attributes
// defined in the message type are found in the service type and the attributes
// are set with unique "rpc:tag" numbers.
//
// msgAtt is the Request/Response message attribute. validateMessage assumes
// that the msgAtt is not Empty.
// serviceAtt is the Payload/Result attribute.
// e is the endpoint expression.
// req if true indicates the Request message is being validated.
func validateMessage(msgAtt, serviceAtt *AttributeExpr, e *GRPCEndpointExpr, req bool) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	msgKind := "Response"
	serviceKind := "Result"
	if req {
		msgKind = "Request"
		serviceKind = "Payload"
	}
	if isEmpty(serviceAtt) {
		verr.Add(e, "%s message is defined but %s is not defined in method", msgKind, serviceKind)
		return verr
	}

	if !IsObject(serviceAtt.Type) {
		// service type (payload or result) is a primitive, array, or map
		// The message type must have at most one field and that field must be
		// of the same type as the service type.
		msgObj := AsObject(msgAtt.Type)
		if flen := len(*msgObj); flen != 1 {
			verr.Add(e, "%s is not an object type. %s message should have at most 1 field. Got %d.", serviceKind, msgKind, flen)
		} else {
			for _, f := range *msgObj {
				if f.Attribute.Type != serviceAtt.Type {
					verr.Add(e, "%s message field %q is %q type but the %s type is %q.", msgKind, f.Name, f.Attribute.Type.Name(), serviceKind, serviceAtt.Type.Name())
				}
			}
		}
	} else {
		// service type is an object. Verify the attributes defined in the
		// message are found in the service type.
		// msgFields will contain the attributes from the service type that has the
		// same name as the message attributes so that we can validate the
		// rpc:tag in the meta.
		msgFields := &Object{}
		for _, nat := range *AsObject(msgAtt.Type) {
			if a := serviceAtt.Find(nat.Name); a != nil {
				msgFields.Set(nat.Name, a)
				break
			}
			verr.Add(e, "%s message attribute %q is not found in %s", msgKind, nat.Name, serviceKind)
		}
		// validate rpc:tag in meta for the message fields
		verr.Merge(validateRPCTags(msgFields, e))
	}
	return verr
}

// validateRPCTags verifies whether every attribute in the object type has
// "rpc:tag" set in the meta and the tag numbers are unique.
func validateRPCTags(fields *Object, e *GRPCEndpointExpr) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	foundRPC := make(map[string]string)
	for _, nat := range *fields {
		if union := AsUnion(nat.Attribute.Type); union != nil {
			for _, branch := range union.Values {
				tag, ok := branch.Attribute.FieldTag()
				if !ok {
					continue
				}
				name := nat.Name + "." + branch.Name
				if a, ok := foundRPC[tag]; ok {
					verr.Add(e, "field number %s in attribute %q already exists for attribute %q", tag, name, a)
				} else {
					foundRPC[tag] = name
				}
			}
			continue
		}
		if tag, ok := nat.Attribute.FieldTag(); !ok {
			verr.Add(e, "attribute %q does not have \"rpc:tag\" defined in the meta, use \"Field\" to define the attribute of a type used in a gRPC method", nat.Name)
		} else if a, ok := foundRPC[tag]; ok {
			verr.Add(e, "field number %s in attribute %q already exists for attribute %q", tag, nat.Name, a)
		} else {
			foundRPC[tag] = nat.Name
		}
	}
	return verr
}

// validateMetadata validates the gRPC metadata. It compares the given metadata
// with the service type (Payload or Result) and ensures all the attributes
// defined in the metadata type are found in the service type.
//
// metAtt is the Request/Response metadata attribute. validateMetadata assumes
// that the metAtt is not Empty.
// serviceAtt is the Payload/Result attribute.
// e is the endpoint expression.
// req if true indicates the Request metadata is being validated.
func validateMetadata(metAtt *MappedAttributeExpr, serviceAtt *AttributeExpr, e *GRPCEndpointExpr, req bool) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	metKind := "Response"
	serviceKind := "Result"
	if req {
		metKind = "Request"
		serviceKind = "Payload"
	}
	if isEmpty(serviceAtt) {
		verr.Add(e, "%s metadata is defined but %s is not defined in method", metKind, serviceKind)
		return verr
	}
	if IsObject(serviceAtt.Type) {
		// service type is an object type. Ensure the attributes defined in
		// the metadata are found in the service type.
		for _, nat := range *AsObject(metAtt.Type) {
			if a := serviceAtt.Find(nat.Name); a == nil {
				verr.Add(e, "%s metadata attribute %q is not found in %s", metKind, nat.Name, serviceKind)
			}
		}
	} else {
		verr.Add(e, "%s metadata is defined but method %s is not an object type", metKind, serviceKind)
	}
	return verr
}

// getSecurityAttributes returns the attributes that describes a security
// scheme from a method expression.
func getSecurityAttributes(m *MethodExpr) []string {
	var secAttrs []string

	for _, req := range m.validationRequirements() {
		for _, sch := range req.Schemes {
			switch sch.Kind {
			case BasicAuthKind:
				if field := TaggedAttribute(m.Payload, "security:username"); field != "" {
					secAttrs = append(secAttrs, field)
				}
				if field := TaggedAttribute(m.Payload, "security:password"); field != "" {
					secAttrs = append(secAttrs, field)
				}
			case APIKeyKind:
				if field := TaggedAttribute(m.Payload, "security:apikey:"+sch.SchemeName); field != "" {
					secAttrs = append(secAttrs, field)
				}
			case JWTKind:
				if field := TaggedAttribute(m.Payload, "security:token"); field != "" {
					secAttrs = append(secAttrs, field)
				}
			case OAuth2Kind:
				if field := TaggedAttribute(m.Payload, "security:accesstoken"); field != "" {
					secAttrs = append(secAttrs, field)
				}
			}
		}
	}
	return secAttrs
}
