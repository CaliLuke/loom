package codegen

import (
	"net/http"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func transportStringSlice(att *expr.AttributeExpr) bool {
	arr := expr.AsArray(att.Type)
	return arr != nil && arr.ElemType.Type.Kind() == expr.StringKind
}

func transportFieldBinding(name string, fieldAttr, svcAtt *expr.AttributeExpr, svcCtx *codegen.AttributeContext) (string, expr.DataType, bool) {
	fieldType := svcAtt.Type
	if !expr.IsObject(svcAtt.Type) {
		return "", fieldType, false
	}
	svcField := svcAtt.Find(name)
	if svcField == nil {
		return "", fieldAttr.Type, false
	}
	fieldType = svcField.Type
	fieldName := codegen.GoifyAtt(fieldAttr, name, true)
	if svcCtx == nil {
		return fieldName, fieldType, svcAtt.IsPrimitivePointer(name, true)
	}
	return fieldName, fieldType, svcCtx.IsPrimitivePointer(name, svcAtt)
}

func (sds *ServicesData) buildTransportAttributeData(
	name string,
	attr *expr.AttributeExpr,
	required bool,
	pointer bool,
	fieldName string,
	fieldType expr.DataType,
	fieldPointer bool,
	validateCtx *codegen.AttributeContext,
	scope *codegen.NameScope,
) *AttributeData {
	varName := scope.Name(codegen.Goify(name, false))
	typeRef := scope.GoTypeRef(attr)
	if pointer {
		typeRef = "*" + typeRef
	}
	validateAttr := attr
	validateTarget := varName
	validateRequired := required
	textUnmarshaler := isStringMetaType(attr)
	if textUnmarshaler {
		validateAttr = attributeWithoutFormatValidation(attr)
		validateTarget = varName + "Raw"
		validateRequired = true
	}
	return &AttributeData{
		Name:              name,
		Description:       attr.Description,
		FieldName:         fieldName,
		FieldPointer:      fieldPointer,
		FieldType:         fieldType,
		VarName:           varName,
		Required:          required,
		Type:              attr.Type,
		TypeName:          scope.GoTypeName(attr),
		TypeRef:           typeRef,
		Pointer:           pointer,
		Validate:          codegen.AttributeValidationCode(validateAttr, nil, validateCtx, validateRequired, expr.IsAlias(attr.Type), validateTarget, name),
		IsTextUnmarshaler: textUnmarshaler,
		DefaultValue:      attr.DefaultValue,
		Example:           attr.Example(sds.Root.API.ExampleGenerator),
	}
}

func isStringMetaType(attr *expr.AttributeExpr) bool {
	if attr == nil || attr.Type == nil || attr.Type.Kind() != expr.StringKind {
		return false
	}
	typeName, _ := codegen.GetMetaType(attr)
	return typeName != ""
}

func attributeWithoutFormatValidation(attr *expr.AttributeExpr) *expr.AttributeExpr {
	if attr == nil || attr.Validation == nil || attr.Validation.Format == "" {
		return attr
	}
	copyAttr := *attr
	copyValidation := *attr.Validation
	copyValidation.Format = ""
	copyAttr.Validation = &copyValidation
	return &copyAttr
}

func (sds *ServicesData) buildTransportElement(
	name string,
	elem string,
	attr *expr.AttributeExpr,
	stringSlice bool,
	required bool,
	pointer bool,
	fieldName string,
	fieldType expr.DataType,
	fieldPointer bool,
	validateCtx *codegen.AttributeContext,
	scope *codegen.NameScope,
) *Element {
	return &Element{
		HTTPName:      elem,
		AttributeName: name,
		StringSlice:   stringSlice,
		Slice:         expr.AsArray(attr.Type) != nil,
		AttributeData: sds.buildTransportAttributeData(name, attr, required, pointer, fieldName, fieldType, fieldPointer, validateCtx, scope),
	}
}

func (sds *ServicesData) extractHeaders(headersIR []*transportir.Header, svcAtt *expr.AttributeExpr, svcCtx *codegen.AttributeContext, scope *codegen.NameScope) []*HeaderData {
	headers := make([]*HeaderData, 0, len(headersIR))
	for _, headerIR := range headersIR {
		name := headerIR.Name
		elem := headerIR.HTTPName
		var attr *expr.AttributeExpr
		if attr = svcAtt.Find(name); attr == nil {
			attr = svcAtt
		}
		stringSlice := transportStringSlice(attr)
		hattr := makeHTTPType(attr)
		pointer := headerIR.PrimitivePointer
		fieldName, fieldType, fieldPointer := transportFieldBinding(name, attr, svcAtt, svcCtx)
		headers = append(headers, &HeaderData{
			CanonicalName: http.CanonicalHeaderKey(elem),
			Element:       sds.buildTransportElement(name, elem, hattr, stringSlice, headerIR.Required, pointer, fieldName, fieldType, fieldPointer, svcCtx, scope),
		})
	}
	return headers
}

func (sds *ServicesData) extractResponseCookies(cookiesIR []*transportir.Cookie, svcAtt *expr.AttributeExpr, svcCtx *codegen.AttributeContext, scope *codegen.NameScope) []*CookieData {
	cookies := make([]*CookieData, 0, len(cookiesIR))
	for _, cookieIR := range cookiesIR {
		name := cookieIR.Name
		if name == "" {
			continue
		}
		cookie := sds.cookieData(name, cookieIR.HTTPName, cookieIR.Required, cookieIR.PrimitivePointer, cookieIR.Attribute, svcAtt, svcCtx, scope)
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

func (sds *ServicesData) cookieData(name, elem string, required bool, pointer bool, mappedAttr *expr.AttributeExpr, svcAtt *expr.AttributeExpr, svcCtx *codegen.AttributeContext, scope *codegen.NameScope) *CookieData {
	var hattr *expr.AttributeExpr
	if hattr = svcAtt.Find(name); hattr == nil {
		if mappedAttr != nil {
			hattr = mappedAttr
		} else {
			hattr = svcAtt
		}
	}
	stringSlice := transportStringSlice(hattr)
	hattr = makeHTTPType(hattr)
	fieldName, fieldType, fieldPointer := transportFieldBinding(name, hattr, svcAtt, svcCtx)
	return &CookieData{
		Element: sds.buildTransportElement(name, elem, hattr, stringSlice, required, pointer, fieldName, fieldType, fieldPointer, svcCtx, scope),
	}
}
