package codegen

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

func jsonrpcResponseDecoderSection(e *httpcodegen.EndpointData) codegen.Section {
	return codegen.MustJenniferSection("jsonrpc-response-decoder", func(stmt *jen.Statement) {
		comment := fmt.Sprintf("%s returns a decoder for responses returned by the %s service %s JSON-RPC method. restoreBody controls whether the response body should be restored after having been read.", e.ResponseDecoder, e.ServiceName, e.Method.Name)
		codegen.Doc(stmt, comment)
		stmt.Func().Id(e.ResponseDecoder).
			Params(
				jen.Id("decoder").Func().Params(jen.Op("*").Qual("net/http", "Response")).Add(codegen.TypeRef("loomhttp.Decoder")),
				jen.Id("restoreBody").Bool(),
			).
			Params(
				jen.Func().
					Params(jen.Op("*").Qual("net/http", "Response")).
					Params(jen.Any(), jen.Error()),
			).
			Block(
				jen.Return(
					jen.Func().
						Params(jen.Id("resp").Op("*").Qual("net/http", "Response")).
						Params(jen.Any(), jen.Error()).
						BlockFunc(func(g *jen.Group) {
							writeJSONRPCResponseDecoderBody(g, e)
						}),
				),
			)
	})
}

func writeJSONRPCResponseDecoderBody(g *jen.Group, e *httpcodegen.EndpointData) {
	writeJSONRPCResponseRestoreBody(g)
	writeJSONRPCResponseStatusCheck(g, e)
	writeJSONRPCResponseDecodeEnvelope(g, e)
	writeJSONRPCResponseErrorHandling(g, e)
	writeJSONRPCResponseSuccessHandling(g, e)
}

func writeJSONRPCResponseRestoreBody(g *jen.Group) {
	g.If(jen.Id("restoreBody")).Block(
		jen.List(jen.Id("b"), jen.Id("err")).Op(":=").Qual("io", "ReadAll").Call(jen.Id("resp").Dot("Body")),
		jen.If(jen.Id("err").Op("!=").Nil()).Block(
			jen.Return(jen.Nil(), jen.Id("err")),
		),
		jen.Id("resp").Dot("Body").Op("=").Qual("io", "NopCloser").Call(jen.Qual("bytes", "NewBuffer").Call(jen.Id("b"))),
		jen.Defer().Func().Params().Block(
			jen.Id("resp").Dot("Body").Op("=").Qual("io", "NopCloser").Call(jen.Qual("bytes", "NewBuffer").Call(jen.Id("b"))),
		).Call(),
	)
	g.Add(codegen.Expr("defer resp.Body.Close()"))
	g.Line()
}

func writeJSONRPCResponseStatusCheck(g *jen.Group, e *httpcodegen.EndpointData) {
	g.If(jen.Id("resp").Dot("StatusCode").Op("!=").Qual("net/http", "StatusOK")).Block(
		jen.List(jen.Id("body"), jen.Id("_")).Op(":=").Qual("io", "ReadAll").Call(jen.Id("resp").Dot("Body")),
		jen.Return(
			jen.Nil(),
			errInvalidResponseExpr(e.ServiceName, e.Method.Name, jen.Id("resp").Dot("StatusCode"), jen.String().Call(jen.Id("body"))),
		),
	)
	g.Line()
}

func writeJSONRPCResponseDecodeEnvelope(g *jen.Group, e *httpcodegen.EndpointData) {
	g.Var().Id("jresp").Qual("github.com/CaliLuke/loom/jsonrpc", "RawResponse")
	g.If(
		jen.Err().Op(":=").Id("decoder").Call(jen.Id("resp")).Dot("Decode").Call(jen.Op("&").Id("jresp")),
		jen.Err().Op("!=").Nil(),
	).Block(
		jen.Return(
			jen.Nil(),
			errDecodingExpr(e.ServiceName, e.Method.Name, jen.Id("err")),
		),
	)
	g.Line()
}

func writeJSONRPCResponseErrorHandling(g *jen.Group, e *httpcodegen.EndpointData) {
	g.If(jen.Id("jresp").Dot("Error").Op("!=").Nil()).BlockFunc(func(eg *jen.Group) {
		eg.Switch(jen.Id("jresp").Dot("Error").Dot("Code")).BlockFunc(func(sg *jen.Group) {
			writeJSONRPCErrorDecodeSwitch(sg, e)
			sg.Default().Block(
				jen.Return(jen.Nil(), jen.Id("jresp").Dot("Error")),
			)
		})
	})
	g.Line()
}

func writeJSONRPCResponseSuccessHandling(g *jen.Group, e *httpcodegen.EndpointData) {
	if e.Result == nil || len(e.Result.Responses) == 0 {
		g.Return(jen.Nil(), jen.Nil())
		return
	}
	resp := e.Result.Responses[0]
	g.Id("resp").Dot("Body").Op("=").Qual("io", "NopCloser").Call(jen.Qual("bytes", "NewBuffer").Call(jen.Id("jresp").Dot("Result")))
	writeSingleResponseDecode(g, resp, e.ServiceName, e.Method)
	writeJSONRPCDecodedResponseReturn(g, e, resp)
}

func writeJSONRPCDecodedResponseReturn(g *jen.Group, e *httpcodegen.EndpointData, resp *httpcodegen.ResponseData) {
	switch {
	case resp.ResultInit != nil:
		writeJSONRPCDecodedInitReturn(g, e, resp)
	case resp.ClientBody != nil:
		g.Return(jen.Id("body"), jen.Nil())
	case len(resp.Headers) > 0:
		g.Return(jen.Id(resp.Headers[0].VarName), jen.Nil())
	case len(resp.Cookies) > 0:
		g.Return(jen.Id(resp.Cookies[0].VarName), jen.Nil())
	default:
		g.Return(jen.Nil(), jen.Nil())
	}
}

func writeJSONRPCDecodedInitReturn(g *jen.Group, e *httpcodegen.EndpointData, resp *httpcodegen.ResponseData) {
	if resp.ViewedResult != nil {
		writeJSONRPCViewedInitReturn(g, e, resp)
		return
	}
	g.Id("res").Op(":=").Id(resp.ResultInit.Name).Call(jsonrpcInitArgs(resp.ResultInit.ClientArgs)...)
	writeJSONRPCResponseTagAssignment(g, resp)
	g.Return(jen.Id("res"), jen.Nil())
}

func writeJSONRPCViewedInitReturn(g *jen.Group, e *httpcodegen.EndpointData, resp *httpcodegen.ResponseData) {
	g.Id("p").Op(":=").Id(resp.ResultInit.Name).Call(jsonrpcInitArgs(resp.ResultInit.ClientArgs)...)
	if resp.TagName != "" {
		g.Id("tmp").Op(":=").Lit(resp.TagValue)
		g.Id("p").Dot(resp.TagName).Op("=").Op("&").Id("tmp")
	}
	if e.Method.ViewedResult != nil && e.Method.ViewedResult.ViewName != "" {
		g.Id("view").Op(":=").Lit(e.Method.ViewedResult.ViewName)
	} else {
		g.Id("view").Op(":=").Id("resp").Dot("Header").Dot("Get").Call(jen.Lit("loom-view"))
	}
	g.Id("vres").Op(":=").Add(codegen.Expr(viewedResultPrefix(e.Method.ViewedResult) + e.Method.ViewedResult.ViewsPkg + "." + e.Method.ViewedResult.VarName + "{Projected: p, View: view}"))
	if resp.ClientBody != nil {
		g.If(
			jen.Id("err").Op("=").Add(codegen.Expr(e.Method.ViewedResult.ViewsPkg+".Validate"+e.Method.Result+"(vres)")),
			jen.Id("err").Op("!=").Nil(),
		).Block(
			jen.Return(jen.Nil(), errValidationExpr(e.ServiceName, e.Method.Name, jen.Id("err"))),
		)
	}
	g.Id("res").Op(":=").Add(codegen.Expr(e.ServicePkgName + "." + e.Method.ViewedResult.ResultInit.Name + "(vres)"))
	g.Return(jen.Id("res"), jen.Nil())
}

func jsonrpcInitArgs(args []*httpcodegen.InitArgData) []jen.Code {
	initArgs := make([]jen.Code, 0, len(args))
	for _, arg := range args {
		initArgs = append(initArgs, codegen.Expr(arg.Ref))
	}
	return initArgs
}

func writeJSONRPCResponseTagAssignment(g *jen.Group, resp *httpcodegen.ResponseData) {
	if resp.TagName == "" || isViewedResponse(resp) {
		return
	}
	if resp.TagPointer {
		g.Id("tmp").Op(":=").Lit(resp.TagValue)
		g.Id("res").Dot(resp.TagName).Op("=").Op("&").Id("tmp")
		return
	}
	g.Id("res").Dot(resp.TagName).Op("=").Lit(resp.TagValue)
}

func writeJSONRPCErrorDecodeSwitch(g *jen.Group, e *httpcodegen.EndpointData) {
	for _, group := range e.Errors {
		if len(group.Errors) == 0 || group.Errors[0].Response == nil {
			continue
		}
		g.Case(codegen.Expr(group.StatusCode)).BlockFunc(func(cg *jen.Group) {
			if len(group.Errors) > 1 {
				writeJSONRPCNamedErrorDecode(cg, group, e)
				return
			}
			writeJSONRPCErrorResponseDecode(cg, group.Errors[0].Response, e.ServiceName, e.Method)
			writeResultInitReturn(cg, group.Errors[0].Response)
		})
	}
}

func writeJSONRPCNamedErrorDecode(g *jen.Group, group *httpcodegen.ErrorGroupData, e *httpcodegen.EndpointData) {
	g.Var().Id("jerrData").Qual("github.com/CaliLuke/loom/jsonrpc", "ErrorData")
	g.If(jen.Len(jen.Id("jresp").Dot("Error").Dot("Data")).Op(">").Lit(0)).Block(
		jen.If(
			jen.Err().Op(":=").Qual("encoding/json", "Unmarshal").Call(jen.Id("jresp").Dot("Error").Dot("Data"), jen.Op("&").Id("jerrData")),
			jen.Err().Op("!=").Nil(),
		).Block(
			jen.Return(jen.Nil(), errDecodingExpr(e.ServiceName, e.Method.Name, jen.Id("err"))),
		),
	)
	g.Switch(jen.Id("jerrData").Dot("Name")).BlockFunc(func(sg *jen.Group) {
		for _, item := range group.Errors {
			if item.Response == nil {
				continue
			}
			sg.Case(jen.Lit(item.Name)).BlockFunc(func(cg *jen.Group) {
				writeJSONRPCErrorResponseDecode(cg, item.Response, e.ServiceName, e.Method)
				writeResultInitReturn(cg, item.Response)
			})
		}
		sg.Default().Block(
			jen.Return(jen.Nil(), errInvalidResponseExpr(e.ServiceName, e.Method.Name, jen.Id("resp").Dot("StatusCode"), jen.String().Call(jen.Id("jresp").Dot("Error").Dot("Data")))),
		)
	})
}

func writeJSONRPCErrorResponseDecode(g *jen.Group, data *httpcodegen.ResponseData, serviceName string, method *service.MethodData) {
	g.Id("resp").Dot("Body").Op("=").Qual("io", "NopCloser").Call(jen.Qual("bytes", "NewBuffer").Call(jen.Id("jresp").Dot("Error").Dot("Data")))
	writeSingleResponseDecode(g, data, serviceName, method)
}

func writeResultInitReturn(g *jen.Group, resp *httpcodegen.ResponseData) {
	switch {
	case resp.ResultInit != nil:
		g.Return(jen.Nil(), jen.Id(resp.ResultInit.Name).Call(jsonrpcInitArgs(resp.ResultInit.ClientArgs)...))
	case resp.ClientBody != nil:
		g.Return(jen.Nil(), jen.Id("body"))
	default:
		g.Return(jen.Nil(), jen.Nil())
	}
}

func writeSingleResponseDecode(g *jen.Group, data *httpcodegen.ResponseData, serviceName string, method *service.MethodData) {
	if data.ClientBody != nil {
		g.Var().Defs(
			jen.Id("body").Add(codegen.TypeRef(data.ClientBody.VarName)),
			jen.Id("err").Error(),
		)
		g.Id("err").Op("=").Id("decoder").Call(jen.Id("resp")).Dot("Decode").Call(jen.Op("&").Id("body"))
		g.If(jen.Id("err").Op("!=").Nil()).Block(
			jen.Return(jen.Nil(), errDecodingExpr(serviceName, method.Name, jen.Id("err"))),
		)
		if data.ClientBody.ValidateRef != "" {
			g.Add(codegen.Expr(data.ClientBody.ValidateRef))
			g.If(jen.Id("err").Op("!=").Nil()).Block(
				jen.Return(jen.Nil(), errValidationExpr(serviceName, method.Name, jen.Id("err"))),
			)
		}
	}
	if len(data.Headers) > 0 {
		writeResponseHeaderBlock(g, data)
	}
	if len(data.Cookies) > 0 {
		writeResponseCookieBlock(g, data)
	}
	if data.MustValidate {
		g.If(jen.Id("err").Op("!=").Nil()).Block(
			jen.Return(jen.Nil(), errValidationExpr(serviceName, method.Name, jen.Id("err"))),
		)
	}
}

func writeResponseHeaderBlock(g *jen.Group, data *httpcodegen.ResponseData) {
	defs := []jen.Code{}
	for _, header := range data.Headers {
		defs = append(defs, jen.Id(header.VarName).Add(codegen.TypeRef(header.TypeRef)))
	}
	if data.ClientBody == nil && data.MustValidate {
		defs = append(defs, jen.Id("err").Error())
	}
	g.Var().Defs(defs...)
	for _, header := range data.Headers {
		writeResponseHeaderDecode(g, header)
		if header.Validate != "" {
			g.Add(codegen.Expr(header.Validate))
		}
	}
}

func writeResponseHeaderDecode(g *jen.Group, h *httpcodegen.HeaderData) {
	switch {
	case h.Type.Name() == "string" || h.Type.Name() == "any":
		g.Id(h.VarName + "Raw").Op(":=").Id("resp").Dot("Header").Dot("Get").Call(jen.Lit(h.CanonicalName))
		if h.Required {
			g.If(jen.Id(h.VarName + "Raw").Op("==").Lit("")).Block(
				jen.Id("err").Op("=").Add(missingFieldMergeExpr(jen.Id("err"), h.Name, "header")),
			)
			g.Id(h.VarName).Op("=").Add(codegen.Expr(stringPointerPrefix(h.Type.Name(), h.Pointer) + h.VarName + "Raw"))
		} else {
			g.If(jen.Id(h.VarName + "Raw").Op("!=").Lit("")).Block(
				jen.Id(h.VarName).Op("=").Add(codegen.Expr(stringPointerPrefix(h.Type.Name(), h.Pointer) + h.VarName + "Raw")),
			).Else().BlockFunc(func(eg *jen.Group) {
				if h.DefaultValue != nil {
					eg.Id(h.VarName).Op("=").Add(codegen.Expr(literalValue(h.Type.Name(), h.DefaultValue)))
				}
			})
		}
	case h.StringSlice:
		g.Id(h.VarName).Op("=").Id("resp").Dot("Header").Index(jen.Lit(h.CanonicalName))
		if h.Required {
			g.If(jen.Id(h.VarName).Op("==").Nil()).Block(
				jen.Id("err").Op("=").Add(missingFieldMergeExpr(jen.Id("err"), h.Name, "header")),
			)
		}
	case h.Slice:
		g.BlockFunc(func(bg *jen.Group) {
			bg.Id(h.VarName + "Raw").Op(":=").Id("resp").Dot("Header").Index(jen.Lit(h.CanonicalName))
			if h.Required {
				bg.If(jen.Id(h.VarName + "Raw").Op("==").Nil()).Block(
					jen.Return(jen.Nil(), errValidationExpr("", "", missingFieldExpr(h.Name, "header"))),
				)
			}
			writeElementSliceConversion(bg, h.AttributeData)
		})
	default:
		g.BlockFunc(func(bg *jen.Group) {
			bg.Id(h.VarName + "Raw").Op(":=").Id("resp").Dot("Header").Dot("Get").Call(jen.Lit(h.CanonicalName))
			if h.Required {
				bg.If(jen.Id(h.VarName + "Raw").Op("==").Lit("")).Block(
					jen.Return(jen.Nil(), errValidationExpr("", "", missingFieldExpr(h.Name, "header"))),
				)
			}
			writeQueryTypeConversion(bg, h.AttributeData)
		})
	}
}

func writeResponseCookieBlock(g *jen.Group, data *httpcodegen.ResponseData) {
	g.Add(codegen.Expr("var ("))
	for _, cookie := range data.Cookies {
		g.Add(codegen.Expr(cookie.VarName + " " + cookie.TypeRef))
		g.Add(codegen.Expr(cookie.VarName + "Raw string"))
	}
	g.Add(codegen.Expr("cookies = resp.Cookies()"))
	if data.ClientBody == nil && data.MustValidate && len(data.Headers) == 0 {
		g.Add(codegen.Expr("err error"))
	}
	g.Add(codegen.Expr(")"))
	g.For(
		jen.List(jen.Id("_"), jen.Id("c")).Op(":=").Range().Id("cookies"),
	).BlockFunc(func(fg *jen.Group) {
		fg.Switch(jen.Id("c").Dot("Name")).BlockFunc(func(sg *jen.Group) {
			for _, cookie := range data.Cookies {
				sg.Case(jen.Lit(cookie.HTTPName)).Block(
					jen.Id(cookie.VarName + "Raw").Op("=").Id("c").Dot("Value"),
				)
			}
		})
	})
	for _, cookie := range data.Cookies {
		writeResponseCookieDecode(g, cookie)
		if cookie.Validate != "" {
			g.Add(codegen.Expr(cookie.Validate))
		}
	}
}

func writeResponseCookieDecode(g *jen.Group, c *httpcodegen.CookieData) {
	if c.Type.Name() == "string" || c.Type.Name() == "any" {
		if c.Required {
			g.If(jen.Id(c.VarName + "Raw").Op("==").Lit("")).Block(
				jen.Id("err").Op("=").Add(missingFieldMergeExpr(jen.Id("err"), c.Name, "cookie")),
			)
			g.Id(c.VarName).Op("=").Add(codegen.Expr(stringPointerPrefix(c.Type.Name(), c.Pointer) + c.VarName + "Raw"))
		} else {
			g.If(jen.Id(c.VarName + "Raw").Op("!=").Lit("")).Block(
				jen.Id(c.VarName).Op("=").Add(codegen.Expr(stringPointerPrefix(c.Type.Name(), c.Pointer) + c.VarName + "Raw")),
			)
		}
		return
	}
	g.BlockFunc(func(bg *jen.Group) {
		if c.Required {
			bg.If(jen.Id(c.VarName + "Raw").Op("==").Lit("")).Block(
				jen.Return(jen.Nil(), errValidationExpr("", "", missingFieldExpr(c.Name, "cookie"))),
			)
		}
		writeQueryTypeConversion(bg, c.AttributeData)
	})
}

func writeElementSliceConversion(g *jen.Group, a *httpcodegen.AttributeData) {
	g.Id(a.VarName).Op("=").Make(codegen.TypeRef(a.TypeRef), jen.Len(jen.Id(a.VarName+"Raw")))
	g.For(
		jen.List(jen.Id("i"), jen.Id("rv")).Op(":=").Range().Id(a.VarName + "Raw"),
	).BlockFunc(func(fg *jen.Group) {
		writeSliceItemConversion(fg, a)
	})
}

func writeSliceItemConversion(g *jen.Group, a *httpcodegen.AttributeData) {
	arr := expr.AsArray(a.Type)
	if arr == nil {
		g.Comment("unsupported non-array type for var " + a.VarName)
		return
	}
	if writeScalarSliceItemConversion(g, a, arr.ElemType.Type.Name()) {
		return
	}
	g.Id(a.VarName).Index(jen.Id("i")).Op("=").Id("rv")
}

func writeQueryTypeConversion(g *jen.Group, a *httpcodegen.AttributeData) {
	if writeScalarQueryTypeConversion(g, a, a.Type.Name()) {
		return
	}
	g.Comment("unsupported type " + a.Type.Name() + " for var " + a.VarName)
}

type scalarParseSpec struct {
	expr        string
	description string
	assignType  string
	assignFn    func(*jen.Group, *httpcodegen.AttributeData, string)
}

func writeScalarSliceItemConversion(g *jen.Group, a *httpcodegen.AttributeData, typeName string) bool {
	if typeName == "bytes" {
		g.Id(a.VarName).Index(jen.Id("i")).Op("=").Index().Byte().Call(jen.Id("rv"))
		return true
	}
	spec, ok := scalarParseSpecs()[typeName]
	if !ok {
		return false
	}
	g.Add(codegen.Expr(spec.expr))
	g.If(jen.Id("err2").Op("!=").Nil()).Block(
		jen.Id("err").Op("=").Add(invalidFieldTypeMergeExpr(jen.Id("err"), a.Name, jen.Id(a.VarName+"Raw"), "array of "+spec.description+"s")),
	)
	g.Id(a.VarName).Index(jen.Id("i")).Op("=").Add(codegen.Expr(spec.assignType + "(v)"))
	return true
}

func writeScalarQueryTypeConversion(g *jen.Group, a *httpcodegen.AttributeData, typeName string) bool {
	if typeName == "bytes" {
		g.Id(a.VarName).Op("=").Index().Byte().Call(jen.Id(a.VarName + "Raw"))
		return true
	}
	spec, ok := scalarParseSpecs()[typeName]
	if !ok {
		return false
	}
	g.Add(codegen.Expr(strings.ReplaceAll(spec.expr, "rv", a.VarName+"Raw")))
	g.If(jen.Id("err2").Op("!=").Nil()).Block(
		jen.Id("err").Op("=").Add(invalidFieldTypeMergeExpr(jen.Id("err"), a.Name, jen.Id(a.VarName+"Raw"), spec.description)),
	)
	spec.assignFn(g, a, spec.assignType)
	return true
}

func scalarParseSpecs() map[string]scalarParseSpec {
	return map[string]scalarParseSpec{
		"int":     {expr: "v, err2 := strconv.ParseInt(rv, 10, strconv.IntSize)", description: "integer", assignType: "int", assignFn: assignConverted},
		"int32":   {expr: "v, err2 := strconv.ParseInt(rv, 10, 32)", description: "integer", assignType: "int32", assignFn: assignConverted},
		"int64":   {expr: "v, err2 := strconv.ParseInt(rv, 10, 64)", description: "integer", assignType: "int64", assignFn: assignDirectOrCast},
		"uint":    {expr: "v, err2 := strconv.ParseUint(rv, 10, strconv.IntSize)", description: "unsigned integer", assignType: "uint", assignFn: assignConverted},
		"uint32":  {expr: "v, err2 := strconv.ParseUint(rv, 10, 32)", description: "unsigned integer", assignType: "uint32", assignFn: assignConverted},
		"uint64":  {expr: "v, err2 := strconv.ParseUint(rv, 10, 64)", description: "unsigned integer", assignType: "uint64", assignFn: assignDirectOrCast},
		"float32": {expr: "v, err2 := strconv.ParseFloat(rv, 32)", description: "float", assignType: "float32", assignFn: assignConverted},
		"float64": {expr: "v, err2 := strconv.ParseFloat(rv, 64)", description: "float", assignType: "float64", assignFn: assignDirectOrCast},
		"boolean": {expr: "v, err2 := strconv.ParseBool(rv)", description: "boolean", assignType: "bool", assignFn: assignDirectOrCast},
	}
}

func assignConverted(g *jen.Group, a *httpcodegen.AttributeData, baseType string) {
	targetType := baseType
	if a.TypeRef != "" {
		targetType = strings.TrimPrefix(a.TypeRef, "*")
	}
	if a.Pointer {
		g.Id("pv").Op(":=").Add(codegen.Expr(targetType + "(v)"))
		g.Id(a.VarName).Op("=").Op("&").Id("pv")
		return
	}
	g.Id(a.VarName).Op("=").Add(codegen.Expr(targetType + "(v)"))
}

func assignDirectOrCast(g *jen.Group, a *httpcodegen.AttributeData, builtin string) {
	if a.TypeRef != "" && a.TypeRef != builtin && a.TypeRef != "*"+builtin {
		if a.Pointer {
			g.Id(a.VarName).Op("=").Add(codegen.Expr("(" + a.TypeRef + ")(&v)"))
		} else {
			g.Id(a.VarName).Op("=").Add(codegen.Expr("(" + a.TypeRef + ")(v)"))
		}
		return
	}
	if a.Pointer {
		g.Id(a.VarName).Op("=").Op("&").Id("v")
	} else {
		g.Id(a.VarName).Op("=").Id("v")
	}
}

func literalValue(typeName string, v any) string {
	if typeName == "string" {
		return fmt.Sprintf("%q", v)
	}
	return fmt.Sprintf("%#v", v)
}

func stringPointerPrefix(typeName string, pointer bool) string {
	if typeName == "string" && pointer {
		return "&"
	}
	return ""
}

func isViewedResponse(resp *httpcodegen.ResponseData) bool {
	return resp.ViewedResult != nil
}

func viewedResultPrefix(v *service.ViewedResultTypeData) string {
	if v == nil || v.IsCollection {
		return ""
	}
	return "&"
}

func errInvalidResponseExpr(serviceName, methodName string, statusCode, body jen.Code) jen.Code {
	return jen.Id("loomhttp").Dot("ErrInvalidResponse").Call(
		jen.Lit(serviceName),
		jen.Lit(methodName),
		statusCode,
		body,
	)
}

func errDecodingExpr(serviceName, methodName string, errExpr jen.Code) jen.Code {
	return jen.Id("loomhttp").Dot("ErrDecodingError").Call(
		jen.Lit(serviceName),
		jen.Lit(methodName),
		errExpr,
	)
}

func errValidationExpr(serviceName, methodName string, errExpr jen.Code) jen.Code {
	return jen.Id("loomhttp").Dot("ErrValidationError").Call(
		jen.Lit(serviceName),
		jen.Lit(methodName),
		errExpr,
	)
}

func missingFieldExpr(name, location string) jen.Code {
	return jen.Qual("github.com/CaliLuke/loom", "MissingFieldError").Call(
		jen.Lit(name),
		jen.Lit(location),
	)
}

func missingFieldMergeExpr(currentErr jen.Code, name, location string) jen.Code {
	return jen.Qual("github.com/CaliLuke/loom", "MergeErrors").Call(
		currentErr,
		missingFieldExpr(name, location),
	)
}

func invalidFieldTypeMergeExpr(currentErr jen.Code, name string, actual jen.Code, expected string) jen.Code {
	return jen.Qual("github.com/CaliLuke/loom", "MergeErrors").Call(
		currentErr,
		jen.Qual("github.com/CaliLuke/loom", "InvalidFieldTypeError").Call(
			jen.Lit(name),
			actual,
			jen.Lit(expected),
		),
	)
}
