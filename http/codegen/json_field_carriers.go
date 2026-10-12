package codegen

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

// nameJSONFormCarriers gives each JSON subtree a declaration boundary. The form
// root retains native field presence; each JSON subtree uses the existing JSON
// object/collection carriers and their shared conversions.
func nameJSONFormCarriers(endpoints []*transportir.Endpoint) {
	scope, _ := bodyTypeNameScope(endpointBodyTypeNames(endpoints))
	for _, endpoint := range endpoints {
		request := endpoint.Request
		if !request.FormEncoded || request.SkipBodyEncode || request.Body == nil {
			continue
		}
		promote := func(attribute *expr.AttributeExpr, name string) {
			if httpencoding.Select(httpencoding.FormField, transportir.URLShape(attribute)) != httpencoding.JSON {
				return
			}
			if custom, _ := codegen.GetMetaType(attribute); custom != "" {
				return // A custom JSON codec owns its physical representation.
			}
			constraints := selectedTransportConstraints(attribute)
			value := makeHTTPType(attribute)
			if named, ok := value.Type.(expr.UserType); ok {
				value.Type = named.Attribute().Type
			}
			value.Validation = constraints.Validation().Lowered()
			if selected, ok := constraints.Default(); ok {
				value.DefaultValue = selected
			}
			attribute.Type = &expr.UserTypeExpr{TypeName: scope.Unique(codegen.Goify(endpoint.Name+" "+name+" JSON", true)), AttributeExpr: value}
		}
		if object := expr.AsObject(request.Body.Type); object != nil {
			for _, field := range *object {
				if request.FormFieldModes[expr.ElementName(field.Name)] == httpencoding.JSON {
					promote(field.Attribute, field.Name)
				}
			}
		} else if mapping := expr.AsMap(request.Body.Type); mapping != nil {
			promote(mapping.ElemType, "entry")
		}
	}
}

func recordJSONFormCarriers(request *transportir.Request, layout representation.Layout) {
	record := func(attribute *expr.AttributeExpr) {
		if custom, _ := codegen.GetMetaType(attribute); custom == "" {
			layout.RecordAll(attribute, true, true, false)
		}
	}
	if !request.FormEncoded || request.SkipBodyEncode || request.Body == nil {
		return
	}
	if object := expr.AsObject(request.Body.Type); object != nil {
		for _, field := range *object {
			if request.FormFieldModes[expr.ElementName(field.Name)] == httpencoding.JSON {
				record(field.Attribute)
			}
		}
	} else if mapping := expr.AsMap(request.Body.Type); mapping != nil && httpencoding.Select(httpencoding.FormField, transportir.URLShape(mapping.ElemType)) == httpencoding.JSON {
		record(mapping.ElemType)
	}
}

func (b *payloadBuilder) jsonQueryCarrier(parameter *ParamData, attribute *expr.AttributeExpr) {
	if !parameter.JSONField {
		return
	}
	context := httpContext(b.sd.Scope, true, true)
	context.JSONPresence, context.CollectionElementPresence = true, true
	parameter.JSONDecodeType = goTypeDef(b.sd.Scope, attribute, true, false, true)
	if custom, _ := codegen.GetMetaType(attribute); custom != "" {
		// Opaque custom types own their JSON decoding and null policy.
		parameter.JSONDecodeType = parameter.TypeRef
		parameter.JSONDecodeCode = parameter.VarName + " = decoded"
		return
	}
	conversion, helpers, err := codegen.GoTransform(attribute, attribute, "decoded", parameter.VarName, context, serviceContext("", b.sd.Scope), "unmarshal", false)
	if err != nil {
		panic(codegen.NewError(b.sds.Ctx, attribute, fmt.Errorf("JSON query carrier: %w", err)))
	}
	parameter.JSONDecodeCode = conversion
	b.sd.ServerTransformHelpers = codegen.AppendHelpers(b.sd.ServerTransformHelpers, helpers)
}
