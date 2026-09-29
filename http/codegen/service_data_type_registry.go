package codegen

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func (sds *ServicesData) collectEndpointBodyAttributeTypes(endpointIR *transportir.Endpoint, sd *ServiceData) {
	unionBranchTypes := make(map[string]struct{})
	collectUnionBranchUserTypes(endpointIR.Request.Body, unionBranchTypes)
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
		if _, ok := unionBranchTypes[userType.Hash()]; ok {
			data.ValidateDef = "// no validations"
			data.ValidateRef = fmt.Sprintf("err = Validate%s(v)", data.VarName)
		}
	}

	appendTypeData := func(att *expr.AttributeExpr, ptr, server, jsonPresence bool, target *[]*TypeData) {
		representation.WalkUserTypes(att.Type, func(ut expr.UserType) {
			if d := sds.attributeTypeData(ut, true, ptr, server, jsonPresence, sd); d != nil {
				ensureUnionBranchValidator(d, ut)
				*target = append(*target, d)
			}
		})
	}
	requestJSONPresence := !endpointIR.Request.FormEncoded && !endpointIR.Request.Multipart
	appendTypeData(endpointIR.Request.Body, true, true, requestJSONPresence, &sd.ServerBodyAttributeTypes)
	appendTypeData(endpointIR.Request.Body, false, false, false, &sd.ClientBodyAttributeTypes)

	if endpointIR.Stream.RequestPayload != nil && endpointIR.Stream.RequestPayload.Type != expr.Empty {
		appendTypeData(endpointIR.Request.StreamingBody, true, true, true, &sd.ServerBodyAttributeTypes)
		appendTypeData(endpointIR.Request.StreamingBody, false, false, false, &sd.ClientBodyAttributeTypes)
	}

	if endpointIR.Response.Result != nil {
		md := sd.Service.Method(endpointIR.MethodName)
		for _, response := range endpointIR.Response.Responses {
			body := effectiveClientResponseBody(response.Body, endpointIR.Response.Result, md)
			representation.WalkUserTypes(body.Type, func(ut expr.UserType) {
				if d := sds.attributeTypeData(ut, false, true, false, true, sd); d != nil {
					ensureUnionBranchValidator(d, ut)
					sd.ClientBodyAttributeTypes = append(sd.ClientBodyAttributeTypes, d)
				}
			})
		}
	}
	for _, httpError := range endpointIR.Response.ErrorResponses {
		representation.WalkUserTypes(httpError.Body.Type, func(ut expr.UserType) {
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
	server, client := representation.ServiceLayouts(endpoints)
	sd.ServerJSONPresenceTypes, sd.ServerPresencePointerTypes, sd.ServerPresenceUseDefaultTypes = server.JSONPresence, server.Pointer, server.UseDefault
	sd.ClientJSONPresenceTypes, sd.ClientPresencePointerTypes, sd.ClientPresenceUseDefaultTypes = client.JSONPresence, client.Pointer, client.UseDefault
	for _, endpoint := range endpoints {
		recordServerRequestValidationTypes(sd, endpoint.Request.Body)
		recordServerRequestValidationTypes(sd, endpoint.Request.StreamingBody)
		for _, response := range append(append([]*transportir.ResponseStatus(nil), endpoint.Response.Responses...), endpoint.Response.ErrorResponses...) {
			if containsUntaggedUnion(response.Body) {
				recordServerRequestValidationTypes(sd, response.Body)
			}
		}
	}
}

func recordServerRequestValidationTypes(sd *ServiceData, attribute *expr.AttributeExpr) {
	if attribute == nil {
		return
	}
	if sd.ServerRequestValidationTypes == nil {
		sd.ServerRequestValidationTypes = make(map[string]bool)
	}
	representation.WalkUserTypes(attribute.Type, func(userType expr.UserType) {
		sd.ServerRequestValidationTypes[userType.Hash()] = true
	})
}

func recordRootTypeLayout(sd *ServiceData, attribute *expr.AttributeExpr, server, jsonPresence, pointer, useDefault bool) {
	ensureTypeLayoutMaps(sd)
	presence, pointers, defaults := typeLayoutMaps(sd, server)
	representation.Layout{JSONPresence: presence, Pointer: pointers, UseDefault: defaults}.RecordRoot(attribute, jsonPresence, pointer, useDefault)
}

func recordAttributeTypeLayouts(sd *ServiceData, attribute *expr.AttributeExpr, server, jsonPresence, pointer, useDefault bool) {
	ensureTypeLayoutMaps(sd)
	presence, pointers, defaults := typeLayoutMaps(sd, server)
	representation.Layout{JSONPresence: presence, Pointer: pointers, UseDefault: defaults}.RecordAll(attribute, jsonPresence, pointer, useDefault)
}

// recordUserTypeLayout records the layout of the Go type of userType unless
// the type already has a layout of equal or higher priority. Layouts are keyed
// by the hash of the user type, which names its Go type: the request and
// response body types of a result type share the identifier of the result
// type but are different Go types with different layouts.
func recordUserTypeLayout(sd *ServiceData, userType expr.UserType, server, jsonPresence, pointer, useDefault bool) {
	ensureTypeLayoutMaps(sd)
	presence, pointers, defaults := typeLayoutMaps(sd, server)
	representation.Layout{JSONPresence: presence, Pointer: pointers, UseDefault: defaults}.Record(userType, jsonPresence, pointer, useDefault)
}

func typeLayoutMaps(sd *ServiceData, server bool) (map[string]bool, map[string]bool, map[string]bool) {
	if server {
		return sd.ServerJSONPresenceTypes, sd.ServerPresencePointerTypes, sd.ServerPresenceUseDefaultTypes
	}
	return sd.ClientJSONPresenceTypes, sd.ClientPresencePointerTypes, sd.ClientPresenceUseDefaultTypes
}

func applyUserTypeLayout(ctx *codegen.AttributeContext, sd *ServiceData, attribute *expr.AttributeExpr, server bool) {
	ctx.JSONPresenceTypes, ctx.PresencePointerTypes, ctx.PresenceUseDefaultTypes = typeLayoutMaps(sd, server)
	representation.ApplyLayout(ctx, attribute)
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
		example = att.Example(sds.examplesFor(rd))
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
	serverUnionBranch := server && rd.ServerRequestValidationTypes[ut.Hash()]
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
	return data.ServerRequestValidationTypes[ut.Hash()]
}

func needsClientRequestBodyValidatorStub(ut expr.UserType) bool {
	if ut == nil || ut.Attribute() == nil || ut.Attribute().Meta == nil {
		return false
	}
	_, ok := ut.Attribute().Meta.Last("oneof:type:tag")
	return ok
}
