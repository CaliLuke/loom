package service

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

type (
	// httpMetadataCarrier retains the native value and identity of one actual
	// HTTP response occurrence. Shared types do not merge occurrence values.
	httpMetadataCarrier struct {
		response  *expr.HTTPResponseExpr
		attribute *expr.AttributeExpr
		role      string
		name      string
	}

	// httpMetadataDeclaration groups one raw object only after its occurrences
	// agree on native storage. It is a plan, not an allocated named type.
	httpMetadataDeclaration struct {
		method     *expr.MethodExpr
		carrier    httpMetadataCarrier
		object     *expr.Object
		fields     *expr.ObjectFields
		location   string
		attributes []*expr.AttributeExpr
	}
)

func httpMetadataCarriers(root *expr.RootExpr, method *expr.MethodExpr) []httpMetadataCarrier {
	if root == nil || root.API == nil || root.API.HTTP == nil || method == nil {
		return nil
	}
	var carriers []httpMetadataCarrier
	for _, service := range root.API.HTTP.Services {
		for _, endpoint := range service.HTTPEndpoints {
			if endpoint.MethodExpr != method || endpoint.IsJSONRPC() {
				continue
			}
			for _, response := range endpoint.Responses {
				if response.Handshake != nil {
					carriers = append(carriers, httpMetadataCarrier{response: response, attribute: response.Handshake, role: "handshake"})
				}
			}
			for _, informational := range endpoint.Informational {
				carriers = append(carriers, httpMetadataCarrier{
					response:  informational.HTTPResponseExpr,
					attribute: informational.Value,
					role:      "informational",
					name:      informational.Name,
				})
			}
		}
	}
	return carriers
}

// normalizeHTTPMetadataCarriers publishes one storage declaration per anonymous
// object identity within a method. Semantic capture precedes this operation.
// Root predicates and annotations remain on their original occurrences.
func normalizeHTTPMetadataCarriers(root *expr.RootExpr, service *expr.ServiceExpr, scope *codegen.NameScope) {
	declarations := planHTTPMetadataDeclarations(root, service)
	for _, plan := range declarations {
		var required []string
		for name, facts := range plan.fields.Fields() {
			if facts.LayoutRequired {
				required = append(required, name)
			}
		}
		definition := &expr.AttributeExpr{
			Type:                 expr.Dup(plan.object),
			AdditionalAttributes: plan.carrier.attribute.AdditionalAttributes,
			Validation:           &expr.ValidationExpr{Required: required},
		}
		if plan.location != "" {
			definition.Meta = expr.MetaExpr{"struct:pkg:path": {plan.location}}
		}
		name := codegen.Goify(plan.method.Name, true) + codegen.Goify(plan.carrier.role, true) + codegen.Goify(plan.carrier.name, true)
		declaration := &expr.UserTypeExpr{
			AttributeExpr: definition,
			TypeName:      scope.PeekUnique(name),
			UID:           service.Name + "#" + plan.method.Name + "#http-" + plan.carrier.role + "#" + plan.carrier.name,
		}
		for _, attribute := range plan.attributes {
			attribute.Type = declaration
		}
	}
}

// planHTTPMetadataDeclarations checks every group before changing any Type or
// allocating a named declaration. Conflicting synthetic roots cannot select the
// first occurrence's layout; public raw Response arguments have bare roots.
func planHTTPMetadataDeclarations(root *expr.RootExpr, service *expr.ServiceExpr) []*httpMetadataDeclaration {
	var plans []*httpMetadataDeclaration
	for _, method := range service.Methods {
		groups := make(map[*expr.Object]*httpMetadataDeclaration)
		for _, carrier := range httpMetadataCarriers(root, method) {
			if carrier.attribute == nil {
				continue
			}
			object, raw := carrier.attribute.Type.(*expr.Object)
			if !raw {
				continue
			}
			fields := expr.NewObjectFields(carrier.attribute)
			location := httpMetadataDestination(carrier.attribute)
			plan := groups[object]
			if plan == nil {
				plan = &httpMetadataDeclaration{method: method, carrier: carrier, object: object, fields: fields, location: location}
				groups[object] = plan
				plans = append(plans, plan)
			} else if reason := incompatibleHTTPMetadataStorage(plan, carrier.attribute, fields, location); reason != "" {
				context := codegen.NewSilentContext().WithService(service).WithMethod(method)
				panic(codegen.NewError(context, carrier.response, fmt.Errorf("incompatible anonymous HTTP metadata %s", reason)))
			}
			plan.attributes = append(plan.attributes, carrier.attribute)
		}
	}
	return plans
}

func incompatibleHTTPMetadataStorage(plan *httpMetadataDeclaration, attribute *expr.AttributeExpr, fields *expr.ObjectFields, location string) string {
	if plan.carrier.attribute.AdditionalAttributes != attribute.AdditionalAttributes {
		return "additional-members carrier: one raw object must share its native additional-members declaration"
	}
	if plan.location != location {
		return "destination: one raw object cannot select multiple native packages"
	}
	for name, facts := range plan.fields.Fields() {
		if facts.LayoutRequired != fields.FieldFacts(name).LayoutRequired {
			return fmt.Sprintf("field layout for %q: one raw object cannot have different native requiredness", name)
		}
	}
	return ""
}

func httpMetadataDestination(attribute *expr.AttributeExpr) string {
	// This temporary view queries the existing location owner; it is never
	// named, allocated in a NameScope, or published as a declaration.
	view := expr.UserTypeExpr{AttributeExpr: attribute}
	if location := codegen.UserTypeLocation(&view); location != nil {
		return location.RelImportPath
	}
	return ""
}
