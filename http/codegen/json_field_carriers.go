package codegen

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

type formCarrier struct {
	body     *expr.AttributeExpr
	contract *bodyCarrierPlan
	label    string
}

// nameFormFieldCarriers gives each field codec an independent declaration.
// Native text roots retain native storage even inside a JSON media union;
// JSON subtrees retain JSON field and element presence when decoding.
func nameFormFieldCarriers(endpoints []*transportir.Endpoint) {
	scope, _ := bodyTypeNameScope(endpointBodyTypeNames(endpoints))
	for _, endpoint := range endpoints {
		for _, carrier := range formCarriers(endpoint) {
			nameFormBodyRoots(carrier.body, carrier.contract, endpoint.Name+" "+carrier.label, scope)
			promote := func(attribute *expr.AttributeExpr, name string, json bool) {
				constraints := selectedTransportConstraints(attribute)
				value := makeHTTPType(attribute)
				// Flattened declarations retain occurrence-first alias metadata.
				value.Meta = expr.HTTPMediaMetadata(value)
				prefix := endpoint.Name + " " + carrier.label + " " + name
				kind := "Form"
				if json {
					kind = "JSON"
					// A recursive JSON field can reach its enclosing native
					// form declaration. Its detached copy needs a JSON identity.
					walkBodyTypes(value, func(typ expr.DataType) {
						if named, ok := typ.(expr.UserType); ok {
							if _, native := named.Attribute().Meta["http:media:form-root"]; native {
								delete(named.Attribute().Meta, "http:media:form-root")
								named.Rename(scope.Unique(codegen.Goify(prefix+" "+named.Name()+" JSON", true)))
							}
						}
					})
				}
				if named, ok := value.Type.(expr.UserType); ok {
					value.Type = named.Attribute().Type
				}
				value.Validation = constraints.Validation().Lowered()
				if selected, ok := constraints.Default(); ok {
					value.DefaultValue = selected
				}
				if value.Meta == nil {
					value.Meta = make(expr.MetaExpr)
				}
				delete(value.Meta, "http:media:form-root")
				if !json {
					value.Meta["http:media:form-root"] = []string{"true"}
				}
				attribute.Type = &expr.UserTypeExpr{TypeName: scope.Unique(codegen.Goify(prefix+" "+kind, true)), AttributeExpr: value}
			}
			walkFormFieldCarriers(carrier.body, carrier.contract, promote)
		}
	}
}

func recordFormFieldCarriers(body *expr.AttributeExpr, contract *bodyCarrierPlan, layout representation.Layout) {
	walkFormFieldCarriers(body, contract, func(attribute *expr.AttributeExpr, _ string, json bool) {
		if json {
			layout.RecordAll(attribute, true, true, false)
		}
	})
}

// formCarriers lists the admitted form contracts in both directions. JSON leaves
// retain decoder presence for outgoing exact-one replay as well as decoding;
// enclosing form objects and text leaves keep their native layouts.
func formCarriers(endpoint *transportir.Endpoint) []formCarrier {
	var carriers []formCarrier
	request := endpoint.Request
	if !request.SkipBodyEncode {
		if request.FormEncoded && request.Form != nil {
			carriers = append(carriers, formCarrier{request.Body, formBodyCarrierPlan(request.Form), ""})
		}
		if plan := multipartBodyCarrierPlan(request.MultipartEncoding); plan != nil {
			carriers = append(carriers, formCarrier{request.Body, plan, ""})
		}
		for _, variant := range request.MediaVariants {
			if plan := multipartBodyCarrierPlan(variant.Multipart); plan != nil {
				carriers = append(carriers, formCarrier{variant.Body, plan, variant.Branch})
			}
			if variant.Form != nil {
				carriers = append(carriers, formCarrier{variant.Body, formBodyCarrierPlan(variant.Form), variant.Branch})
			}
		}
	}
	if endpoint.Response.SkipBodyEncode {
		return carriers
	}
	for _, statuses := range [][]*transportir.ResponseStatus{endpoint.Response.Responses, endpoint.Response.ErrorResponses} {
		for _, status := range statuses {
			for _, variant := range status.MediaVariants {
				if plan := multipartBodyCarrierPlan(variant.Multipart); plan != nil {
					carriers = append(carriers, formCarrier{variant.Body, plan, variant.Branch + " response"})
				}
				if variant.Form != nil {
					carriers = append(carriers, formCarrier{variant.Body, formBodyCarrierPlan(variant.Form), variant.Branch + " response"})
				}
			}
		}
	}
	return carriers
}

func walkFormFieldCarriers(body *expr.AttributeExpr, contract *bodyCarrierPlan, visit func(*expr.AttributeExpr, string, bool)) {
	walkFormBodies(body, contract, func(body *expr.AttributeExpr, contract *bodyCarrierPlan) {
		if object := expr.AsObject(body.Type); object != nil {
			for _, field := range *object {
				captured, visible := contract.fields[field.Name]
				if visible {
					walkFieldCarriers(field.Attribute, captured, field.Name, visit)
				}
			}
		} else if mapping := expr.AsMap(body.Type); mapping != nil && contract.element != nil {
			walkFieldCarriers(mapping.ElemType, *contract.element, "entry", visit)
		}
	})
}

// nameFormBodyRoots gives ancestors of projected fields their own identities.
// Two occurrences cannot share a body declaration after their fields acquire
// different codec-owned types, even when the semantic source ID is the same.
func nameFormBodyRoots(body *expr.AttributeExpr, contract *bodyCarrierPlan, prefix string, scope *codegen.NameScope) {
	if body == nil || contract == nil {
		return
	}
	seen := make(map[expr.UserType]bool)
	for current := body; ; {
		named, ok := current.Type.(expr.UserType)
		if !ok || seen[named] {
			break
		}
		seen[named] = true
		named.Rename(scope.Unique(codegen.Goify(prefix+" "+named.Name()+" Form", true)))
		current = named.Attribute()
		if current.Meta == nil {
			current.Meta = make(expr.MetaExpr)
		}
		current.Meta["http:media:form-root"] = []string{"true"}
	}
	if union := expr.AsUnion(body.Type); union != nil && len(contract.branches) > 0 {
		for index, branch := range union.Values {
			nameFormBodyRoots(branch.Attribute, contract.branches[index], prefix+" "+branch.Name, scope)
		}
	}
}

// walkFieldCarriers visits the complete captured tree, including native leaves.
// Naming replaces each boundary before descending into its detached graph.
func walkFieldCarriers(attribute *expr.AttributeExpr, role fieldCarrierRole, name string, visit func(*expr.AttributeExpr, string, bool)) {
	if custom, _ := codegen.GetMetaType(attribute); custom != "" {
		return
	}
	json := role.json
	if json || !expr.IsPrimitive(attribute.Type) {
		visit(attribute, name, json)
	}
	if json || len(role.branches) == 0 {
		return
	}
	union := expr.AsUnion(attribute.Type)
	for index, branch := range union.Values {
		walkFieldCarriers(branch.Attribute, role.branches[index], name+" "+branch.Name, visit)
	}
}

func walkFormBodies(body *expr.AttributeExpr, contract *bodyCarrierPlan, visit func(*expr.AttributeExpr, *bodyCarrierPlan)) {
	if body == nil || contract == nil {
		return
	}
	if union := expr.AsUnion(body.Type); union != nil && len(contract.branches) > 0 {
		for index, branch := range union.Values {
			walkFormBodies(branch.Attribute, contract.branches[index], visit)
		}
		return
	}
	visit(body, contract)
}

func (b *payloadBuilder) jsonQueryCarrier(parameter *ParamData, attribute *expr.AttributeExpr) {
	if !parameter.JSONField {
		return
	}
	context := httpContext(b.sd.Scope, true, true)
	context.JSONPresence, context.CollectionElementPresence = true, true
	parameter.JSONDecodeType = goTypeDef(b.sd.Scope, attribute, &codegen.AttributeContext{Pointer: true, UseDefault: false, JSONPresence: true, CollectionElementPresence: true})
	if custom, _ := codegen.GetMetaType(attribute); custom != "" {
		// Opaque custom types own their JSON decoding and null policy.
		parameter.JSONDecodeType = parameter.TypeRef
		parameter.JSONDecodeCode = parameter.VarName + " = decoded"
		return
	}
	plan, err := codegen.GoTransform(attribute, attribute, "decoded", parameter.VarName, context, serviceContext("", b.sd.Scope), "unmarshal", false)
	if err != nil {
		panic(codegen.NewError(b.sds.Ctx, attribute, fmt.Errorf("JSON query carrier: %w", err)))
	}
	parameter.JSONDecodeCode = inlineConversionCode(plan)
	b.sd.ServerTransformHelpers = codegen.AppendHelpers(b.sd.ServerTransformHelpers, plan.Helpers)
}
