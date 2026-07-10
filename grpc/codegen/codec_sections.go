//nolint:errcheck // Generator helpers write only to in-memory builders.
package codegen

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
)

func grpcRequestEncoderSection(endpoint *EndpointData) codegen.Section {
	return codegen.NewJenniferSection("request-encoder", func(stmt *jen.Statement) {
		stmt.Add(codegen.Expr(renderGRPCRequestEncoder(endpoint)))
	})
}

func grpcResponseDecoderSection(endpoint *EndpointData) codegen.Section {
	return codegen.NewJenniferSection("response-decoder", func(stmt *jen.Statement) {
		stmt.Add(codegen.Expr(renderGRPCResponseDecoder(endpoint)))
	})
}

func grpcResponseEncoderSection(endpoint *EndpointData) codegen.Section {
	return codegen.NewJenniferSection("response-encoder", func(stmt *jen.Statement) {
		stmt.Add(codegen.Expr(renderGRPCResponseEncoder(endpoint)))
	})
}

func grpcRequestDecoderSection(endpoint *EndpointData) codegen.Section {
	return codegen.NewJenniferSection("request-decoder", func(stmt *jen.Statement) {
		stmt.Add(codegen.Expr(renderGRPCRequestDecoder(endpoint)))
	})
}

func renderGRPCRequestEncoder(endpoint *EndpointData) string {
	var b sourceBuilder
	fmt.Fprintf(&b, "%s\n", codegen.Comment("Encode"+endpoint.Method.VarName+"Request encodes requests sent to "+endpoint.ServiceName+" "+endpoint.Method.Name+" endpoint."))
	fmt.Fprintf(&b, "func Encode%sRequest(ctx context.Context, v any, md *metadata.MD) (any, error) {\n", endpoint.Method.VarName)
	fmt.Fprintf(&b, "\tpayload, ok := v.(%s)\n", endpoint.PayloadRef)
	b.Add("\tif !ok {\n")
	fmt.Fprintf(&b, "\t\treturn nil, loomgrpc.ErrInvalidType(%q, %q, %q, v)\n", endpoint.ServiceName, endpoint.Method.Name, endpoint.PayloadRef)
	b.Add("\t}\n")
	for _, md := range endpoint.Request.Metadata {
		b.Add(renderGRPCMetadataAppend(md, "payload", endpoint.MetadataSchemes))
	}
	if endpoint.Request.ClientConvert != nil {
		if endpoint.Request.StreamEnvelope != nil {
			if endpoint.Request.ClientConvert.Init.ErrorAware {
				fmt.Fprintf(&b, "\tmessage, err := %s(%s)\n", endpoint.Request.ClientConvert.Init.Name, renderInitArgList(endpoint.Request.ClientConvert.Init.Args))
				b.Add("\tif err != nil {\n\t\treturn nil, err\n\t}\n")
			} else {
				fmt.Fprintf(&b, "\tmessage := %s(%s)\n", endpoint.Request.ClientConvert.Init.Name, renderInitArgList(endpoint.Request.ClientConvert.Init.Args))
			}
			fmt.Fprintf(&b, "\treturn &%s.%s{\n", endpoint.PkgName, endpoint.Request.Message.VarName)
			fmt.Fprintf(&b, "\t\t%s: &%s{\n", endpoint.Request.StreamEnvelope.FieldName, endpoint.Request.StreamEnvelope.InitialWrapperRef)
			fmt.Fprintf(&b, "\t\t\t%s: message,\n", endpoint.Request.StreamEnvelope.InitialFieldName)
			b.Add("\t\t},\n")
			b.Add("\t}, nil\n")
		} else {
			if endpoint.Request.ClientConvert.Init.ErrorAware {
				fmt.Fprintf(&b, "\treturn %s(%s)\n", endpoint.Request.ClientConvert.Init.Name, renderInitArgList(endpoint.Request.ClientConvert.Init.Args))
			} else {
				fmt.Fprintf(&b, "\treturn %s(%s), nil\n", endpoint.Request.ClientConvert.Init.Name, renderInitArgList(endpoint.Request.ClientConvert.Init.Args))
			}
		}
	} else {
		b.Add("\treturn nil, nil\n")
	}
	b.Add("}\n")
	return b.String()
}

func renderGRPCResponseDecoder(endpoint *EndpointData) string {
	var b sourceBuilder
	fmt.Fprintf(&b, "%s\n", codegen.Comment("Decode"+endpoint.Method.VarName+"Response decodes responses from the "+endpoint.ServiceName+" "+endpoint.Method.Name+" endpoint."))
	fmt.Fprintf(&b, "func Decode%sResponse(ctx context.Context, v any, hdr, trlr metadata.MD) (any, error) {\n", endpoint.Method.VarName)
	addGRPCResponseMetadataDecode(&b, endpoint)
	addGRPCViewedResultHeaderDecode(&b, endpoint)
	if endpoint.ClientStream != nil {
		return grpcClientStreamResponseDecoder(&b, endpoint)
	}
	fmt.Fprintf(&b, "\tmessage, ok := v.(%s)\n", endpoint.Response.ClientConvert.SrcRef)
	b.Add("\tif !ok {\n")
	fmt.Fprintf(&b, "\t\treturn nil, loomgrpc.ErrInvalidType(%q, %q, %q, v)\n", endpoint.ServiceName, endpoint.Method.Name, endpoint.Response.ClientConvert.SrcRef)
	b.Add("\t}\n")
	addGRPCClientResponseValidation(&b, endpoint)
	fmt.Fprintf(&b, "\tres := %s(%s)\n", endpoint.Response.ClientConvert.Init.Name, renderInitArgList(endpoint.Response.ClientConvert.Init.Args))
	if endpoint.ViewedResultRef != "" {
		addGRPCViewedResponseReturn(&b, endpoint)
	} else {
		b.Add("\treturn res, nil\n")
	}
	b.Add("}\n")
	return b.String()
}

func renderGRPCRequestDecoder(endpoint *EndpointData) string {
	var b sourceBuilder
	fmt.Fprintf(&b, "%s\n", codegen.Comment(`Decode`+endpoint.Method.VarName+`Request decodes requests sent to "`+endpoint.ServiceName+`" service "`+endpoint.Method.Name+`" endpoint.`))
	fmt.Fprintf(&b, "func Decode%sRequest(ctx context.Context, v any, md metadata.MD) (any, error) {\n", endpoint.Method.VarName)
	addGRPCRequestMetadataDecode(&b, endpoint)
	if endpoint.Request.PayloadMessage != nil || (endpoint.Method.StreamingPayload == "" && !isEmpty(endpoint.Request.Message.Type)) {
		addGRPCRequestMessageDecode(&b, endpoint)
	}
	fmt.Fprintf(&b, "\tvar payload %s\n", endpoint.PayloadRef)
	b.Add("\t{\n")
	addGRPCPayloadInit(&b, endpoint)
	addGRPCMetadataSchemeNormalization(&b, endpoint)
	b.Add("\t}\n")
	b.Add("\treturn payload, nil\n")
	b.Add("}\n")
	return b.String()
}

func addGRPCResponseMetadataDecode(b *sourceBuilder, endpoint *EndpointData) {
	if len(endpoint.Response.Headers) == 0 && len(endpoint.Response.Trailers) == 0 {
		return
	}
	b.Add("\tvar (\n")
	for _, md := range endpoint.Response.Headers {
		fmt.Fprintf(b, "\t\t%s %s\n", md.VarName, md.TypeRef)
	}
	for _, md := range endpoint.Response.Trailers {
		fmt.Fprintf(b, "\t\t%s %s\n", md.VarName, md.TypeRef)
	}
	b.Add("\t\terr error\n")
	b.Add("\t)\n")
	b.Add("\t{\n")
	addGRPCMetadataDecodeBlocks(b, endpoint.Response.Headers, "hdr")
	addGRPCMetadataDecodeBlocks(b, endpoint.Response.Trailers, "trlr")
	b.Add("\t}\n")
	b.Add("\tif err != nil {\n\t\treturn nil, err\n\t}\n")
}

func addGRPCMetadataDecodeBlocks(b *sourceBuilder, metadata []*MetadataData, source string) {
	for _, md := range metadata {
		b.Add("\n")
		b.Add(renderGRPCMetadataDecode(md, source))
		if md.Validate != "" {
			fmt.Fprintf(b, "\t\t%s\n", md.Validate)
		}
	}
}

func addGRPCViewedResultHeaderDecode(b *sourceBuilder, endpoint *EndpointData) {
	if endpoint.ViewedResultRef == "" {
		return
	}
	b.Add("\tvar view string\n")
	b.Add("\t{\n")
	b.Add("\t\tif vals := hdr.Get(\"loom-view\"); len(vals) > 0 {\n")
	b.Add("\t\t\tview = vals[0]\n")
	b.Add("\t\t}\n")
	b.Add("\t}\n")
}

func grpcClientStreamResponseDecoder(b *sourceBuilder, endpoint *EndpointData) string {
	fmt.Fprintf(b, "\treturn &%s{\n", endpoint.ClientStream.VarName)
	fmt.Fprintf(b, "\t\tstream: v.(%s),\n", endpoint.ClientStream.Interface)
	if endpoint.ViewedResultRef != "" {
		b.Add("\t\tview: view,\n")
	}
	b.Add("\t}, nil\n")
	b.Add("}\n")
	return b.String()
}

func addGRPCClientResponseValidation(b *sourceBuilder, endpoint *EndpointData) {
	if endpoint.Response.ClientConvert.Validation == nil || endpoint.ViewedResultRef != "" {
		return
	}
	assign := ":="
	if len(endpoint.Response.Headers) > 0 || len(endpoint.Response.Trailers) > 0 {
		assign = "="
	}
	fmt.Fprintf(b, "\tif err %s %s(message); err != nil {\n\t\treturn nil, err\n\t}\n", assign, endpoint.Response.ClientConvert.Validation.Name)
}

func addGRPCViewedResponseReturn(b *sourceBuilder, endpoint *EndpointData) {
	prefix := ""
	if !endpoint.Method.ViewedResult.IsCollection {
		prefix = "&"
	}
	fmt.Fprintf(b, "\tvres := %s%s{Projected: res, View: view}\n", prefix, endpoint.Method.ViewedResult.FullName)
	fmt.Fprintf(b, "\tif err := %s.Validate%s(vres); err != nil {\n\t\treturn nil, err\n\t}\n", endpoint.Method.ViewedResult.ViewsPkg, endpoint.Method.Result)
	fmt.Fprintf(b, "\tout, err := %s.%s(%s)\n", endpoint.ServicePkgName, endpoint.Method.ViewedResult.ResultInit.Name, renderServiceInitArgList(endpoint.Method.ViewedResult.ResultInit.Args))
	b.Add("\tif err != nil {\n\t\treturn nil, err\n\t}\n")
	b.Add("\treturn out, nil\n")
}

func addGRPCRequestMetadataDecode(b *sourceBuilder, endpoint *EndpointData) {
	if len(endpoint.Request.Metadata) == 0 {
		return
	}
	b.Add("\tvar (\n")
	for _, md := range endpoint.Request.Metadata {
		fmt.Fprintf(b, "\t\t%s %s\n", md.VarName, md.TypeRef)
	}
	b.Add("\t\terr error\n")
	b.Add("\t)\n")
	b.Add("\t{\n")
	for _, md := range endpoint.Request.Metadata {
		b.Add(renderGRPCRequestMetadataDecode(md))
		if md.Validate != "" {
			fmt.Fprintf(b, "\t\t%s\n", md.Validate)
		}
	}
	b.Add("\t}\n")
	b.Add("\tif err != nil {\n\t\treturn nil, err\n\t}\n")
}

func addGRPCRequestMessageDecode(b *sourceBuilder, endpoint *EndpointData) {
	messageRef := endpoint.Request.ServerConvert.SrcRef
	if endpoint.Request.PayloadMessage != nil {
		messageRef = endpoint.Request.PayloadMessage.Ref
	}
	fmt.Fprintf(b, "\tvar (\n\t\tmessage %s\n\t\tok bool\n\t)\n", messageRef)
	b.Add("\t{\n")
	if endpoint.Request.StreamEnvelope != nil {
		envRef := endpoint.Request.Message.Ref
		b.Add("\t\tif v == nil {\n")
		b.Add("\t\t\treturn nil, loom.MissingFieldError(\"initial_payload\", \"stream\")\n")
		b.Add("\t\t}\n")
		fmt.Fprintf(b, "\t\tvar envelope %s\n", envRef)
		fmt.Fprintf(b, "\t\tif envelope, ok = v.(%s); !ok {\n", envRef)
		fmt.Fprintf(b, "\t\t\treturn nil, loomgrpc.ErrInvalidType(%q, %q, %q, v)\n", endpoint.ServiceName, endpoint.Method.Name, envRef)
		b.Add("\t\t}\n")
		fmt.Fprintf(b, "\t\tswitch body := envelope.%s.(type) {\n", endpoint.Request.StreamEnvelope.FieldName)
		fmt.Fprintf(b, "\t\tcase *%s:\n", endpoint.Request.StreamEnvelope.InitialWrapperRef)
		fmt.Fprintf(b, "\t\t\tif body.%s == nil {\n", endpoint.Request.StreamEnvelope.InitialFieldName)
		b.Add("\t\t\t\treturn nil, loom.MissingFieldError(\"initial_payload\", \"stream\")\n")
		b.Add("\t\t\t}\n")
		fmt.Fprintf(b, "\t\t\tmessage = body.%s\n", endpoint.Request.StreamEnvelope.InitialFieldName)
		fmt.Fprintf(b, "\t\tcase *%s:\n", endpoint.Request.StreamEnvelope.StreamItemWrapperRef)
		b.Add("\t\t\treturn nil, loom.InvalidFieldTypeError(\"body\", \"stream_item\", \"initial_payload\")\n")
		b.Add("\t\tdefault:\n")
		b.Add("\t\t\treturn nil, loom.MissingFieldError(\"initial_payload\", \"stream\")\n")
		b.Add("\t\t}\n")
	} else {
		fmt.Fprintf(b, "\t\tif message, ok = v.(%s); !ok {\n", messageRef)
		fmt.Fprintf(b, "\t\t\treturn nil, loomgrpc.ErrInvalidType(%q, %q, %q, v)\n", endpoint.ServiceName, endpoint.Method.Name, endpoint.Request.Message.Ref)
		b.Add("\t\t}\n")
	}
	if endpoint.Request.ServerConvert.Validation != nil {
		assign := ":="
		if len(endpoint.Request.Metadata) > 0 {
			assign = "="
		}
		fmt.Fprintf(b, "\t\tif err %s %s(message); err != nil {\n\t\t\treturn nil, err\n\t\t}\n", assign, endpoint.Request.ServerConvert.Validation.Name)
	}
	b.Add("\t}\n")
}

func addGRPCPayloadInit(b *sourceBuilder, endpoint *EndpointData) {
	if endpoint.Request.ServerConvert != nil {
		fmt.Fprintf(b, "\t\tpayload = %s(%s)\n", endpoint.Request.ServerConvert.Init.Name, renderInitArgList(endpoint.Request.ServerConvert.Init.Args))
		return
	}
	if len(endpoint.Request.Metadata) > 0 {
		fmt.Fprintf(b, "\t\tpayload = %s\n", endpoint.Request.Metadata[0].VarName)
	}
}

func addGRPCMetadataSchemeNormalization(b *sourceBuilder, endpoint *EndpointData) {
	for _, scheme := range endpoint.MetadataSchemes {
		if scheme.Type == "Basic" {
			continue
		}
		if !scheme.CredRequired {
			fmt.Fprintf(b, "\t\tif payload.%s != nil {\n", scheme.CredField)
		}
		fmt.Fprintf(b, "\t\tif strings.Contains(%spayload.%s, \" \") {\n", pointerPrefix(scheme.CredPointer), scheme.CredField)
		b.Add("\t\t\t// Remove authorization scheme prefix (e.g. \"Bearer\")\n")
		fmt.Fprintf(b, "\t\t\tcred := strings.SplitN(%spayload.%s, \" \", 2)[1]\n", pointerPrefix(scheme.CredPointer), scheme.CredField)
		fmt.Fprintf(b, "\t\t\tpayload.%s = %scred\n", scheme.CredField, addrPrefix(scheme.CredPointer))
		b.Add("\t\t}\n")
		if !scheme.CredRequired {
			b.Add("\t\t}\n")
		}
	}
}

func renderGRPCResponseEncoder(endpoint *EndpointData) string {
	var b sourceBuilder
	fmt.Fprintf(&b, "%s\n", codegen.Comment(`Encode`+endpoint.Method.VarName+`Response encodes responses from the "`+endpoint.ServiceName+`" service "`+endpoint.Method.Name+`" endpoint.`))
	fmt.Fprintf(&b, "func Encode%sResponse(ctx context.Context, v any, hdr, trlr *metadata.MD) (any, error) {\n", endpoint.Method.VarName)
	if endpoint.ViewedResultRef != "" {
		fmt.Fprintf(&b, "\tvres, ok := v.(%s)\n", endpoint.ViewedResultRef)
		b.Add("\tif !ok {\n")
		fmt.Fprintf(&b, "\t\treturn nil, loomgrpc.ErrInvalidType(%q, %q, %q, v)\n", endpoint.ServiceName, endpoint.Method.Name, endpoint.ViewedResultRef)
		b.Add("\t}\n")
		b.Add("\tresult := vres.Projected\n")
		b.Add("\t(*hdr).Append(\"loom-view\", vres.View)\n")
	} else if endpoint.ResultRef != "" {
		fmt.Fprintf(&b, "\tresult, ok := v.(%s)\n", endpoint.ResultRef)
		b.Add("\tif !ok {\n")
		fmt.Fprintf(&b, "\t\treturn nil, loomgrpc.ErrInvalidType(%q, %q, %q, v)\n", endpoint.ServiceName, endpoint.Method.Name, endpoint.ResultRef)
		b.Add("\t}\n")
	}
	if endpoint.Response.ServerConvert.Init.ErrorAware {
		fmt.Fprintf(&b, "\tresp, err := %s(%s)\n", endpoint.Response.ServerConvert.Init.Name, renderInitArgList(endpoint.Response.ServerConvert.Init.Args))
		b.Add("\tif err != nil {\n\t\treturn nil, err\n\t}\n")
	} else {
		fmt.Fprintf(&b, "\tresp := %s(%s)\n", endpoint.Response.ServerConvert.Init.Name, renderInitArgList(endpoint.Response.ServerConvert.Init.Args))
	}
	for _, md := range endpoint.Response.Headers {
		b.Add("\n")
		b.Add(renderGRPCMetadataEncode(md, "(*hdr)"))
	}
	for _, md := range endpoint.Response.Trailers {
		b.Add("\n")
		b.Add(renderGRPCMetadataEncode(md, "(*trlr)"))
	}
	b.Add("\treturn resp, nil\n")
	b.Add("}\n")
	return b.String()
}

func renderGRPCStringConversion(dt expr.DataType, varName, target string) string {
	stmt := &jen.Statement{}
	switch dt.Name() {
	case "boolean":
		stmt.Id(varName).Op(":=").Qual("strconv", "FormatBool").Call(exprCode(target))
	case "int":
		stmt.Id(varName).Op(":=").Qual("strconv", "Itoa").Call(exprCode(target))
	case "int32":
		stmt.Id(varName).Op(":=").Qual("strconv", "FormatInt").Call(jen.Int64().Call(exprCode(target)), jen.Lit(10))
	case "int64":
		stmt.Id(varName).Op(":=").Qual("strconv", "FormatInt").Call(exprCode(target), jen.Lit(10))
	case "uint":
		stmt.Id(varName).Op(":=").Qual("strconv", "FormatUint").Call(jen.Uint64().Call(exprCode(target)), jen.Lit(10))
	case "uint32":
		stmt.Id(varName).Op(":=").Qual("strconv", "FormatUint").Call(jen.Uint64().Call(exprCode(target)), jen.Lit(10))
	case "uint64":
		stmt.Id(varName).Op(":=").Qual("strconv", "FormatUint").Call(exprCode(target), jen.Lit(10))
	case "float32":
		stmt.Id(varName).Op(":=").Qual("strconv", "FormatFloat").Call(jen.Float64().Call(exprCode(target)), jen.LitByte('f'), jen.Lit(-1), jen.Lit(32))
	case "float64":
		stmt.Id(varName).Op(":=").Qual("strconv", "FormatFloat").Call(exprCode(target), jen.LitByte('f'), jen.Lit(-1), jen.Lit(64))
	case "string":
		stmt.Id(varName).Op(":=").Add(exprCode(target))
	case "bytes":
		stmt.Id(varName).Op(":=").String().Call(exprCode(target))
	case "any":
		stmt.Id(varName).Op(":=").Qual("fmt", "Sprintf").Call(jen.Lit("%v"), exprCode(target))
	default:
		return renderJenLine(jen.Commentf("unsupported type %s for field %s", dt.Name(), varName))
	}
	return renderJenLine(stmt)
}

func renderGRPCStringParse(md *MetadataData, rawVar string) string {
	name := renderJen(jen.Lit(md.VarName))
	stmt := &jen.Statement{}
	switch md.Type.Name() {
	case "bytes":
		stmt.Add(exprCode(md.VarName)).Op("=").Index().Byte().Call(exprCode(rawVar))
	case "int":
		stmt.Add(renderParseBlock("ParseInt", rawVar, name, "integer", exprCode("strconv.IntSize"), renderParsedAssign(md, "int(v)")))
	case "int32":
		stmt.Add(renderParseBlock("ParseInt", rawVar, name, "integer", jen.Lit(32), renderParsedAssign(md, "int32(v)")))
	case "int64":
		stmt.Add(renderParseBlock("ParseInt", rawVar, name, "integer", jen.Lit(64), renderDirectOrValueAssign(md)))
	case "uint":
		stmt.Add(renderParseBlock("ParseUint", rawVar, name, "unsigned integer", exprCode("strconv.IntSize"), renderParsedAssign(md, "uint(v)")))
	case "uint32":
		stmt.Add(renderParseBlock("ParseUint", rawVar, name, "unsigned integer", jen.Lit(32), renderParsedAssign(md, "uint32(v)")))
	case "uint64":
		stmt.Add(renderParseBlock("ParseUint", rawVar, name, "unsigned integer", jen.Lit(64), renderDirectOrValueAssign(md)))
	case "float32":
		stmt.Add(renderFloatParseBlock(rawVar, name, "float", jen.Lit(32), renderParsedAssign(md, "float32(v)")))
	case "float64":
		stmt.Add(renderFloatParseBlock(rawVar, name, "float", jen.Lit(64), renderDirectOrValueAssign(md)))
	case "boolean":
		stmt.List(jen.Id("v"), jen.Id("err2")).Op(":=").Qual("strconv", "ParseBool").Call(exprCode(rawVar)).Line()
		stmt.If(jen.Id("err2").Op("!=").Nil()).Block(
			exprCode(`err = loom.MergeErrors(err, loom.InvalidFieldTypeError(` + name + `, ` + rawVar + `, "boolean"))`),
		).Line()
		stmt.Add(exprCode(renderDirectOrValueAssign(md)))
	default:
		return renderJenLine(jen.Commentf("unsupported type %s for var %s", md.Type.Name(), md.VarName))
	}
	return renderJenLine(stmt)
}

func renderParsedAssign(md *MetadataData, value string) string {
	stmt := &jen.Statement{}
	if md.Pointer {
		stmt.Id("pv").Op(":=").Add(exprCode(value)).Line()
		stmt.Add(exprCode(md.VarName)).Op("=").Op("&").Id("pv")
		return renderJenLine(stmt)
	}
	stmt.Add(exprCode(md.VarName)).Op("=").Add(exprCode(value))
	return renderJenLine(stmt)
}

func renderDirectOrValueAssign(md *MetadataData) string {
	stmt := &jen.Statement{}
	if md.Pointer {
		stmt.Add(exprCode(md.VarName)).Op("=").Op("&").Id("v")
		return renderJenLine(stmt)
	}
	stmt.Add(exprCode(md.VarName)).Op("=").Id("v")
	return renderJenLine(stmt)
}

func renderGRPCSliceConversion(md *MetadataData, rawVar string) string {
	var b sourceBuilder
	fmt.Fprintf(&b, "%s = make(%s, len(%s))\n", md.VarName, md.TypeRef, rawVar)
	fmt.Fprintf(&b, "for i, rv := range %s {\n", rawVar)
	b.Add(indent(renderGRPCSliceItemConversion(md), 1))
	b.Add("}\n")
	return b.String()
}

func renderGRPCSliceItemConversion(md *MetadataData) string {
	name := renderJen(jen.Lit(md.VarName))
	elemName := expr.AsArray(md.Type).ElemType.Type.Name()
	stmt := &jen.Statement{}
	switch elemName {
	case "string":
		stmt.Add(exprCode(md.VarName)).Index(jen.Id("i")).Op("=").Id("rv")
	case "bytes":
		stmt.Add(exprCode(md.VarName)).Index(jen.Id("i")).Op("=").Index().Byte().Call(jen.Id("rv"))
	case "int":
		stmt.Add(renderSliceParseBlock("ParseInt", name, md.VarName, "array of integers", exprCode("strconv.IntSize"), "int(v)"))
	case "int32":
		stmt.Add(renderSliceParseBlock("ParseInt", name, md.VarName, "array of integers", jen.Lit(32), "int32(v)"))
	case "int64":
		stmt.Add(renderSliceParseBlock("ParseInt", name, md.VarName, "array of integers", jen.Lit(64), "v"))
	case "uint":
		stmt.Add(renderSliceParseBlock("ParseUint", name, md.VarName, "array of unsigned integers", exprCode("strconv.IntSize"), "uint(v)"))
	case "uint32":
		stmt.Add(renderSliceParseBlock("ParseUint", name, md.VarName, "array of unsigned integers", jen.Lit(32), "uint32(v)"))
	case "uint64":
		stmt.Add(renderSliceParseBlock("ParseUint", name, md.VarName, "array of unsigned integers", jen.Lit(64), "v"))
	case "float32":
		stmt.Add(renderSliceFloatParseBlock(name, md.VarName, "array of floats", jen.Lit(32), "float32(v)"))
	case "float64":
		stmt.Add(renderSliceFloatParseBlock(name, md.VarName, "array of floats", jen.Lit(64), "v"))
	case "boolean":
		stmt.List(jen.Id("v"), jen.Id("err2")).Op(":=").Qual("strconv", "ParseBool").Call(jen.Id("rv")).Line()
		stmt.If(jen.Id("err2").Op("!=").Nil()).Block(
			exprCode(`err = loom.MergeErrors(err, loom.InvalidFieldTypeError(` + name + `, rv, "array of booleans"))`),
		).Line()
		stmt.Add(exprCode(md.VarName)).Index(jen.Id("i")).Op("=").Id("v")
	case "any":
		stmt.Add(exprCode(md.VarName)).Index(jen.Id("i")).Op("=").Id("rv")
	default:
		return renderJenLine(jen.Commentf("unsupported slice type %s for var %s", elemName, md.VarName))
	}
	return renderJenLine(stmt)
}

func renderParseBlock(fn, rawVar, name, kind string, bits *jen.Statement, assign string) *jen.Statement {
	stmt := &jen.Statement{}
	stmt.List(jen.Id("v"), jen.Id("err2")).Op(":=").Qual("strconv", fn).Call(exprCode(rawVar), jen.Lit(10), bits).Line()
	stmt.If(jen.Id("err2").Op("!=").Nil()).Block(
		exprCode(`err = loom.MergeErrors(err, loom.InvalidFieldTypeError(` + name + `, ` + rawVar + `, "` + kind + `"))`),
	).Line()
	stmt.Add(exprCode(assign))
	return stmt
}

func renderFloatParseBlock(rawVar, name, kind string, bits *jen.Statement, assign string) *jen.Statement {
	stmt := &jen.Statement{}
	stmt.List(jen.Id("v"), jen.Id("err2")).Op(":=").Qual("strconv", "ParseFloat").Call(exprCode(rawVar), bits).Line()
	stmt.If(jen.Id("err2").Op("!=").Nil()).Block(
		exprCode(`err = loom.MergeErrors(err, loom.InvalidFieldTypeError(` + name + `, ` + rawVar + `, "` + kind + `"))`),
	).Line()
	stmt.Add(exprCode(assign))
	return stmt
}

func renderSliceParseBlock(fn, name, varName, kind string, bits *jen.Statement, assign string) *jen.Statement {
	stmt := &jen.Statement{}
	stmt.List(jen.Id("v"), jen.Id("err2")).Op(":=").Qual("strconv", fn).Call(jen.Id("rv"), jen.Lit(10), bits).Line()
	stmt.If(jen.Id("err2").Op("!=").Nil()).Block(
		exprCode(`err = loom.MergeErrors(err, loom.InvalidFieldTypeError(` + name + `, rv, "` + kind + `"))`),
	).Line()
	stmt.Add(exprCode(varName)).Index(jen.Id("i")).Op("=").Add(exprCode(assign))
	return stmt
}

func renderSliceFloatParseBlock(name, varName, kind string, bits *jen.Statement, assign string) *jen.Statement {
	stmt := &jen.Statement{}
	stmt.List(jen.Id("v"), jen.Id("err2")).Op(":=").Qual("strconv", "ParseFloat").Call(jen.Id("rv"), bits).Line()
	stmt.If(jen.Id("err2").Op("!=").Nil()).Block(
		exprCode(`err = loom.MergeErrors(err, loom.InvalidFieldTypeError(` + name + `, rv, "` + kind + `"))`),
	).Line()
	stmt.Add(exprCode(varName)).Index(jen.Id("i")).Op("=").Add(exprCode(assign))
	return stmt
}

func renderInitArgList(args []*InitArgData) string {
	names := make([]string, len(args))
	for i, arg := range args {
		names[i] = arg.Name
	}
	return strings.Join(names, ", ")
}

func renderServiceInitArgList(args []*service.InitArgData) string {
	names := make([]string, len(args))
	for i, arg := range args {
		names[i] = arg.Name
	}
	return strings.Join(names, ", ")
}

func fieldSelector(root, field string) string {
	if field == "" {
		return root
	}
	return root + "." + field
}

func pointerPrefix(pointer bool) string {
	if pointer {
		return "*"
	}
	return ""
}

func addrPrefix(pointer bool) string {
	if pointer {
		return "&"
	}
	return ""
}

func indent(code string, levels int) string {
	return codegen.Indent(code, strings.Repeat("\t", levels))
}
