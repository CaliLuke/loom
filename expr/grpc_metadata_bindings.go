package expr

import (
	"strings"

	"github.com/CaliLuke/loom/eval"
)

func normalizeGRPCMetadata(mapped *MappedAttributeExpr, source *AttributeExpr, channel string, parent eval.Expression, errors *eval.ValidationErrors) *MappedAttributeExpr {
	result := NewMappedAttributeExpr(mapped.AttributeExpr)
	keys := make(map[string]string)
	for _, field := range *AsObject(mapped.Type) {
		_, attribute := source.FindAttribute(field.Name)
		if attribute == nil {
			errors.Add(parent, "%s attribute %q is not found in its source", channel, field.Name)
			continue
		}
		wire := strings.ToLower(mapped.ElemName(field.Name))
		if grpcBinaryMetadataType(attribute.Type) && !strings.HasSuffix(wire, "-bin") {
			wire += "-bin"
		}
		if previous, found := keys[wire]; found && previous != field.Name {
			errors.Add(parent, "%s attributes %q and %q share canonical key %q", channel, previous, field.Name, wire)
		}
		keys[wire] = field.Name
		result.Map(wire, field.Name)
	}
	return result
}

func grpcBinaryMetadataType(dataType DataType) bool {
	dataType = unalias(dataType)
	if array, ok := dataType.(*Array); ok {
		dataType = unalias(array.ElemType.Type)
	}
	return dataType == Bytes
}

// ensureMetadataBindings resolves all wire names once, including implicit
// credentials, before validation and every generated metadata consumer.
func (e *GRPCEndpointExpr) ensureMetadataBindings() *eval.ValidationErrors {
	if e.metadataBindingErrors != nil {
		return e.metadataBindingErrors
	}
	errors := new(eval.ValidationErrors)
	e.metadataBindingErrors = errors
	e.Metadata = DupMappedAtt(e.Metadata)
	if payload := AsObject(e.MethodExpr.Payload.Type); payload != nil {
		e.bindMetadataSecurity(func(field, wire string) {
			e.addPayloadFieldToMetadata(payload, field, wire)
		})
	}
	original := e.Metadata
	e.Metadata = normalizeGRPCMetadata(original, e.MethodExpr.Payload, "request metadata", e, errors)
	for _, requirement := range e.Requirements {
		for _, scheme := range requirement.Schemes {
			if scheme.In == "metadata" && scheme.Name != "" {
				if field, found := original.attributeOfElem(scheme.Name); found {
					scheme.Name = e.Metadata.ElemName(field)
				}
			}
		}
	}
	e.normalizeResponseMetadata(e.Response, e.MethodExpr.Result, errors)
	for _, mapping := range e.GRPCErrors {
		if source := mapping.sourceError(); source != nil {
			e.normalizeResponseMetadata(mapping.Response, source.AttributeExpr, errors)
		}
	}
	return errors
}

func (e *GRPCEndpointExpr) normalizeResponseMetadata(response *GRPCResponseExpr, source *AttributeExpr, errors *eval.ValidationErrors) {
	response.Headers = normalizeGRPCMetadata(response.Headers, source, "response header metadata", e, errors)
	response.Trailers = normalizeGRPCMetadata(response.Trailers, source, "response trailer metadata", e, errors)
}

func (e *GRPCEndpointExpr) addPayloadFieldToMetadata(pobj *Object, field string, transportName string) {
	_, attr := objectAttribute(pobj, field)
	if attr == nil {
		return
	}
	e.Metadata.Type.(*Object).Set(field, DupAtt(attr))
	if transportName != "" {
		e.Metadata.Map(transportName, field)
	}
	if e.MethodExpr.Payload.IsRequired(field) {
		e.Metadata.Validation.AddRequired(field)
	}
}

func (e *GRPCEndpointExpr) bindMetadataSecurity(addToMetadata func(string, string)) {
	requirements := e.MethodExpr.EffectiveRequirements()
	reqLen := len(requirements)
	if reqLen == 0 {
		return
	}
	e.Requirements = make([]*SecurityExpr, 0, reqLen)
	for _, req := range requirements {
		dupReq := DupRequirement(req)
		for _, sch := range dupReq.Schemes {
			e.bindMetadataSecurityScheme(sch, addToMetadata)
		}
		e.Requirements = append(e.Requirements, dupReq)
	}
}

func (e *GRPCEndpointExpr) bindMetadataSecurityScheme(sch *SchemeExpr, addToMetadata func(string, string)) {
	switch sch.Kind {
	case NoKind:
		return
	case BasicAuthKind:
		e.bindMetadataBasicAuthScheme(sch, addToMetadata)
		return
	case APIKeyKind:
		e.bindMetadataCredentialScheme(TaggedAttribute(e.MethodExpr.Payload, "security:apikey:"+sch.SchemeName), sch, addToMetadata)
	case JWTKind:
		e.bindMetadataCredentialScheme(TaggedAttribute(e.MethodExpr.Payload, "security:token"), sch, addToMetadata)
	case OAuth2Kind:
		e.bindMetadataCredentialScheme(TaggedAttribute(e.MethodExpr.Payload, "security:accesstoken"), sch, addToMetadata)
	}
}

func (e *GRPCEndpointExpr) bindMetadataBasicAuthScheme(sch *SchemeExpr, addToMetadata func(string, string)) {
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

func (e *GRPCEndpointExpr) bindMetadataCredentialScheme(field string, sch *SchemeExpr, addToMetadata func(string, string)) {
	sch.Name, sch.In = findKey(e, field)
	if sch.Name == "" {
		sch.Name = "authorization"
		addToMetadata(field, sch.Name)
	}
}
