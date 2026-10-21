package representation

import "github.com/CaliLuke/loom/expr"

// DocumentationPlanRequest captures OpenAPI visibility in the value plan before
// projection. It preserves supplied emitter policies and hides only fields
// explicitly excluded from the documentation contract. Runtime plans are unchanged.
func DocumentationPlanRequest(target *expr.AttributeExpr, request expr.ValuePlanRequest) expr.ValuePlanRequest {
	request.Codec, request.Use = expr.ValueCodecJSON, expr.ValuePlanDocumentation
	request.Fields = append([]expr.ValueFieldPolicy(nil), request.Fields...)
	visited := make(map[*expr.AttributeExpr]bool)
	var walk func(*expr.AttributeExpr)
	walk = func(attribute *expr.AttributeExpr) {
		if attribute == nil || visited[attribute] {
			return
		}
		visited[attribute] = true
		switch actual := attribute.Type.(type) {
		case expr.UserType:
			walk(actual.Attribute())
		case *expr.Object:
			for _, field := range *actual {
				if disabled, _ := field.Attribute.Meta.Last("openapi:generate"); disabled == "false" {
					found := false
					for index := range request.Fields {
						policy := &request.Fields[index]
						if policy.Parent == attribute && policy.Target == field.Attribute && policy.Name == field.Name {
							policy.Visible, policy.Required = false, false
							found = true
						}
					}
					if !found {
						kind := field.Attribute.Type
						for {
							alias, ok := kind.(expr.UserType)
							if !ok {
								break
							}
							kind = alias.Attribute().Type
						}
						request.Fields = append(request.Fields, expr.ValueFieldPolicy{
							Parent: attribute, Target: field.Attribute, Name: field.Name,
							WireName: expr.JSONFieldName(expr.ElementName(field.Name), field.Attribute),
							Presence: expr.ValueFieldRetain, NumericKind: kind.Kind(),
						})
					}
				}
				walk(field.Attribute)
			}
		case *expr.Array:
			walk(actual.ElemType)
		case *expr.Map:
			walk(actual.KeyType)
			walk(actual.ElemType)
		case *expr.Union:
			for _, branch := range actual.Values {
				walk(branch.Attribute)
			}
		}
	}
	walk(target)
	return request
}
