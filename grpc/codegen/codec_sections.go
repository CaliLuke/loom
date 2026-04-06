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
		fmt.Fprintf(&b, "\treturn %s(%s), nil\n", endpoint.Request.ClientConvert.Init.Name, renderInitArgList(endpoint.Request.ClientConvert.Init.Args))
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
	if len(endpoint.Response.Headers) > 0 || len(endpoint.Response.Trailers) > 0 {
		b.Add("\tvar (\n")
		for _, md := range endpoint.Response.Headers {
			fmt.Fprintf(&b, "\t\t%s %s\n", md.VarName, md.TypeRef)
		}
		for _, md := range endpoint.Response.Trailers {
			fmt.Fprintf(&b, "\t\t%s %s\n", md.VarName, md.TypeRef)
		}
		b.Add("\t\terr error\n")
		b.Add("\t)\n")
		b.Add("\t{\n")
		for _, md := range endpoint.Response.Headers {
			b.Add("\n")
			b.Add(renderGRPCMetadataDecode(md, "hdr"))
			if md.Validate != "" {
				fmt.Fprintf(&b, "\t\t%s\n", md.Validate)
			}
		}
		for _, md := range endpoint.Response.Trailers {
			b.Add("\n")
			b.Add(renderGRPCMetadataDecode(md, "trlr"))
			if md.Validate != "" {
				fmt.Fprintf(&b, "\t\t%s\n", md.Validate)
			}
		}
		b.Add("\t}\n")
		b.Add("\tif err != nil {\n\t\treturn nil, err\n\t}\n")
	}
	if endpoint.ViewedResultRef != "" {
		b.Add("\tvar view string\n")
		b.Add("\t{\n")
		b.Add("\t\tif vals := hdr.Get(\"loom-view\"); len(vals) > 0 {\n")
		b.Add("\t\t\tview = vals[0]\n")
		b.Add("\t\t}\n")
		b.Add("\t}\n")
	}
	if endpoint.ClientStream != nil {
		fmt.Fprintf(&b, "\treturn &%s{\n", endpoint.ClientStream.VarName)
		fmt.Fprintf(&b, "\t\tstream: v.(%s),\n", endpoint.ClientStream.Interface)
		if endpoint.ViewedResultRef != "" {
			b.Add("\t\tview: view,\n")
		}
		b.Add("\t}, nil\n")
		b.Add("}\n")
		return b.String()
	}
	fmt.Fprintf(&b, "\tmessage, ok := v.(%s)\n", endpoint.Response.ClientConvert.SrcRef)
	b.Add("\tif !ok {\n")
	fmt.Fprintf(&b, "\t\treturn nil, loomgrpc.ErrInvalidType(%q, %q, %q, v)\n", endpoint.ServiceName, endpoint.Method.Name, endpoint.Response.ClientConvert.SrcRef)
	b.Add("\t}\n")
	if endpoint.Response.ClientConvert.Validation != nil && endpoint.ViewedResultRef == "" {
		assign := ":="
		if len(endpoint.Response.Headers) > 0 || len(endpoint.Response.Trailers) > 0 {
			assign = "="
		}
		fmt.Fprintf(&b, "\tif err %s %s(message); err != nil {\n\t\treturn nil, err\n\t}\n", assign, endpoint.Response.ClientConvert.Validation.Name)
	}
	fmt.Fprintf(&b, "\tres := %s(%s)\n", endpoint.Response.ClientConvert.Init.Name, renderInitArgList(endpoint.Response.ClientConvert.Init.Args))
	if endpoint.ViewedResultRef != "" {
		prefix := ""
		if !endpoint.Method.ViewedResult.IsCollection {
			prefix = "&"
		}
		fmt.Fprintf(&b, "\tvres := %s%s{Projected: res, View: view}\n", prefix, endpoint.Method.ViewedResult.FullName)
		fmt.Fprintf(&b, "\tif err := %s.Validate%s(vres); err != nil {\n\t\treturn nil, err\n\t}\n", endpoint.Method.ViewedResult.ViewsPkg, endpoint.Method.Result)
		fmt.Fprintf(&b, "\treturn %s.%s(%s), nil\n", endpoint.ServicePkgName, endpoint.Method.ViewedResult.ResultInit.Name, renderServiceInitArgList(endpoint.Method.ViewedResult.ResultInit.Args))
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
	if len(endpoint.Request.Metadata) > 0 {
		b.Add("\tvar (\n")
		for _, md := range endpoint.Request.Metadata {
			fmt.Fprintf(&b, "\t\t%s %s\n", md.VarName, md.TypeRef)
		}
		b.Add("\t\terr error\n")
		b.Add("\t)\n")
		b.Add("\t{\n")
		for _, md := range endpoint.Request.Metadata {
			b.Add(renderGRPCRequestMetadataDecode(md))
			if md.Validate != "" {
				fmt.Fprintf(&b, "\t\t%s\n", md.Validate)
			}
		}
		b.Add("\t}\n")
		b.Add("\tif err != nil {\n\t\treturn nil, err\n\t}\n")
	}
	if endpoint.Method.StreamingPayload == "" && !isEmpty(endpoint.Request.Message.Type) {
		fmt.Fprintf(&b, "\tvar (\n\t\tmessage %s\n\t\tok bool\n\t)\n", endpoint.Request.ServerConvert.SrcRef)
		b.Add("\t{\n")
		fmt.Fprintf(&b, "\t\tif message, ok = v.(%s); !ok {\n", endpoint.Request.ServerConvert.SrcRef)
		fmt.Fprintf(&b, "\t\t\treturn nil, loomgrpc.ErrInvalidType(%q, %q, %q, v)\n", endpoint.ServiceName, endpoint.Method.Name, endpoint.Request.Message.Ref)
		b.Add("\t\t}\n")
		if endpoint.Request.ServerConvert.Validation != nil {
			assign := ":="
			if len(endpoint.Request.Metadata) > 0 {
				assign = "="
			}
			fmt.Fprintf(&b, "\t\tif err %s %s(message); err != nil {\n\t\t\treturn nil, err\n\t\t}\n", assign, endpoint.Request.ServerConvert.Validation.Name)
		}
		b.Add("\t}\n")
	}
	fmt.Fprintf(&b, "\tvar payload %s\n", endpoint.PayloadRef)
	b.Add("\t{\n")
	if endpoint.Request.ServerConvert != nil {
		fmt.Fprintf(&b, "\t\tpayload = %s(%s)\n", endpoint.Request.ServerConvert.Init.Name, renderInitArgList(endpoint.Request.ServerConvert.Init.Args))
	} else if len(endpoint.Request.Metadata) > 0 {
		fmt.Fprintf(&b, "\t\tpayload = %s\n", endpoint.Request.Metadata[0].VarName)
	}
	for _, scheme := range endpoint.MetadataSchemes {
		if scheme.Type == "Basic" {
			continue
		}
		if !scheme.CredRequired {
			fmt.Fprintf(&b, "\t\tif payload.%s != nil {\n", scheme.CredField)
		}
		fmt.Fprintf(&b, "\t\tif strings.Contains(%spayload.%s, \" \") {\n", pointerPrefix(scheme.CredPointer), scheme.CredField)
		b.Add("\t\t\t// Remove authorization scheme prefix (e.g. \"Bearer\")\n")
		fmt.Fprintf(&b, "\t\t\tcred := strings.SplitN(%spayload.%s, \" \", 2)[1]\n", pointerPrefix(scheme.CredPointer), scheme.CredField)
		fmt.Fprintf(&b, "\t\t\tpayload.%s = %scred\n", scheme.CredField, addrPrefix(scheme.CredPointer))
		b.Add("\t\t}\n")
		if !scheme.CredRequired {
			b.Add("\t\t}\n")
		}
	}
	b.Add("\t}\n")
	b.Add("\treturn payload, nil\n")
	b.Add("}\n")
	return b.String()
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
	fmt.Fprintf(&b, "\tresp := %s(%s)\n", endpoint.Response.ServerConvert.Init.Name, renderInitArgList(endpoint.Response.ServerConvert.Init.Args))
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

func renderGRPCMetadataAppend(md *MetadataData, root string, schemes []*service.SchemeData) string {
	var b sourceBuilder
	value := fieldSelector(root, md.FieldName)
	switch {
	case md.StringSlice:
		fmt.Fprintf(&b, "\tfor _, value := range %s {\n", value)
		fmt.Fprintf(&b, "\t\t(*md).Append(%q, value)\n", md.Name)
		b.Add("\t}\n")
	case md.Slice:
		fmt.Fprintf(&b, "\tfor _, value := range %s {\n", value)
		b.Add(indent(renderGRPCStringConversion(expr.AsArray(md.Type).ElemType.Type, "valueStr", "value"), 2))
		fmt.Fprintf(&b, "\t\t(*md).Append(%q, valueStr)\n", md.Name)
		b.Add("\t}\n")
	default:
		if md.Pointer {
			fmt.Fprintf(&b, "\tif %s != nil {\n", value)
		}
		if md.Name == "Authorization" && isBearer(schemes) {
			fmt.Fprintf(&b, "\t\tif !strings.Contains(%s%s, \" \") {\n", pointerPrefix(md.Pointer), value)
			fmt.Fprintf(&b, "\t\t\t(*md).Append(%q, \"Bearer \"+%s%s)\n", md.Name, pointerPrefix(md.Pointer), value)
			b.Add("\t\t} else {\n")
			fmt.Fprintf(&b, "\t\t\t(*md).Append(%q, %s)\n", md.Name, renderMetadataSingleValue(md, value))
			b.Add("\t\t}\n")
		} else {
			fmt.Fprintf(&b, "\t\t(*md).Append(%q, %s)\n", md.Name, renderMetadataSingleValue(md, value))
		}
		if md.Pointer {
			b.Add("\t}\n")
		}
	}
	return b.String()
}

func renderGRPCMetadataDecode(md *MetadataData, mdVar string) string {
	var b sourceBuilder
	name := renderJen(jen.Lit(md.Name))
	switch {
	case md.TypeName == "string" || md.Type.Name() == "any":
		if md.Required {
			fmt.Fprintf(&b, "\t\tif vals := %s.Get(%s); len(vals) == 0 {\n", mdVar, name)
			fmt.Fprintf(&b, "\t\t\terr = loom.MergeErrors(err, loom.MissingFieldError(%s, \"metadata\"))\n", name)
			b.Add("\t\t} else {\n")
			fmt.Fprintf(&b, "\t\t\t%s = vals[0]\n", md.VarName)
			b.Add("\t\t}\n")
		} else {
			fmt.Fprintf(&b, "\t\tif vals := %s.Get(%s); len(vals) > 0 {\n", mdVar, name)
			fmt.Fprintf(&b, "\t\t\t%s = vals[0]\n", md.VarName)
			b.Add("\t\t}\n")
		}
	case md.StringSlice:
		if md.Required {
			fmt.Fprintf(&b, "\t\tif vals := %s.Get(%s); len(vals) == 0 {\n", mdVar, name)
			fmt.Fprintf(&b, "\t\t\terr = loom.MergeErrors(err, loom.MissingFieldError(%s, \"metadata\"))\n", name)
			b.Add("\t\t} else {\n")
			fmt.Fprintf(&b, "\t\t\t%s = vals\n", md.VarName)
			b.Add("\t\t}\n")
		} else {
			fmt.Fprintf(&b, "\t\t%s = %s.Get(%s)\n", md.VarName, mdVar, name)
		}
	case md.Slice:
		rawVar := md.VarName + "Raw"
		if md.Required {
			fmt.Fprintf(&b, "\t\tif %s := %s.Get(%s); len(%s) == 0 {\n", rawVar, mdVar, name, rawVar)
			fmt.Fprintf(&b, "\t\t\terr = loom.MergeErrors(err, loom.MissingFieldError(%s, \"metadata\"))\n", name)
			b.Add("\t\t} else {\n")
			b.Add(indent(renderGRPCSliceConversion(md, rawVar), 3))
			b.Add("\t\t}\n")
		} else {
			fmt.Fprintf(&b, "\t\tif %s := %s.Get(%s); len(%s) > 0 {\n", rawVar, mdVar, name, rawVar)
			b.Add(indent(renderGRPCSliceConversion(md, rawVar), 3))
			b.Add("\t\t}\n")
		}
	default:
		rawVar := md.VarName + "Raw"
		if md.Required {
			fmt.Fprintf(&b, "\t\tif vals := %s.Get(%s); len(vals) == 0 {\n", mdVar, name)
			fmt.Fprintf(&b, "\t\t\terr = loom.MergeErrors(err, loom.MissingFieldError(%s, \"metadata\"))\n", name)
			b.Add("\t\t} else {\n")
			fmt.Fprintf(&b, "\t\t\t%s = vals[0]\n", rawVar)
			b.Add("\n")
			b.Add(indent(renderGRPCStringParse(md, rawVar), 3))
			b.Add("\t\t}\n")
		} else {
			fmt.Fprintf(&b, "\t\tif vals := %s.Get(%s); len(vals) > 0 {\n", mdVar, name)
			fmt.Fprintf(&b, "\t\t\t%s = vals[0]\n", rawVar)
			b.Add("\n")
			b.Add(indent(renderGRPCStringParse(md, rawVar), 3))
			b.Add("\t\t}\n")
		}
	}
	return b.String()
}

func renderGRPCRequestMetadataDecode(md *MetadataData) string {
	var b sourceBuilder
	name := renderJen(jen.Lit(md.Name))
	switch {
	case md.TypeName == "string" || md.Type.Name() == "any":
		if md.Required {
			fmt.Fprintf(&b, "\t\tif vals := md.Get(%s); len(vals) == 0 {\n", name)
			fmt.Fprintf(&b, "\t\t\terr = loom.MergeErrors(err, loom.MissingFieldError(%s, \"metadata\"))\n", name)
			b.Add("\t\t} else {\n")
			fmt.Fprintf(&b, "\t\t\t%s = vals[0]\n", md.VarName)
			b.Add("\t\t}\n")
		} else {
			fmt.Fprintf(&b, "\t\tif vals := md.Get(%s); len(vals) > 0 {\n", name)
			if md.Pointer {
				fmt.Fprintf(&b, "\t\t\t%s = &vals[0]\n", md.VarName)
			} else {
				fmt.Fprintf(&b, "\t\t\t%s = vals[0]\n", md.VarName)
			}
			b.Add("\t\t}\n")
		}
	case md.StringSlice:
		if md.Required {
			fmt.Fprintf(&b, "\t\tif vals := md.Get(%s); len(vals) == 0 {\n", name)
			fmt.Fprintf(&b, "\t\t\terr = loom.MergeErrors(err, loom.MissingFieldError(%s, \"metadata\"))\n", name)
			b.Add("\t\t} else {\n")
			fmt.Fprintf(&b, "\t\t\t%s = vals\n", md.VarName)
			b.Add("\t\t}\n")
		} else {
			fmt.Fprintf(&b, "\t\t%s = md.Get(%s)\n", md.VarName, name)
		}
	case md.Slice:
		rawVar := md.VarName + "Raw"
		if md.Required {
			fmt.Fprintf(&b, "\t\tif %s := md.Get(%s); len(%s) == 0 {\n", rawVar, name, rawVar)
			fmt.Fprintf(&b, "\t\t\terr = loom.MergeErrors(err, loom.MissingFieldError(%s, \"metadata\"))\n", name)
			b.Add("\t\t} else {\n")
			b.Add(indent(renderGRPCSliceConversion(md, rawVar), 3))
			b.Add("\t\t}\n")
		} else {
			fmt.Fprintf(&b, "\t\tif %s := md.Get(%s); len(%s) > 0 {\n", rawVar, name, rawVar)
			b.Add(indent(renderGRPCSliceConversion(md, rawVar), 3))
			b.Add("\t\t}\n")
		}
	default:
		rawVar := md.VarName + "Raw"
		if md.Required {
			fmt.Fprintf(&b, "\t\tif vals := md.Get(%s); len(vals) == 0 {\n", name)
			fmt.Fprintf(&b, "\t\t\terr = loom.MergeErrors(err, loom.MissingFieldError(%s, \"metadata\"))\n", name)
			b.Add("\t\t} else {\n")
			fmt.Fprintf(&b, "\t\t\t%s := vals[0]\n", rawVar)
			b.Add("\n")
			b.Add(indent(renderGRPCStringParse(md, rawVar), 3))
			b.Add("\t\t}\n")
		} else {
			fmt.Fprintf(&b, "\t\tif vals := md.Get(%s); len(vals) > 0 {\n", name)
			fmt.Fprintf(&b, "\t\t\t%s := vals[0]\n", rawVar)
			b.Add("\n")
			b.Add(indent(renderGRPCStringParse(md, rawVar), 3))
			b.Add("\t\t}\n")
		}
	}
	return b.String()
}

func renderGRPCMetadataEncode(md *MetadataData, targetVar string) string {
	var b sourceBuilder
	switch {
	case md.StringSlice:
		fmt.Fprintf(&b, "\t%s.Append(%q, res.%s...)\n", targetVar, md.Name, md.FieldName)
	case md.Slice:
		fmt.Fprintf(&b, "\tfor _, value := range res.%s {\n", md.FieldName)
		b.Add(indent(renderGRPCStringConversion(expr.AsArray(md.Type).ElemType.Type, "valueStr", "value"), 2))
		fmt.Fprintf(&b, "\t\t%s.Append(%q, valueStr)\n", targetVar, md.Name)
		b.Add("\t}\n")
	default:
		if md.Pointer {
			fmt.Fprintf(&b, "\tif res.%s != nil {\n", md.FieldName)
		}
		fmt.Fprintf(&b, "\t\t%s.Append(%q, %s)\n", targetVar, md.Name, renderTemplateMetadataValue(md))
		if md.Pointer {
			b.Add("\t}\n")
		}
	}
	return b.String()
}

func renderMetadataSingleValue(md *MetadataData, value string) string {
	switch md.Type.Name() {
	case "bytes":
		return "string(" + pointerPrefix(md.Pointer) + value + ")"
	case "string":
		return pointerPrefix(md.Pointer) + value
	default:
		return renderJen(jen.Qual("fmt", "Sprintf").Call(jen.Lit("%v"), exprCode(pointerPrefix(md.Pointer)+value)))
	}
}

func renderTemplateMetadataValue(md *MetadataData) string {
	switch md.Type.Name() {
	case "bytes":
		return "string(*p." + md.FieldName + ")"
	case "string":
		if md.Pointer {
			return "*p." + md.FieldName
		}
		return "p." + md.FieldName
	default:
		return renderJen(jen.Qual("fmt", "Sprintf").Call(jen.Lit("%v"), exprCode("*p."+md.FieldName)))
	}
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
		stmt.Add(renderSliceParseBlock("ParseUint", name, md.VarName, "array of unsigned integers", jen.Lit(32), "int32(v)"))
	case "uint64":
		stmt.Add(renderSliceParseBlock("ParseUint", name, md.VarName, "array of unsigned integers", jen.Lit(64), "v"))
	case "float32":
		stmt.Add(renderSliceFloatParseBlock(name, md.VarName, "array of floats", jen.Lit(32), "float32(v)"))
	case "float64":
		stmt.Add(renderSliceFloatParseBlock(name, md.VarName, "array of floats", jen.Lit(64), "v"))
	case "boolean":
		stmt.List(jen.Id("v"), jen.Id("err2")).Op(":=").Qual("strconv", "ParseBool").Call(jen.Id("rv")).Line()
		stmt.If(jen.Id("err2").Op("!=").Nil()).Block(
			exprCode(`err = loom.MergeErrors(err, loom.InvalidFieldTypeError(` + name + `, ` + md.VarName + `Raw, "array of booleans"))`),
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
		exprCode(`err = loom.MergeErrors(err, loom.InvalidFieldTypeError(` + name + `, ` + varName + `Raw, "` + kind + `"))`),
	).Line()
	stmt.Add(exprCode(varName)).Index(jen.Id("i")).Op("=").Add(exprCode(assign))
	return stmt
}

func renderSliceFloatParseBlock(name, varName, kind string, bits *jen.Statement, assign string) *jen.Statement {
	stmt := &jen.Statement{}
	stmt.List(jen.Id("v"), jen.Id("err2")).Op(":=").Qual("strconv", "ParseFloat").Call(jen.Id("rv"), bits).Line()
	stmt.If(jen.Id("err2").Op("!=").Nil()).Block(
		exprCode(`err = loom.MergeErrors(err, loom.InvalidFieldTypeError(` + name + `, ` + varName + `Raw, "` + kind + `"))`),
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
