package codegen

import (
	"net/http"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func (sds *ServicesData) examplesFor(sd *ServiceData) *expr.ExampleGenerator {
	if sd != nil && sd.exampleGenerator != nil {
		return sd.exampleGenerator
	}
	return sds.Root.API.ExampleGenerator
}

func transportStringSlice(att *expr.AttributeExpr) bool {
	arr := expr.AsArray(att.Type)
	return arr != nil && arr.ElemType.Type.Kind() == expr.StringKind
}

// transportFieldBinding resolves the actual physical service occurrence bound
// to a transport value. The returned attribute retains its authored storage
// metadata, which value conversions consume.
func transportFieldBinding(name string, fieldAttr, svcAtt *expr.AttributeExpr, svcCtx *codegen.AttributeContext) (string, *expr.AttributeExpr, bool) {
	if !expr.IsObject(svcAtt.Type) {
		return "", svcAtt, false
	}
	key, svcField := svcAtt.FindAttribute(name)
	if svcField == nil {
		return "", fieldAttr, false
	}
	fieldName := codegen.GoifyAtt(svcField, name, true)
	if expr.AsUnion(svcField.Type) != nil {
		return fieldName, svcField, expr.ServiceFieldLayout(svcAtt, key, svcField, false, true).Pointer
	}
	if svcCtx == nil {
		return fieldName, svcField, svcAtt.IsPrimitivePointer(key, true)
	}
	return fieldName, svcField, svcCtx.IsPrimitivePointer(key, svcAtt)
}

func (sds *ServicesData) buildTransportAttributeData(
	name string,
	attr *expr.AttributeExpr,
	required bool,
	pointer bool,
	fieldName string,
	field *expr.AttributeExpr,
	fieldPointer bool,
	validateCtx *codegen.AttributeContext,
	scope *codegen.NameScope,
	vars *transportVarScope,
	examples *expr.ExampleGenerator,
) *AttributeData {
	var avoid func(string) bool
	if expr.AsMap(attr.Type) != nil {
		avoid = isMapQueryLocal
	}
	varName, locals := vars.allocate(scope.PeekUnique(codegen.Goify(name, false)), avoid)
	typeRef := scope.GoTypeRef(attr)
	if pointer {
		typeRef = "*" + typeRef
	}
	textUnmarshaler := isStringMetaType(attr)
	return &AttributeData{
		Name:              name,
		Description:       attr.Description,
		FieldName:         fieldName,
		FieldPointer:      fieldPointer,
		FieldType:         field.Type,
		fieldAttribute:    field,
		fieldContext:      validateCtx,
		VarName:           varName,
		Required:          required,
		Type:              attr.Type,
		TypeName:          scope.GoTypeName(attr),
		TypeRef:           typeRef,
		Pointer:           pointer,
		Validate:          transportAttributeValidation(attr, validateCtx, required, varName, locals.Raw, name),
		IsTextUnmarshaler: textUnmarshaler,
		DefaultValue:      attr.DefaultValue,
		Example:           attr.Example(examples),
		Locals:            locals,
	}
}

func transportAttributeValidation(attribute *expr.AttributeExpr, context *codegen.AttributeContext, required bool, target, raw, name string) string {
	if isStringMetaType(attribute) {
		attribute = rawTextValidationAttribute(attribute)
		target, required = raw, true
	}
	return codegen.AttributeValidationCode(attribute, nil, context, required, expr.IsAlias(attribute.Type), target, name)
}

func isStringMetaType(attr *expr.AttributeExpr) bool {
	if attr == nil || attr.Type == nil || attr.Type.Kind() != expr.StringKind {
		return false
	}
	typeName, _ := codegen.GetMetaType(attr)
	return typeName != ""
}

// rawTextValidationAttribute describes the native string supplied to a custom
// text parser. Only effective non-format constraints apply to this carrier:
// authored format validation belongs to the parser, and custom Go metadata
// describes the parsed destination rather than the raw transport text.
func rawTextValidationAttribute(attr *expr.AttributeExpr) *expr.AttributeExpr {
	constraints := selectedTransportConstraints(attr)
	validation := constraints.Validation().Lowered()
	validation.Format = ""
	validation.FormatClauses = nil
	defaultValue, _ := constraints.Default()
	return &expr.AttributeExpr{
		Type:         expr.String,
		Validation:   validation,
		DefaultValue: defaultValue,
	}
}

func (sds *ServicesData) buildTransportElement(
	name string,
	elem string,
	attr *expr.AttributeExpr,
	stringSlice bool,
	required bool,
	pointer bool,
	fieldName string,
	field *expr.AttributeExpr,
	fieldPointer bool,
	validateCtx *codegen.AttributeContext,
	scope *codegen.NameScope,
	vars *transportVarScope,
	examples *expr.ExampleGenerator,
) *Element {
	return &Element{
		attribute:     attr,
		context:       validateCtx,
		HTTPName:      elem,
		AttributeName: name,
		StringSlice:   stringSlice,
		Slice:         expr.AsArray(attr.Type) != nil,
		AttributeData: sds.buildTransportAttributeData(name, attr, required, pointer, fieldName, field, fieldPointer, validateCtx, scope, vars, examples),
	}
}

func (sds *ServicesData) extractHeaders(headersIR []*transportir.Header, svcAtt *expr.AttributeExpr, svcCtx *codegen.AttributeContext, scope *codegen.NameScope, vars *transportVarScope, examples *expr.ExampleGenerator) []*HeaderData {
	headers := make([]*HeaderData, 0, len(headersIR))
	for _, headerIR := range headersIR {
		name := headerIR.Name
		elem := headerIR.HTTPName
		_, attr := svcAtt.FindAttribute(name)
		if attr == nil {
			attr = svcAtt
		}
		stringSlice := transportStringSlice(attr)
		hattr := makeHTTPMappedType(headerIR.Attribute, attr)
		pointer := headerIR.PrimitivePointer
		fieldName, field, fieldPointer := transportFieldBinding(name, attr, svcAtt, svcCtx)
		headers = append(headers, &HeaderData{
			CanonicalName: http.CanonicalHeaderKey(elem),
			Element:       sds.buildTransportElement(name, elem, hattr, stringSlice, headerIR.Required, pointer, fieldName, field, fieldPointer, svcCtx, scope, vars, examples),
		})
		header := headers[len(headers)-1]
		header.Codec = headerIR.Codec
		header.Styled = parameterUsesRuntime("header", headerIR.Codec)
		configureStyledCarrier(header.Element, svcCtx)
		native := makeHTTPType(attr)
		if header.nativeContext != nil {
			native = attr
		}
		nativeValidation := transportAttributeValidation(native, svcCtx, header.Required, header.VarName, header.Locals.Raw, header.Name)
		if nativeValidation != "" && nativeValidation != header.Validate {
			header.Validate = strings.TrimSpace(nativeValidation + "\n" + header.Validate)
		}
	}
	return headers
}

func (sds *ServicesData) extractResponseCookies(cookiesIR []*transportir.Cookie, svcAtt *expr.AttributeExpr, svcCtx *codegen.AttributeContext, scope *codegen.NameScope, vars *transportVarScope, examples *expr.ExampleGenerator) []*CookieData {
	cookies := make([]*CookieData, 0, len(cookiesIR))
	for _, cookieIR := range cookiesIR {
		name := cookieIR.Name
		if name == "" {
			continue
		}
		cookie := sds.cookieData(name, cookieIR.HTTPName, cookieIR.Required, cookieIR.PrimitivePointer, cookieIR.Attribute, svcAtt, svcCtx, scope, vars, examples)
		cookie.MaxAge = cookieIR.MaxAge
		cookie.Path = cookieIR.Path
		cookie.Domain = cookieIR.Domain
		cookie.Secure = cookieIR.Secure
		cookie.HTTPOnly = cookieIR.HTTPOnly
		switch cookieIR.SameSite {
		case expr.CookieSameSiteLax:
			cookie.SameSite = "http.SameSiteLaxMode"
		case expr.CookieSameSiteStrict:
			cookie.SameSite = "http.SameSiteStrictMode"
		case expr.CookieSameSiteNone:
			cookie.SameSite = "http.SameSiteNoneMode"
		case expr.CookieSameSiteDefault:
			cookie.SameSite = "http.SameSiteDefaultMode"
		}
		cookies = append(cookies, cookie)
	}
	return cookies
}

func (sds *ServicesData) cookieData(name, elem string, required bool, pointer bool, mappedAttr *expr.AttributeExpr, svcAtt *expr.AttributeExpr, svcCtx *codegen.AttributeContext, scope *codegen.NameScope, vars *transportVarScope, examples *expr.ExampleGenerator) *CookieData {
	_, serviceAttr := svcAtt.FindAttribute(name)
	if serviceAttr == nil {
		if expr.IsObject(svcAtt.Type) && mappedAttr != nil {
			serviceAttr = mappedAttr
		} else {
			serviceAttr = svcAtt
		}
	}
	if mappedAttr == nil {
		mappedAttr = serviceAttr
	}
	stringSlice := transportStringSlice(serviceAttr)
	hattr := makeHTTPMappedType(mappedAttr, serviceAttr)
	fieldName, field, fieldPointer := transportFieldBinding(name, hattr, svcAtt, svcCtx)
	return &CookieData{
		Element: sds.buildTransportElement(name, elem, hattr, stringSlice, required, pointer, fieldName, field, fieldPointer, svcCtx, scope, vars, examples),
	}
}
