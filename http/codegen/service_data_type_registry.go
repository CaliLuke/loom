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

	appendTypeData := func(att *expr.AttributeExpr, ptr, server bool, target *[]*TypeData) {
		collectUserTypes(att.Type, func(ut expr.UserType) {
			if d := sds.attributeTypeData(ut, true, ptr, server, sd); d != nil {
				if !server && d.ValidateDef == "" {
					if _, ok := unionBranchTypes[ut.ID()]; ok {
						d.ValidateDef = "// no validations"
						d.ValidateRef = fmt.Sprintf("err = Validate%s(v)", d.VarName)
					}
				}
				*target = append(*target, d)
			}
		})
	}
	appendTypeData(endpointIR.Request.RawBody, true, true, &sd.ServerBodyAttributeTypes)
	appendTypeData(endpointIR.Request.RawBody, false, false, &sd.ClientBodyAttributeTypes)

	if endpointIR.Stream.RequestPayload != nil && endpointIR.Stream.RequestPayload.Type != expr.Empty {
		appendTypeData(endpointIR.Request.StreamingBody, true, true, &sd.ServerBodyAttributeTypes)
		appendTypeData(endpointIR.Request.StreamingBody, false, false, &sd.ClientBodyAttributeTypes)
	}

	if endpointIR.Response.Result != nil {
		md := sd.Service.Method(endpointIR.MethodName)
		for _, response := range endpointIR.Response.Responses {
			body := effectiveClientResponseBody(response.Body, endpointIR.Response.Result, md)
			collectUserTypes(body.Type, func(ut expr.UserType) {
				if d := sds.attributeTypeData(ut, false, true, false, sd); d != nil {
					sd.ClientBodyAttributeTypes = append(sd.ClientBodyAttributeTypes, d)
				}
			})
		}
	}
	for _, httpError := range endpointIR.Response.ErrorResponses {
		collectUserTypes(httpError.Body.Type, func(ut expr.UserType) {
			if d := sds.attributeTypeData(ut, false, true, false, sd); d != nil {
				sd.ClientBodyAttributeTypes = append(sd.ClientBodyAttributeTypes, d)
			}
		})
	}
}

func (sds *ServicesData) attributeTypeData(ut expr.UserType, req, ptr, server bool, rd *ServiceData) *TypeData {
	if ut == expr.Empty {
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
	name = rd.Scope.GoTypeName(att)
	ctx := "request"
	if !req {
		ctx = "response"
	}
	desc = name + " is used to define fields on " + ctx + " body types."
	if (req || !req && !server) && !expr.IsAlias(ut) {
		// Generate validations for responses client-side and for
		// requests server-side and CLI.
		// Alias types are validated inline in the parent type
		validate = codegen.ValidationCode(ut.Attribute(), ut, hctx, true, expr.IsAlias(ut), false, "body")
		if validate == "" && req && !server && needsClientRequestBodyValidatorStub(ut) {
			validate = "// no validations"
		}
	}
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
		Def:         goTypeDef(rd.Scope, ut.Attribute(), ptr, hctx.UseDefault),
		Ref:         rd.Scope.GoTypeRef(att),
		ValidateDef: validate,
		ValidateRef: validateRef,
		Example:     example,
	}
}

func needsClientRequestBodyValidatorStub(ut expr.UserType) bool {
	if ut == nil || ut.Attribute() == nil || ut.Attribute().Meta == nil {
		return false
	}
	_, ok := ut.Attribute().Meta.Last("oneof:type:tag")
	return ok
}
