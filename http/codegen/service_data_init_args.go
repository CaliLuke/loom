package codegen

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
)

func buildPayloadFieldArgs(request *RequestData) []*InitArgData {
	args := make([]*InitArgData, 0, len(request.PathParams)+len(request.QueryParams)+len(request.Headers)+len(request.Cookies))
	appendField := func(attr *AttributeData, description string, defaultValue any) {
		field := *attr
		field.Description = description
		field.DefaultValue = defaultValue
		args = append(args, &InitArgData{Ref: attr.VarName, AttributeData: &field})
	}
	for _, param := range request.PathParams {
		appendField(param.AttributeData, param.Description, nil)
	}
	for _, param := range request.QueryParams {
		appendField(param.AttributeData, "", param.DefaultValue)
	}
	for _, header := range request.Headers {
		appendField(header.AttributeData, "", header.DefaultValue)
	}
	for _, cookie := range request.Cookies {
		if cookie.FieldName == "" {
			continue
		}
		appendField(cookie.AttributeData, "", cookie.DefaultValue)
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

func buildBodyInitArg(
	scope *codegen.NameScope,
	body *expr.AttributeExpr,
	addressObject bool,
	httpctx *codegen.AttributeContext,
) *InitArgData {
	ref := "body"
	if addressObject && (expr.IsObject(body.Type) || expr.IsUnion(body.Type)) && !expr.IsNullable(body) {
		ref = "&body"
	}
	return &InitArgData{
		Ref: ref,
		AttributeData: &AttributeData{
			Name:    "body",
			VarName: "body",
			TypeRef: goBodyTypeRef(scope, body, httpctx),
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
