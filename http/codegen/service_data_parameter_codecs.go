package codegen

import (
	"fmt"
	"slices"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

func (b *payloadBuilder) configureParameterElement(element *Element, parameter *transportir.Parameter, peers []*transportir.Parameter) {
	element.Codec = parameter.Codec
	element.Styled = parameterUsesRuntime(parameter.In, parameter.Codec)
	configureStyledCarrier(element, b.svcctx)
	claims := make(map[string]httpencoding.KeySet, len(peers))
	for _, peer := range peers {
		claims[peer.Name] = peer.Keys
	}
	if b.endpointIR.Security != nil {
		for _, credential := range b.endpointIR.Security.CredentialBindings {
			if credential.Location != parameter.In || slices.Contains(credential.Sources, parameter.Name) {
				continue
			}
			claims["credential "+credential.Name] = httpencoding.KeySet{Literals: []string{credential.Name}}
		}
	}
	element.Namespace = parameterNamespace(parameter.Name, parameter.Codec, parameter.Keys, claims)
}

func parameterUsesRuntime(location string, codec httpencoding.Codec) bool {
	associative := codec.Shape == httpencoding.Object || codec.Shape == httpencoding.ScalarMap
	switch location {
	case "path":
		return codec.Style != httpencoding.Simple || associative
	case "query", "form":
		if codec.Style == httpencoding.JSON || codec.Style == httpencoding.Root {
			return false
		}
		return associative || codec.Style != httpencoding.Form || codec.Shape == httpencoding.Sequence && !codec.Explode
	case "header":
		return associative
	case "cookie":
		return codec.Style == httpencoding.Form || codec.Shape != httpencoding.Scalar
	default:
		return false
	}
}

func parameterNamespace(name string, codec httpencoding.Codec, owned httpencoding.KeySet, peers map[string]httpencoding.KeySet) loomhttp.ParamNamespace {
	namespace := loomhttp.ParamNamespace{}
	for peer, keys := range peers {
		if peer == name {
			continue
		}
		namespace.Literals = append(namespace.Literals, keys.Literals...)
		namespace.Prefixes = append(namespace.Prefixes, keys.Prefixes...)
	}
	slices.Sort(namespace.Literals)
	namespace.Literals = slices.Compact(namespace.Literals)
	slices.Sort(namespace.Prefixes)
	namespace.Prefixes = slices.Compact(namespace.Prefixes)
	if codec.Shape == httpencoding.Object && (codec.Style == httpencoding.DeepObject || codec.Explode && (codec.Style == httpencoding.Form || codec.Style == httpencoding.CookieStyle)) {
		namespace.OwnedLiterals = append([]string{}, owned.Literals...)
	}
	return namespace
}

func (b *payloadBuilder) configureFormFields(request *RequestData) {
	source := b.endpointIR.Request
	if !source.FormEncoded || source.Body == nil {
		return
	}
	body := expr.UnwrapInlineHTTPBody(source.Body)
	request.StyledFormFields, request.OrdinaryFormFields = formFieldData(body, source.FormFieldCodecs)
}

func formFieldData(body *expr.AttributeExpr, codecs map[string]httpencoding.Codec) (styled, ordinary []*FormFieldData) {
	object := expr.AsObject(body.Type)
	if object == nil || len(codecs) == 0 {
		return nil, nil
	}
	claims := make(map[string]httpencoding.KeySet, len(*object))
	for _, field := range *object {
		name := expr.ElementName(field.Name)
		_, keys, err := httpencoding.Resolve(httpencoding.FormField, name, expr.HTTPValueShape(field.Attribute), expr.HTTPParameterSerialization(body, "openapi:encoding:"+name+":"))
		if err != nil {
			panic(fmt.Errorf("validated form field %q: %w", name, err))
		}
		claims[field.Name] = keys
	}
	for _, field := range *object {
		name := expr.ElementName(field.Name)
		codec := codecs[name]
		data := &FormFieldData{
			AttributeName: expr.AttributeName(field.Name),
			HTTPName:      name,
			FieldName:     codegen.GoifyAtt(field.Attribute, expr.AttributeName(field.Name), true),
			Attribute:     field.Attribute,
			Codec:         codec,
			Namespace:     parameterNamespace(field.Name, codec, claims[field.Name], claims),
			Required:      body.IsRequired(field.Name),
		}
		if parameterUsesRuntime("form", codec) {
			styled = append(styled, data)
		} else {
			ordinary = append(ordinary, data)
		}
	}
	return styled, ordinary
}

// configureStyledCarrier uses the service declaration for newly admitted flat
// objects. Existing scalar and collection carriers retain their prior layout.
func configureStyledCarrier(element *Element, ctx *codegen.AttributeContext) {
	if !element.Styled || !expr.IsObject(element.Type) {
		return
	}
	att := *element.attribute
	att.Type = element.FieldType
	element.attribute = &att
	element.context = ctx
	element.Type = att.Type
	element.TypeName = ctx.Scope.Name(&att, ctx.Pkg(&att), ctx.Pointer, ctx.UseDefault)
	element.TypeRef = ctx.Scope.Ref(&att, ctx.Pkg(&att))
	if _, inline := att.Type.(*expr.Object); inline {
		// An anonymous object at a service field is a pointer even though a
		// standalone anonymous struct reference is a value.
		element.TypeRef = fmt.Sprintf("%#v", jen.Op("*").Add(codegen.TypeRef(element.TypeName)))
	}
	element.Validate = codegen.AttributeValidationCode(&att, nil, ctx, element.Required, false, element.VarName, element.Name)
}
