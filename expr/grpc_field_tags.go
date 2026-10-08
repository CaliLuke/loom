package expr

import "github.com/CaliLuke/loom/eval"

// validateMessageFieldTags checks precisely the values emitted as messages:
// ordinary and streaming inputs, results, and declared error details. Metadata
// and implicitly mapped credentials are not protobuf fields.
func (e *GRPCEndpointExpr) validateMessageFieldTags() *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	var implicitCredentials []string
	for _, name := range getSecurityAttributes(e.MethodExpr) {
		if key, _ := findKey(e, name); key == "" {
			implicitCredentials = append(implicitCredentials, name)
		}
	}
	request := rpcMessageAttribute(e.MethodExpr.Payload, implicitCredentials, e.Metadata)
	verr.Merge(validateRPCMessageTags(request, e))
	verr.Merge(validateRPCMessageTags(e.MethodExpr.StreamingPayload, e))
	verr.Merge(validateRPCMessageTags(rpcMessageAttribute(e.MethodExpr.Result, nil, e.Response.Headers, e.Response.Trailers), e))
	for _, mapping := range e.GRPCErrors {
		if source := mapping.sourceError(); source != nil && source.Type != ErrorResult && IsObject(source.Type) {
			verr.Merge(validateRPCMessageTags(rpcMessageAttribute(source.AttributeExpr, nil, mapping.Response.Headers, mapping.Response.Trailers), e))
		}
	}
	return verr
}

// rpcMessageAttribute selects fields using the same metadata ownership as
// message finalization. The new object slice leaves the authored graph intact.
func rpcMessageAttribute(att *AttributeExpr, excluded []string, metadata ...*MappedAttributeExpr) *AttributeExpr {
	if att == nil {
		return nil
	}
	// Keep named wrappers until the traversal observes an opaque mapping on
	// either the occurrence or its underlying type.
	for current := att; current != nil; {
		if len(current.Meta["struct:field:proto"]) > 0 {
			return att
		}
		named, ok := current.Type.(UserType)
		if !ok {
			break
		}
		current = named.Attribute()
	}
	fields := AsObject(att.Type)
	if fields == nil {
		for _, mapped := range metadata {
			if !mapped.IsEmpty() {
				return &AttributeExpr{Type: Empty}
			}
		}
		return att
	}
	selected := append(Object{}, (*fields)...)
	for _, name := range excluded {
		selected.Delete(name)
	}
	for _, mapped := range metadata {
		if obj := AsObject(mapped.Type); obj != nil {
			for _, field := range *obj {
				key, _ := objectAttribute(fields, field.Name)
				selected.Delete(key)
			}
		}
	}
	return &AttributeExpr{Type: &selected, Meta: att.Meta}
}

// validateRPCTags validates the selected message and every message reachable
// through its fields. Field numbers belong to each object, never to the graph.
func validateRPCTags(fields *Object, e *GRPCEndpointExpr) *eval.ValidationErrors {
	return validateRPCMessageTags(&AttributeExpr{Type: fields}, e)
}

func validateRPCMessageTags(att *AttributeExpr, e *GRPCEndpointExpr) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	seen := make(map[*AttributeExpr]bool)
	var visit func(*AttributeExpr)
	visit = func(att *AttributeExpr) {
		if att == nil || seen[att] || len(att.Meta["struct:field:proto"]) > 0 {
			return
		}
		seen[att] = true
		switch dt := att.Type.(type) {
		case UserType:
			visit(dt.Attribute())
		case *Object:
			verr.Merge(validateRPCObjectTags(dt, e))
			for _, field := range *dt {
				visit(field.Attribute)
			}
		case *Array:
			visit(dt.ElemType)
		case *Map:
			visit(dt.KeyType)
			visit(dt.ElemType)
		case *Union:
			for _, branch := range dt.Values {
				visit(branch.Attribute)
			}
		}
	}
	visit(att)
	return verr
}

// validateRPCObjectTags verifies whether every attribute in the object type and
// every branch of its union attributes has a field tag and the tag numbers
// are unique. Branches of a union passed to Field take the tags that
// UnionFieldTags derives from the field tag.
func validateRPCObjectTags(fields *Object, e *GRPCEndpointExpr) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	owners := make(map[string]rpcTagOwner)
	claim := func(tag, name string, derived bool) {
		owner, ok := owners[tag]
		if !ok {
			owners[tag] = rpcTagOwner{name: name, derived: derived}
			return
		}
		hint := ""
		if derived || owner.derived {
			hint = "; a OneOf passed to Field numbers its branches consecutively from the field number"
		}
		verr.Add(e, "field number %s in attribute %q already exists for attribute %q%s", tag, name, owner.name, hint)
	}
	for _, nat := range *fields {
		if union := AsUnion(nat.Attribute.Type); union != nil {
			for i, tag := range nat.Attribute.UnionFieldTags() {
				branch := union.Values[i]
				if tag == "" {
					verr.Add(e, "union branch %q of attribute %q does not have \"rpc:tag\" defined in the meta, use \"Field\" to define each branch of a OneOf block or pass the OneOf to \"Field\" to number its branches", branch.Name, nat.Name)
					continue
				}
				_, explicit := branch.Attribute.FieldTag()
				claim(tag, nat.Name+"."+branch.Name, !explicit)
			}
			continue
		}
		tag, ok := nat.Attribute.FieldTag()
		if !ok {
			verr.Add(e, "attribute %q does not have \"rpc:tag\" defined in the meta, use \"Field\" to define the attribute of a type used in a gRPC method", nat.Name)
			continue
		}
		claim(tag, nat.Name, false)
	}
	return verr
}
