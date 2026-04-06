package expr

import (
	"fmt"

	"github.com/CaliLuke/loom/eval"
)

type (
	// GRPCResponseExpr defines a gRPC response including its status code, result
	// type, and metadata.
	GRPCResponseExpr struct {
		// gRPC status code
		StatusCode int
		// Response description
		Description string
		// Response Message if any
		Message *AttributeExpr
		// Parent expression, one of EndpointExpr, ServiceExpr or
		// RootExpr.
		Parent eval.Expression
		// Headers is the header metadata to be sent in the gRPC response.
		Headers *MappedAttributeExpr
		// Trailers is the trailer metadata to be sent in the gRPC response.
		Trailers *MappedAttributeExpr
		// Meta is a list of key/value pairs.
		Meta MetaExpr
	}
)

// EvalName returns the generic definition name used in error messages.
func (r *GRPCResponseExpr) EvalName() string {
	var suffix string
	if r.Parent != nil {
		suffix = fmt.Sprintf(" of %s", r.Parent.EvalName())
	}
	return "gRPC response" + suffix
}

// Prepare makes sure the response message and metadata are initialized.
func (r *GRPCResponseExpr) Prepare() {
	if r.Message == nil {
		r.Message = &AttributeExpr{Type: Empty}
	}
	if r.Message.Validation == nil {
		r.Message.Validation = &ValidationExpr{}
	}
	if r.Headers == nil {
		r.Headers = NewEmptyMappedAttributeExpr()
	}
	if r.Headers.Validation == nil {
		r.Headers.Validation = &ValidationExpr{}
	}
	if r.Trailers == nil {
		r.Trailers = NewEmptyMappedAttributeExpr()
	}
	if r.Trailers.Validation == nil {
		r.Trailers.Validation = &ValidationExpr{}
	}
}

// Validate checks that the response definition is consistent: its status is set
// and the result type definition if any is valid.
func (r *GRPCResponseExpr) Validate(e *GRPCEndpointExpr) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	hasMessage, hasHeaders, hasTrailers := r.validateResponseComponents(verr, e)
	verr.Merge(r.validateResponseShape(e, hasMessage, hasHeaders, hasTrailers))
	return verr
}

// Finalize ensures that the response message type is set. If Message DSL is
// used to set the response message then the message type is set by mapping
// the attributes to the method Result expression. If no response message set
// explicitly, the message is set from the method Result expression.
func (r *GRPCResponseExpr) Finalize(a *GRPCEndpointExpr, svcAtt *AttributeExpr) {
	r.Parent = a

	if svcObj := AsObject(svcAtt.Type); svcObj != nil {
		r.finalizeObjectResponse(svcAtt, svcObj)
	} else {
		r.finalizeScalarResponse(svcAtt)
	}
}

func (r *GRPCResponseExpr) validateResponseComponents(verr *eval.ValidationErrors, e *GRPCEndpointExpr) (bool, bool, bool) {
	hasMessage := r.Message.Type != Empty
	if hasMessage {
		verr.Merge(r.Message.Validate("gRPC response message", r))
		verr.Merge(validateMessage(r.Message, e.MethodExpr.Result, e, false))
	}
	hasHeaders := !r.Headers.IsEmpty()
	if hasHeaders {
		verr.Merge(r.Headers.Validate("gRPC response header metadata", r))
		verr.Merge(validateMetadata(r.Headers, e.MethodExpr.Result, e, false))
	}
	hasTrailers := !r.Trailers.IsEmpty()
	if hasTrailers {
		verr.Merge(r.Trailers.Validate("gRPC response trailer metadata", r))
		verr.Merge(validateMetadata(r.Trailers, e.MethodExpr.Result, e, false))
	}
	return hasMessage, hasHeaders, hasTrailers
}

func (r *GRPCResponseExpr) validateResponseShape(e *GRPCEndpointExpr, hasMessage, hasHeaders, hasTrailers bool) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	if robj := AsObject(e.MethodExpr.Result.Type); robj != nil {
		verr.Merge(r.validateObjectResponseShape(e, robj, hasMessage, hasHeaders, hasTrailers))
		return verr
	}
	verr.Merge(r.validateScalarResponseShape(e, hasMessage, hasHeaders, hasTrailers))
	return verr
}

func (r *GRPCResponseExpr) validateObjectResponseShape(e *GRPCEndpointExpr, robj *Object, hasMessage, hasHeaders, hasTrailers bool) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	switch {
	case hasMessage && hasHeaders:
		verr.Merge(validateDistinctResponseObjects(e, AsObject(r.Message.Type), AsObject(r.Headers.Type), "response message", "header metadata"))
	case hasMessage && hasTrailers:
		verr.Merge(validateDistinctResponseObjects(e, AsObject(r.Message.Type), AsObject(r.Trailers.Type), "response message", "trailer metadata"))
	case hasHeaders && hasTrailers:
		verr.Merge(validateDistinctResponseObjects(e, AsObject(r.Trailers.Type), AsObject(r.Headers.Type), "response trailer metadata", "header metadata"))
	case !hasMessage && !hasHeaders && !hasTrailers:
		verr.Merge(validateRPCTags(robj, e))
	}
	return verr
}

func (r *GRPCResponseExpr) validateScalarResponseShape(e *GRPCEndpointExpr, hasMessage, hasHeaders, hasTrailers bool) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	switch {
	case hasMessage && hasHeaders:
		verr.Add(e, "Both response message and header metadata are defined, but result is not an object. Define either header metadata or message or make result an object type.")
	case hasMessage && hasTrailers:
		verr.Add(e, "Both response message and trailer metadata are defined, but result is not an object. Define either trailer metadata or message or make result an object type.")
	case hasHeaders && hasTrailers:
		verr.Add(e, "Both response header and trailer metadata are defined, but result is not an object. Define either trailer or header metadata or make result an object type.")
	}
	return verr
}

func validateDistinctResponseObjects(e *GRPCEndpointExpr, left, right *Object, leftKind, rightKind string) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	for _, nat := range *left {
		if right.Attribute(nat.Name) != nil {
			verr.Add(e, "Attribute %q defined in both %s and %s. Define the attribute in either %s or %s.", nat.Name, leftKind, rightKind, leftKind, rightKind)
		}
	}
	return verr
}

func (r *GRPCResponseExpr) finalizeObjectResponse(svcAtt *AttributeExpr, svcObj *Object) {
	msgObj := Dup(svcObj).(*Object)
	r.initMetadataFromResult(r.Headers, svcAtt, svcObj, msgObj)
	r.initMetadataFromResult(r.Trailers, svcAtt, svcObj, msgObj)
	r.addResultMessageAttributes(svcAtt, msgObj)
	r.initMessageFromResult(svcObj)
	r.propagateProtoStructName(svcAtt)
}

func (r *GRPCResponseExpr) initMetadataFromResult(mapped *MappedAttributeExpr, svcAtt *AttributeExpr, svcObj, msgObj *Object) {
	for _, nat := range *AsObject(mapped.Type) {
		initAttrFromDesign(nat.Attribute, svcObj.Attribute(nat.Name))
		if svcAtt.IsRequired(nat.Name) {
			mapped.Validation.AddRequired(nat.Name)
		}
		msgObj.Delete(nat.Name)
	}
}

func (r *GRPCResponseExpr) addResultMessageAttributes(svcAtt *AttributeExpr, msgObj *Object) {
	if len(*msgObj) == 0 {
		return
	}
	if r.Message.Type == Empty {
		r.Message.Type = &Object{}
	}
	resObj := AsObject(r.Message.Type)
	for _, nat := range *msgObj {
		if resObj.Attribute(nat.Name) == nil {
			resObj.Set(nat.Name, nat.Attribute)
		}
		if svcAtt.IsRequired(nat.Name) {
			r.Message.Validation.AddRequired(nat.Name)
		}
	}
}

func (r *GRPCResponseExpr) initMessageFromResult(svcObj *Object) {
	for _, nat := range *AsObject(r.Message.Type) {
		svcAtt := DupAtt(svcObj.Attribute(nat.Name))
		initAttrFromDesign(nat.Attribute, svcAtt)
		if nat.Attribute.Meta == nil {
			nat.Attribute.Meta = svcAtt.Meta
		} else {
			nat.Attribute.Meta.Merge(svcAtt.Meta)
		}
	}
}

func (r *GRPCResponseExpr) propagateProtoStructName(svcAtt *AttributeExpr) {
	ut, ok := svcAtt.Type.(UserType)
	if !ok {
		return
	}
	proto, ok := ut.Attribute().Meta.Last("struct:name:proto")
	if !ok {
		return
	}
	if r.Message.Meta == nil {
		r.Message.Meta = ut.Attribute().Meta
		return
	}
	r.Message.Meta["struct:name:proto"] = []string{proto}
}

func (r *GRPCResponseExpr) finalizeScalarResponse(svcAtt *AttributeExpr) {
	switch {
	case !r.Headers.IsEmpty():
		initAttrFromDesign(r.Headers.AttributeExpr, svcAtt)
	case !r.Trailers.IsEmpty():
		initAttrFromDesign(r.Trailers.AttributeExpr, svcAtt)
	default:
		initAttrFromDesign(r.Message, svcAtt)
	}
}

// Dup creates a copy of the response expression.
func (r *GRPCResponseExpr) Dup() *GRPCResponseExpr {
	return &GRPCResponseExpr{
		StatusCode:  r.StatusCode,
		Description: r.Description,
		Parent:      r.Parent,
		Meta:        r.Meta,
		Message:     DupAtt(r.Message),
		Headers:     NewMappedAttributeExpr(r.Headers.Attribute()),
		Trailers:    NewMappedAttributeExpr(r.Trailers.Attribute()),
	}
}
