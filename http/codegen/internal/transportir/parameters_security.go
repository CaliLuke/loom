package transportir

import "github.com/CaliLuke/loom/expr"

func buildPathParameters(endpoint *expr.HTTPEndpointExpr) []*Parameter {
	return buildMappedParameters(endpoint, endpoint.PathParams(), "path")
}

func buildQueryParameters(endpoint *expr.HTTPEndpointExpr) []*Parameter {
	params := buildMappedParameters(endpoint, endpoint.QueryParams(), "query")
	if endpoint == nil || endpoint.MapQueryParams == nil {
		return params
	}
	name := "query"
	attr := endpoint.MethodExpr.Payload
	required := true
	if mappedName := *endpoint.MapQueryParams; mappedName != "" {
		name = mappedName
		attr = expr.AsObject(endpoint.MethodExpr.Payload.Type).Attribute(mappedName)
		required = endpoint.MethodExpr.Payload.IsRequired(mappedName)
	}
	params = append(params, &Parameter{
		Name:           name,
		HTTPName:       name,
		In:             "query",
		Attribute:      attr,
		Required:       required,
		Map:            expr.AsMap(attr.Type) != nil,
		MapQueryParams: endpoint.MapQueryParams,
	})
	return params
}

func buildHeaderParameters(endpoint *expr.HTTPEndpointExpr) []*Parameter {
	return buildMappedParameters(endpoint, endpoint.Headers, "header")
}

func buildCookieParameters(endpoint *expr.HTTPEndpointExpr) []*Parameter {
	return buildMappedParameters(endpoint, endpoint.Cookies, "cookie")
}

func buildMappedParameters(endpoint *expr.HTTPEndpointExpr, mapped *expr.MappedAttributeExpr, in string) []*Parameter {
	if endpoint == nil || mapped == nil {
		return nil
	}
	var params []*Parameter
	expr.WalkMappedAttr(mapped, func(name, element string, attr *expr.AttributeExpr) error { // nolint: errcheck
		params = append(params, &Parameter{
			Name:             name,
			HTTPName:         element,
			In:               in,
			Attribute:        attr,
			Required:         mapped.IsRequired(name),
			PrimitivePointer: mapped.IsPrimitivePointer(name, true),
			Map:              expr.AsMap(attr.Type) != nil,
			StringSlice:      isStringSlice(attr),
			Slice:            expr.AsArray(attr.Type) != nil,
		})
		return nil
	})
	return params
}

func buildSecurity(endpoint *expr.HTTPEndpointExpr) *Security {
	if endpoint == nil {
		return nil
	}
	security := &Security{
		Requirements: endpoint.Requirements,
		Disabled:     endpoint.MethodExpr != nil && hasMetaKey(endpoint.MethodExpr.Meta, "security:no"),
	}
	for _, requirement := range endpoint.Requirements {
		for _, scheme := range requirement.Schemes {
			security.Parameters = append(security.Parameters, &SecurityParameter{
				Name:       scheme.Name,
				In:         scheme.In,
				SchemeName: scheme.SchemeName,
			})
		}
	}
	return security
}

func hasMetaKey(meta expr.MetaExpr, key string) bool {
	_, ok := meta[key]
	return ok
}

func isStringSlice(attr *expr.AttributeExpr) bool {
	array := expr.AsArray(attr.Type)
	return array != nil && array.ElemType.Type.Kind() == expr.StringKind
}
