package codegen

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func (sds *ServicesData) collectBodyAttributeTypes(sd *ServiceData) {
	for _, root := range sd.orderedBodyRoots() {
		// Server responses are collected while building their projected type.
		if root.server && !root.request {
			continue
		}
		branches := make(map[string]struct{})
		collectUnionBranchUserTypes(root.attribute, branches)
		target := &sd.ClientBodyAttributeTypes
		if root.server {
			target = &sd.ServerBodyAttributeTypes
		}
		representation.WalkUserTypes(root.attribute.Type, func(ut expr.UserType) {
			data := sds.attributeTypeData(ut, root.request, root.request == root.server, root.server, root.jsonPresence, sd)
			if data == nil {
				return
			}
			if _, branch := branches[ut.Hash()]; branch && data.ValidateDef == "" {
				data.ValidateDef = "// no validations"
				data.ValidateRef = fmt.Sprintf("err = Validate%s(v)", data.VarName)
			}
			*target = append(*target, data)
		})
	}
}

func recordServiceTypeLayouts(endpoints []*transportir.Endpoint, sd *ServiceData) {
	if sd == nil {
		return
	}
	server, client := representation.ServiceLayouts(endpoints)
	for _, endpoint := range endpoints {
		recordJSONFormCarriers(endpoint.Request, server)
	}
	sd.ServerJSONPresenceTypes, sd.ServerPresencePointerTypes, sd.ServerPresenceUseDefaultTypes = server.JSONPresence, server.Pointer, server.UseDefault
	sd.ClientJSONPresenceTypes, sd.ClientPresencePointerTypes, sd.ClientPresenceUseDefaultTypes = client.JSONPresence, client.Pointer, client.UseDefault
	for _, endpoint := range endpoints {
		recordServerRequestValidationTypes(sd, endpoint.Request.Body)
		recordServerRequestValidationTypes(sd, endpoint.Request.StreamingBody)
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
		Name:               ut.Name(),
		VarName:            name,
		Description:        desc,
		Def:                goValueTypeDef(rd.Scope, ut.Attribute(), hctx.Pointer, hctx.UseDefault, hctx.JSONPresence),
		Ref:                rd.Scope.GoTypeRef(att),
		ValidateDef:        validate,
		validationFunction: codegen.NamedValidationFunction(att, hctx, "body"),
		ValidateRef:        validateRef,
		Example:            example,
	}
}

func attributeValidationDefinition(ut expr.UserType, req, server bool, rd *ServiceData, hctx *codegen.AttributeContext) string {
	if !shouldGenerateAttributeValidation(ut, req, server, rd) {
		return ""
	}
	// Generate validations for responses client-side and for requests
	// server-side and CLI. Alias types are validated inline in the parent type.
	validate := codegen.ValidationCode(ut.Attribute(), ut, codegen.NamedValidationFunction(&expr.AttributeExpr{Type: ut}, hctx, "body").Context(hctx), true, expr.IsAlias(ut), false, "body")
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
