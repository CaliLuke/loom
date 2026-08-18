package codegen

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func (sds *ServicesData) collectEndpointBodyAttributeTypes(endpointIR *transportir.Endpoint, sd *ServiceData) {
	unionBranchTypes := make(map[string]struct{})
	collectUnionBranchUserTypes(endpointIR.Request.RawBody, unionBranchTypes)
	if endpointIR.Stream.RequestPayload != nil && endpointIR.Stream.RequestPayload.Type != expr.Empty {
		collectUnionBranchUserTypes(endpointIR.Request.StreamingBody, unionBranchTypes)
	}
	for _, response := range endpointIR.Response.Responses {
		collectUnionBranchUserTypes(response.Body, unionBranchTypes)
	}
	for _, response := range endpointIR.Response.ErrorResponses {
		collectUnionBranchUserTypes(response.Body, unionBranchTypes)
	}
	ensureUnionBranchValidator := func(data *TypeData, userType expr.UserType) {
		if data == nil || data.ValidateDef != "" {
			return
		}
		if _, ok := unionBranchTypes[userType.ID()]; ok {
			data.ValidateDef = "// no validations"
			data.ValidateRef = fmt.Sprintf("err = Validate%s(v)", data.VarName)
		}
	}

	appendTypeData := func(att *expr.AttributeExpr, ptr, server, jsonPresence bool, target *[]*TypeData) {
		collectUserTypes(att.Type, func(ut expr.UserType) {
			if d := sds.attributeTypeData(ut, true, ptr, server, jsonPresence, sd); d != nil {
				ensureUnionBranchValidator(d, ut)
				*target = append(*target, d)
			}
		})
	}
	requestJSONPresence := !endpointIR.Request.FormEncoded && !endpointIR.Request.Multipart
	appendTypeData(endpointIR.Request.RawBody, true, true, requestJSONPresence, &sd.ServerBodyAttributeTypes)
	appendTypeData(endpointIR.Request.RawBody, false, false, false, &sd.ClientBodyAttributeTypes)

	if endpointIR.Stream.RequestPayload != nil && endpointIR.Stream.RequestPayload.Type != expr.Empty {
		appendTypeData(endpointIR.Request.StreamingBody, true, true, true, &sd.ServerBodyAttributeTypes)
		appendTypeData(endpointIR.Request.StreamingBody, false, false, false, &sd.ClientBodyAttributeTypes)
	}

	if endpointIR.Response.Result != nil {
		md := sd.Service.Method(endpointIR.MethodName)
		for _, response := range endpointIR.Response.Responses {
			body := effectiveClientResponseBody(response.Body, endpointIR.Response.Result, md)
			collectUserTypes(body.Type, func(ut expr.UserType) {
				if d := sds.attributeTypeData(ut, false, true, false, true, sd); d != nil {
					ensureUnionBranchValidator(d, ut)
					sd.ClientBodyAttributeTypes = append(sd.ClientBodyAttributeTypes, d)
				}
			})
		}
	}
	for _, httpError := range endpointIR.Response.ErrorResponses {
		collectUserTypes(httpError.Body.Type, func(ut expr.UserType) {
			if d := sds.attributeTypeData(ut, false, true, false, true, sd); d != nil {
				ensureUnionBranchValidator(d, ut)
				sd.ClientBodyAttributeTypes = append(sd.ClientBodyAttributeTypes, d)
			}
		})
	}
}

func recordServiceTypeLayouts(endpoints []*transportir.Endpoint, sd *ServiceData) {
	if sd == nil {
		return
	}
	ensureTypeLayoutMaps(sd)
	for _, endpoint := range endpoints {
		recordEndpointRootTypeLayouts(endpoint, sd)
	}
	for _, endpoint := range endpoints {
		recordServerResponseNestedTypeLayouts(endpoint, sd)
	}
	for _, endpoint := range endpoints {
		recordServerRequestNestedTypeLayouts(endpoint, sd)
	}
	for _, endpoint := range endpoints {
		recordClientNestedTypeLayouts(endpoint, sd)
	}
}

func recordEndpointRootTypeLayouts(endpoint *transportir.Endpoint, sd *ServiceData) {
	if endpoint == nil {
		return
	}
	requestJSONPresence := !endpoint.Request.FormEncoded && !endpoint.Request.Multipart
	recordRootTypeLayout(sd, endpoint.Request.RawBody, true, requestJSONPresence, true, false)
	recordRootTypeLayout(sd, endpoint.Request.RawBody, false, false, false, true)
	if endpoint.Stream.RequestPayload != nil && endpoint.Stream.RequestPayload.Type != expr.Empty {
		recordRootTypeLayout(sd, endpoint.Request.StreamingBody, true, true, true, false)
		recordRootTypeLayout(sd, endpoint.Request.StreamingBody, false, false, false, true)
	}
	for _, response := range endpoint.Response.Responses {
		recordRootTypeLayout(sd, response.Body, true, false, false, true)
		recordRootTypeLayout(sd, response.Body, false, true, true, false)
	}
	for _, response := range endpoint.Response.ErrorResponses {
		recordRootTypeLayout(sd, response.Body, true, false, false, true)
		recordRootTypeLayout(sd, response.Body, false, true, true, false)
	}
}

func recordServerResponseNestedTypeLayouts(endpoint *transportir.Endpoint, sd *ServiceData) {
	if endpoint == nil {
		return
	}
	for _, response := range endpoint.Response.Responses {
		if containsUntaggedUnion(response.Body) {
			recordServerRequestValidationTypes(sd, response.Body)
		}
		recordNestedTypeLayouts(sd, response.Body, true, false, false, true)
	}
	for _, response := range endpoint.Response.ErrorResponses {
		if containsUntaggedUnion(response.Body) {
			recordServerRequestValidationTypes(sd, response.Body)
		}
		recordNestedTypeLayouts(sd, response.Body, true, false, false, true)
	}
}

func recordServerRequestNestedTypeLayouts(endpoint *transportir.Endpoint, sd *ServiceData) {
	if endpoint == nil {
		return
	}
	requestJSONPresence := !endpoint.Request.FormEncoded && !endpoint.Request.Multipart
	recordServerRequestValidationTypes(sd, endpoint.Request.RawBody)
	recordNestedTypeLayouts(sd, endpoint.Request.RawBody, true, requestJSONPresence, true, false)
	if endpoint.Stream.RequestPayload != nil && endpoint.Stream.RequestPayload.Type != expr.Empty {
		recordServerRequestValidationTypes(sd, endpoint.Request.StreamingBody)
		recordNestedTypeLayouts(sd, endpoint.Request.StreamingBody, true, true, true, false)
	}
}

func recordServerRequestValidationTypes(sd *ServiceData, attribute *expr.AttributeExpr) {
	if attribute == nil {
		return
	}
	if sd.ServerRequestValidationTypes == nil {
		sd.ServerRequestValidationTypes = make(map[string]bool)
	}
	collectUserTypes(attribute.Type, func(userType expr.UserType) {
		sd.ServerRequestValidationTypes[userType.ID()] = true
		sd.ServerRequestValidationTypes[userType.Name()] = true
	})
}

func recordClientNestedTypeLayouts(endpoint *transportir.Endpoint, sd *ServiceData) {
	if endpoint == nil {
		return
	}
	recordNestedTypeLayouts(sd, endpoint.Request.RawBody, false, false, false, true)
	if endpoint.Stream.RequestPayload != nil && endpoint.Stream.RequestPayload.Type != expr.Empty {
		recordNestedTypeLayouts(sd, endpoint.Request.StreamingBody, false, false, false, true)
	}
	for _, response := range endpoint.Response.Responses {
		recordNestedTypeLayouts(sd, response.Body, false, true, true, false)
	}
	for _, response := range endpoint.Response.ErrorResponses {
		recordNestedTypeLayouts(sd, response.Body, false, true, true, false)
	}
}

func recordRootTypeLayout(sd *ServiceData, attribute *expr.AttributeExpr, server, jsonPresence, pointer, useDefault bool) {
	if attribute == nil {
		return
	}
	if userType, ok := attribute.Type.(expr.UserType); ok {
		recordUserTypeLayout(sd, userType, server, jsonPresence, pointer, useDefault)
	}
}

func recordAttributeTypeLayouts(sd *ServiceData, attribute *expr.AttributeExpr, server, jsonPresence, pointer, useDefault bool) {
	if attribute == nil {
		return
	}
	collectUserTypes(attribute.Type, func(userType expr.UserType) {
		recordUserTypeLayout(sd, userType, server, jsonPresence, pointer, useDefault)
	})
}

func recordNestedTypeLayouts(sd *ServiceData, attribute *expr.AttributeExpr, server, jsonPresence, pointer, useDefault bool) {
	if attribute == nil {
		return
	}
	rootID := ""
	if root, ok := attribute.Type.(expr.UserType); ok {
		rootID = root.ID()
	}
	collectUserTypes(attribute.Type, func(userType expr.UserType) {
		if userType.ID() == rootID {
			return
		}
		recordUserTypeLayout(sd, userType, server, jsonPresence, pointer, useDefault)
	})
}

func recordUserTypeLayout(sd *ServiceData, userType expr.UserType, server, jsonPresence, pointer, useDefault bool) {
	ensureTypeLayoutMaps(sd)
	jsonPresenceTypes, pointerTypes, useDefaultTypes := typeLayoutMaps(sd, server)
	name := userType.Name()
	if recordedJSONPresence, recorded := jsonPresenceTypes[userType.ID()]; recorded {
		recordedPointer := pointerTypes[userType.ID()]
		if typeLayoutPriority(recordedJSONPresence, recordedPointer) >= typeLayoutPriority(jsonPresence, pointer) {
			jsonPresenceTypes[name] = recordedJSONPresence
			pointerTypes[name] = recordedPointer
			useDefaultTypes[name] = useDefaultTypes[userType.ID()]
			return
		}
	} else if recordedJSONPresence, recorded = jsonPresenceTypes[name]; recorded {
		recordedPointer := pointerTypes[name]
		if typeLayoutPriority(recordedJSONPresence, recordedPointer) >= typeLayoutPriority(jsonPresence, pointer) {
			jsonPresenceTypes[userType.ID()] = recordedJSONPresence
			pointerTypes[userType.ID()] = recordedPointer
			useDefaultTypes[userType.ID()] = useDefaultTypes[name]
			return
		}
	}
	jsonPresenceTypes[name] = jsonPresence
	pointerTypes[name] = pointer
	useDefaultTypes[name] = useDefault
	jsonPresenceTypes[userType.ID()] = jsonPresence
	pointerTypes[userType.ID()] = pointer
	useDefaultTypes[userType.ID()] = useDefault
}

func typeLayoutPriority(jsonPresence, pointer bool) uint8 {
	if jsonPresence {
		return 2
	}
	if pointer {
		return 1
	}
	return 0
}

func typeLayoutMaps(sd *ServiceData, server bool) (map[string]bool, map[string]bool, map[string]bool) {
	if server {
		return sd.ServerJSONPresenceTypes, sd.ServerPresencePointerTypes, sd.ServerPresenceUseDefaultTypes
	}
	return sd.ClientJSONPresenceTypes, sd.ClientPresencePointerTypes, sd.ClientPresenceUseDefaultTypes
}

func applyUserTypeLayout(ctx *codegen.AttributeContext, sd *ServiceData, attribute *expr.AttributeExpr, server bool) {
	if attribute == nil {
		return
	}
	userType, ok := attribute.Type.(expr.UserType)
	if !ok {
		return
	}
	jsonPresenceTypes, pointerTypes, useDefaultTypes := typeLayoutMaps(sd, server)
	key := userType.ID()
	jsonPresence, recorded := jsonPresenceTypes[key]
	if !recorded {
		key = userType.Name()
		jsonPresence, recorded = jsonPresenceTypes[key]
		if !recorded {
			return
		}
	}
	ctx.JSONPresence = jsonPresence
	ctx.Pointer = pointerTypes[key]
	ctx.UseDefault = useDefaultTypes[key]
}

func ensureTypeLayoutMaps(sd *ServiceData) {
	if sd.ServerJSONPresenceTypes == nil {
		sd.ServerJSONPresenceTypes = make(map[string]bool)
	}
	if sd.ServerPresencePointerTypes == nil {
		sd.ServerPresencePointerTypes = make(map[string]bool)
	}
	if sd.ServerPresenceUseDefaultTypes == nil {
		sd.ServerPresenceUseDefaultTypes = make(map[string]bool)
	}
	if sd.ClientJSONPresenceTypes == nil {
		sd.ClientJSONPresenceTypes = make(map[string]bool)
	}
	if sd.ClientPresencePointerTypes == nil {
		sd.ClientPresencePointerTypes = make(map[string]bool)
	}
	if sd.ClientPresenceUseDefaultTypes == nil {
		sd.ClientPresenceUseDefaultTypes = make(map[string]bool)
	}
}

func (sds *ServicesData) attributeTypeData(ut expr.UserType, req, ptr, server, jsonPresence bool, rd *ServiceData) *TypeData {
	if ut == expr.Empty {
		return nil
	}
	if expr.IsUnion(ut.Attribute().Type) {
		return nil
	}
	seen := rd.ServerTypeNames
	if !server {
		seen = rd.ClientTypeNames
	}
	if _, ok := seen[ut.Name()]; ok {
		return nil
	}
	seen[ut.Name()] = false
	var (
		name        string
		desc        string
		validate    string
		validateRef string

		att  = &expr.AttributeExpr{Type: ut}
		hctx = httpContext(rd.Scope, req, server)
	)
	recordUserTypeLayout(rd, ut, server, jsonPresence, ptr, hctx.UseDefault)
	applyUserTypeLayout(hctx, rd, att, server)
	if server {
		hctx.JSONPresenceTypes = rd.ServerJSONPresenceTypes
		hctx.PresencePointerTypes = rd.ServerPresencePointerTypes
		hctx.PresenceUseDefaultTypes = rd.ServerPresenceUseDefaultTypes
	} else {
		hctx.JSONPresenceTypes = rd.ClientJSONPresenceTypes
		hctx.PresencePointerTypes = rd.ClientPresencePointerTypes
		hctx.PresenceUseDefaultTypes = rd.ClientPresenceUseDefaultTypes
	}
	name = rd.Scope.GoValueTypeName(att)
	ctx := "request"
	if !req {
		ctx = "response"
	}
	desc = name + " is used to define fields on " + ctx + " body types."
	validate = attributeValidationDefinition(ut, req, server, rd, hctx)
	if validate != "" {
		validateRef = fmt.Sprintf("err = Validate%s(v)", name)
	}
	var example any
	if sds != nil && sds.Root != nil && sds.Root.API != nil {
		example = att.Example(sds.Root.API.ExampleGenerator)
	}
	return &TypeData{
		Name:        ut.Name(),
		VarName:     name,
		Description: desc,
		Def:         goValueTypeDef(rd.Scope, ut.Attribute(), hctx.Pointer, hctx.UseDefault, hctx.JSONPresence),
		Ref:         rd.Scope.GoTypeRef(att),
		ValidateDef: validate,
		ValidateRef: validateRef,
		Example:     example,
	}
}

func attributeValidationDefinition(ut expr.UserType, req, server bool, rd *ServiceData, hctx *codegen.AttributeContext) string {
	if !shouldGenerateAttributeValidation(ut, req, server, rd) {
		return ""
	}
	// Generate validations for responses client-side and for requests
	// server-side and CLI. Alias types are validated inline in the parent type.
	validate := codegen.ValidationCode(ut.Attribute(), ut, hctx, true, expr.IsAlias(ut), false, "body")
	serverUnionBranch := server && (rd.ServerRequestValidationTypes[ut.ID()] || rd.ServerRequestValidationTypes[ut.Name()])
	clientRequestStub := req && !server && needsClientRequestBodyValidatorStub(ut)
	if validate == "" && (serverUnionBranch || clientRequestStub) {
		return "// no validations"
	}
	return validate
}

func shouldGenerateAttributeValidation(ut expr.UserType, request, server bool, data *ServiceData) bool {
	if expr.IsAlias(ut) {
		return false
	}
	if request || !server {
		return true
	}
	return data.ServerRequestValidationTypes[ut.ID()] || data.ServerRequestValidationTypes[ut.Name()]
}

func needsClientRequestBodyValidatorStub(ut expr.UserType) bool {
	if ut == nil || ut.Attribute() == nil || ut.Attribute().Meta == nil {
		return false
	}
	_, ok := ut.Attribute().Meta.Last("oneof:type:tag")
	return ok
}
