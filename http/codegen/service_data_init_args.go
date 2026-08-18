package codegen

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
)

func buildPayloadFieldArgs(request *RequestData) []*InitArgData {
	args := make([]*InitArgData, 0, len(request.PathParams)+len(request.QueryParams)+len(request.Headers)+len(request.Cookies))
	appendField := func(
		ref string,
		name string,
		varName string,
		description string,
		fieldName string,
		fieldPointer bool,
		fieldType expr.DataType,
		typeName string,
		typeRef string,
		typ expr.DataType,
		pointer bool,
		required bool,
		defaultValue any,
		validate string,
		example any,
	) {
		args = append(args, &InitArgData{
			Ref: ref,
			AttributeData: &AttributeData{
				Name:         name,
				VarName:      varName,
				Description:  description,
				FieldName:    fieldName,
				FieldPointer: fieldPointer,
				FieldType:    fieldType,
				TypeName:     typeName,
				TypeRef:      typeRef,
				Type:         typ,
				Pointer:      pointer,
				Required:     required,
				DefaultValue: defaultValue,
				Validate:     validate,
				Example:      example,
			},
		})
	}
	for _, param := range request.PathParams {
		appendField(param.VarName, param.Name, param.VarName, param.Description, param.FieldName, param.FieldPointer, param.FieldType, param.TypeName, param.TypeRef, param.Type, param.Pointer, param.Required, nil, param.Validate, param.Example)
	}
	for _, param := range request.QueryParams {
		appendField(param.VarName, param.Name, param.VarName, "", param.FieldName, param.FieldPointer, param.FieldType, param.TypeName, param.TypeRef, param.Type, param.Pointer, param.Required, param.DefaultValue, param.Validate, param.Example)
	}
	for _, header := range request.Headers {
		appendField(header.VarName, header.Name, header.VarName, "", header.FieldName, header.FieldPointer, header.FieldType, header.TypeName, header.TypeRef, header.Type, header.Pointer, header.Required, header.DefaultValue, header.Validate, header.Example)
	}
	for _, cookie := range request.Cookies {
		if cookie.FieldName == "" {
			continue
		}
		appendField(cookie.VarName, cookie.Name, cookie.VarName, "", cookie.FieldName, cookie.FieldPointer, cookie.FieldType, cookie.TypeName, cookie.TypeRef, cookie.Type, cookie.Pointer, cookie.Required, cookie.DefaultValue, cookie.Validate, cookie.Example)
	}
	return args
}

func buildBasicAuthCLIArgs(ep *service.MethodData, payload *expr.AttributeExpr, svc *service.Data, httpsvrctx *codegen.AttributeContext, generator *expr.ExampleGenerator) []*InitArgData {
	for _, requirement := range ep.Requirements {
		for _, scheme := range requirement.Schemes {
			if scheme.Type != "Basic" {
				continue
			}
			uatt := payload.Find(scheme.UsernameAttr)
			uref := svc.Scope.GoTypeRef(uatt)
			if scheme.UsernamePointer {
				uref = "*" + uref
			}
			patt := payload.Find(scheme.PasswordAttr)
			pref := svc.Scope.GoTypeRef(patt)
			if scheme.PasswordPointer {
				pref = "*" + pref
			}
			return []*InitArgData{
				{
					Ref: scheme.UsernameAttr,
					AttributeData: &AttributeData{
						Name:         scheme.UsernameAttr,
						VarName:      scheme.UsernameAttr,
						FieldName:    scheme.UsernameField,
						FieldPointer: scheme.UsernamePointer,
						FieldType:    uatt.Type,
						Description:  uatt.Description,
						Required:     scheme.UsernameRequired,
						TypeName:     svc.Scope.GoTypeName(uatt),
						TypeRef:      uref,
						Type:         uatt.Type,
						Pointer:      scheme.UsernamePointer,
						Validate:     codegen.ValidationCode(uatt, nil, httpsvrctx, scheme.UsernameRequired, expr.IsAlias(uatt.Type), false, scheme.UsernameAttr),
						Example:      uatt.Example(generator),
					},
				},
				{
					Ref: scheme.PasswordAttr,
					AttributeData: &AttributeData{
						Name:         scheme.PasswordAttr,
						VarName:      scheme.PasswordAttr,
						FieldName:    scheme.PasswordField,
						FieldPointer: scheme.PasswordPointer,
						FieldType:    patt.Type,
						Description:  patt.Description,
						Required:     scheme.PasswordRequired,
						TypeName:     svc.Scope.GoTypeName(patt),
						TypeRef:      pref,
						Type:         patt.Type,
						Pointer:      scheme.PasswordPointer,
						Validate:     codegen.ValidationCode(patt, nil, httpsvrctx, scheme.PasswordRequired, expr.IsAlias(patt.Type), false, scheme.PasswordAttr),
						Example:      patt.Example(generator),
					},
				},
			}
		}
	}
	return nil
}

func buildBodyInitArg(scope *codegen.NameScope, body *expr.AttributeExpr, addressObject bool) *InitArgData {
	ref := "body"
	if addressObject && (expr.IsObject(body.Type) || expr.IsUnion(body.Type)) && !expr.IsNullable(body) {
		ref = "&body"
	}
	return &InitArgData{
		Ref: ref,
		AttributeData: &AttributeData{
			Name:    "body",
			VarName: "body",
			TypeRef: bodyTypeRef(scope, body),
		},
	}
}

func buildHeaderInitArgs(headers []*HeaderData) []*InitArgData {
	args := make([]*InitArgData, 0, len(headers))
	for _, header := range headers {
		args = append(args, &InitArgData{
			Ref: header.VarName,
			AttributeData: &AttributeData{
				Name:         header.Name,
				VarName:      header.VarName,
				FieldName:    header.FieldName,
				FieldPointer: header.FieldPointer,
				FieldType:    header.FieldType,
				Required:     header.Required,
				Pointer:      header.Pointer,
				TypeRef:      header.TypeRef,
				Type:         header.Type,
				Validate:     header.Validate,
				Example:      header.Example,
			},
		})
	}
	return args
}

func buildCookieInitArgs(cookies []*CookieData) []*InitArgData {
	args := make([]*InitArgData, 0, len(cookies))
	for _, cookie := range cookies {
		args = append(args, &InitArgData{
			Ref: cookie.VarName,
			AttributeData: &AttributeData{
				Name:         cookie.Name,
				VarName:      cookie.VarName,
				FieldName:    cookie.FieldName,
				FieldPointer: cookie.FieldPointer,
				FieldType:    cookie.FieldType,
				Required:     cookie.Required,
				Pointer:      cookie.Pointer,
				TypeRef:      cookie.TypeRef,
				Type:         cookie.Type,
				Validate:     cookie.Validate,
				Example:      cookie.Example,
			},
		})
	}
	return args
}

func responseFieldsNeedValidation(headers []*HeaderData, cookies []*CookieData) bool {
	for _, header := range headers {
		if header.Validate != "" || header.Required || needConversion(header.Type) || header.IsTextUnmarshaler {
			return true
		}
	}
	for _, cookie := range cookies {
		if cookie.Validate != "" || cookie.Required || needConversion(cookie.Type) || cookie.IsTextUnmarshaler {
			return true
		}
	}
	return false
}
