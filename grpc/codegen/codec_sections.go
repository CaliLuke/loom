package codegen

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/v3/codegen"
	"github.com/CaliLuke/loom/v3/codegen/service"
	"github.com/CaliLuke/loom/v3/expr"
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
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", codegen.Comment(fmt.Sprintf("Encode%sRequest encodes requests sent to %s %s endpoint.", endpoint.Method.VarName, endpoint.ServiceName, endpoint.Method.Name)))
	fmt.Fprintf(&b, "func Encode%sRequest(ctx context.Context, v any, md *metadata.MD) (any, error) {\n", endpoint.Method.VarName)
	fmt.Fprintf(&b, "\tpayload, ok := v.(%s)\n", endpoint.PayloadRef)
	b.WriteString("\tif !ok {\n")
	fmt.Fprintf(&b, "\t\treturn nil, goagrpc.ErrInvalidType(%q, %q, %q, v)\n", endpoint.ServiceName, endpoint.Method.Name, endpoint.PayloadRef)
	b.WriteString("\t}\n")
	for _, md := range endpoint.Request.Metadata {
		b.WriteString(renderGRPCMetadataAppend(md, "payload", endpoint.MetadataSchemes))
	}
	if endpoint.Request.ClientConvert != nil {
		fmt.Fprintf(&b, "\treturn %s(%s), nil\n", endpoint.Request.ClientConvert.Init.Name, renderInitArgList(endpoint.Request.ClientConvert.Init.Args))
	} else {
		b.WriteString("\treturn nil, nil\n")
	}
	b.WriteString("}\n")
	return b.String()
}

func renderGRPCResponseDecoder(endpoint *EndpointData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", codegen.Comment(fmt.Sprintf("Decode%sResponse decodes responses from the %s %s endpoint.", endpoint.Method.VarName, endpoint.ServiceName, endpoint.Method.Name)))
	fmt.Fprintf(&b, "func Decode%sResponse(ctx context.Context, v any, hdr, trlr metadata.MD) (any, error) {\n", endpoint.Method.VarName)
	if len(endpoint.Response.Headers) > 0 || len(endpoint.Response.Trailers) > 0 {
		b.WriteString("\tvar (\n")
		for _, md := range endpoint.Response.Headers {
			fmt.Fprintf(&b, "\t\t%s %s\n", md.VarName, md.TypeRef)
		}
		for _, md := range endpoint.Response.Trailers {
			fmt.Fprintf(&b, "\t\t%s %s\n", md.VarName, md.TypeRef)
		}
		b.WriteString("\t\terr error\n")
		b.WriteString("\t)\n")
		b.WriteString("\t{\n")
		for _, md := range endpoint.Response.Headers {
			b.WriteString("\n")
			b.WriteString(renderGRPCMetadataDecode(md, "hdr"))
			if md.Validate != "" {
				fmt.Fprintf(&b, "\t\t%s\n", md.Validate)
			}
		}
		for _, md := range endpoint.Response.Trailers {
			b.WriteString("\n")
			b.WriteString(renderGRPCMetadataDecode(md, "trlr"))
			if md.Validate != "" {
				fmt.Fprintf(&b, "\t\t%s\n", md.Validate)
			}
		}
		b.WriteString("\t}\n")
		b.WriteString("\tif err != nil {\n\t\treturn nil, err\n\t}\n")
	}
	if endpoint.ViewedResultRef != "" {
		b.WriteString("\tvar view string\n")
		b.WriteString("\t{\n")
		b.WriteString("\t\tif vals := hdr.Get(\"loom-view\"); len(vals) > 0 {\n")
		b.WriteString("\t\t\tview = vals[0]\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t}\n")
	}
	if endpoint.ClientStream != nil {
		fmt.Fprintf(&b, "\treturn &%s{\n", endpoint.ClientStream.VarName)
		fmt.Fprintf(&b, "\t\tstream: v.(%s),\n", endpoint.ClientStream.Interface)
		if endpoint.ViewedResultRef != "" {
			b.WriteString("\t\tview: view,\n")
		}
		b.WriteString("\t}, nil\n")
		b.WriteString("}\n")
		return b.String()
	}
	fmt.Fprintf(&b, "\tmessage, ok := v.(%s)\n", endpoint.Response.ClientConvert.SrcRef)
	b.WriteString("\tif !ok {\n")
	fmt.Fprintf(&b, "\t\treturn nil, goagrpc.ErrInvalidType(%q, %q, %q, v)\n", endpoint.ServiceName, endpoint.Method.Name, endpoint.Response.ClientConvert.SrcRef)
	b.WriteString("\t}\n")
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
		b.WriteString("\treturn res, nil\n")
	}
	b.WriteString("}\n")
	return b.String()
}

func renderGRPCRequestDecoder(endpoint *EndpointData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", codegen.Comment(fmt.Sprintf("Decode%sRequest decodes requests sent to %q service %q endpoint.", endpoint.Method.VarName, endpoint.ServiceName, endpoint.Method.Name)))
	fmt.Fprintf(&b, "func Decode%sRequest(ctx context.Context, v any, md metadata.MD) (any, error) {\n", endpoint.Method.VarName)
	if len(endpoint.Request.Metadata) > 0 {
		b.WriteString("\tvar (\n")
		for _, md := range endpoint.Request.Metadata {
			fmt.Fprintf(&b, "\t\t%s %s\n", md.VarName, md.TypeRef)
		}
		b.WriteString("\t\terr error\n")
		b.WriteString("\t)\n")
		b.WriteString("\t{\n")
		for _, md := range endpoint.Request.Metadata {
			b.WriteString(renderGRPCRequestMetadataDecode(md))
			if md.Validate != "" {
				fmt.Fprintf(&b, "\t\t%s\n", md.Validate)
			}
		}
		b.WriteString("\t}\n")
		b.WriteString("\tif err != nil {\n\t\treturn nil, err\n\t}\n")
	}
	if endpoint.Method.StreamingPayload == "" && !isEmpty(endpoint.Request.Message.Type) {
		fmt.Fprintf(&b, "\tvar (\n\t\tmessage %s\n\t\tok bool\n\t)\n", endpoint.Request.ServerConvert.SrcRef)
		b.WriteString("\t{\n")
		fmt.Fprintf(&b, "\t\tif message, ok = v.(%s); !ok {\n", endpoint.Request.ServerConvert.SrcRef)
		fmt.Fprintf(&b, "\t\t\treturn nil, goagrpc.ErrInvalidType(%q, %q, %q, v)\n", endpoint.ServiceName, endpoint.Method.Name, endpoint.Request.Message.Ref)
		b.WriteString("\t\t}\n")
		if endpoint.Request.ServerConvert.Validation != nil {
			assign := ":="
			if len(endpoint.Request.Metadata) > 0 {
				assign = "="
			}
			fmt.Fprintf(&b, "\t\tif err %s %s(message); err != nil {\n\t\t\treturn nil, err\n\t\t}\n", assign, endpoint.Request.ServerConvert.Validation.Name)
		}
		b.WriteString("\t}\n")
	}
	fmt.Fprintf(&b, "\tvar payload %s\n", endpoint.PayloadRef)
	b.WriteString("\t{\n")
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
		b.WriteString("\t\t\t// Remove authorization scheme prefix (e.g. \"Bearer\")\n")
		fmt.Fprintf(&b, "\t\t\tcred := strings.SplitN(%spayload.%s, \" \", 2)[1]\n", pointerPrefix(scheme.CredPointer), scheme.CredField)
		fmt.Fprintf(&b, "\t\t\tpayload.%s = %scred\n", scheme.CredField, addrPrefix(scheme.CredPointer))
		b.WriteString("\t\t}\n")
		if !scheme.CredRequired {
			b.WriteString("\t\t}\n")
		}
	}
	b.WriteString("\t}\n")
	b.WriteString("\treturn payload, nil\n")
	b.WriteString("}\n")
	return b.String()
}

func renderGRPCResponseEncoder(endpoint *EndpointData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", codegen.Comment(fmt.Sprintf("Encode%sResponse encodes responses from the %q service %q endpoint.", endpoint.Method.VarName, endpoint.ServiceName, endpoint.Method.Name)))
	fmt.Fprintf(&b, "func Encode%sResponse(ctx context.Context, v any, hdr, trlr *metadata.MD) (any, error) {\n", endpoint.Method.VarName)
	if endpoint.ViewedResultRef != "" {
		fmt.Fprintf(&b, "\tvres, ok := v.(%s)\n", endpoint.ViewedResultRef)
		b.WriteString("\tif !ok {\n")
		fmt.Fprintf(&b, "\t\treturn nil, goagrpc.ErrInvalidType(%q, %q, %q, v)\n", endpoint.ServiceName, endpoint.Method.Name, endpoint.ViewedResultRef)
		b.WriteString("\t}\n")
		b.WriteString("\tresult := vres.Projected\n")
		b.WriteString("\t(*hdr).Append(\"loom-view\", vres.View)\n")
	} else if endpoint.ResultRef != "" {
		fmt.Fprintf(&b, "\tresult, ok := v.(%s)\n", endpoint.ResultRef)
		b.WriteString("\tif !ok {\n")
		fmt.Fprintf(&b, "\t\treturn nil, goagrpc.ErrInvalidType(%q, %q, %q, v)\n", endpoint.ServiceName, endpoint.Method.Name, endpoint.ResultRef)
		b.WriteString("\t}\n")
	}
	fmt.Fprintf(&b, "\tresp := %s(%s)\n", endpoint.Response.ServerConvert.Init.Name, renderInitArgList(endpoint.Response.ServerConvert.Init.Args))
	for _, md := range endpoint.Response.Headers {
		b.WriteString("\n")
		b.WriteString(renderGRPCMetadataEncode(md, "(*hdr)"))
	}
	for _, md := range endpoint.Response.Trailers {
		b.WriteString("\n")
		b.WriteString(renderGRPCMetadataEncode(md, "(*trlr)"))
	}
	b.WriteString("\treturn resp, nil\n")
	b.WriteString("}\n")
	return b.String()
}

func renderGRPCMetadataAppend(md *MetadataData, root string, schemes []*service.SchemeData) string {
	var b strings.Builder
	value := fieldSelector(root, md.FieldName)
	switch {
	case md.StringSlice:
		fmt.Fprintf(&b, "\tfor _, value := range %s {\n", value)
		fmt.Fprintf(&b, "\t\t(*md).Append(%q, value)\n", md.Name)
		b.WriteString("\t}\n")
	case md.Slice:
		fmt.Fprintf(&b, "\tfor _, value := range %s {\n", value)
		b.WriteString(indent(renderGRPCStringConversion(expr.AsArray(md.Type).ElemType.Type, "valueStr", "value"), 2))
		fmt.Fprintf(&b, "\t\t(*md).Append(%q, valueStr)\n", md.Name)
		b.WriteString("\t}\n")
	default:
		if md.Pointer {
			fmt.Fprintf(&b, "\tif %s != nil {\n", value)
		}
		if md.Name == "Authorization" && isBearer(schemes) {
			fmt.Fprintf(&b, "\t\tif !strings.Contains(%s%s, \" \") {\n", pointerPrefix(md.Pointer), value)
			fmt.Fprintf(&b, "\t\t\t(*md).Append(%q, \"Bearer \"+%s%s)\n", md.Name, pointerPrefix(md.Pointer), value)
			b.WriteString("\t\t} else {\n")
			fmt.Fprintf(&b, "\t\t\t(*md).Append(%q, %s)\n", md.Name, renderMetadataSingleValue(md, value))
			b.WriteString("\t\t}\n")
		} else {
			fmt.Fprintf(&b, "\t\t(*md).Append(%q, %s)\n", md.Name, renderMetadataSingleValue(md, value))
		}
		if md.Pointer {
			b.WriteString("\t}\n")
		}
	}
	return b.String()
}

func renderGRPCMetadataDecode(md *MetadataData, mdVar string) string {
	var b strings.Builder
	name := fmt.Sprintf("%q", md.Name)
	switch {
	case md.TypeName == "string" || md.Type.Name() == "any":
		if md.Required {
			fmt.Fprintf(&b, "\t\tif vals := %s.Get(%s); len(vals) == 0 {\n", mdVar, name)
			fmt.Fprintf(&b, "\t\t\terr = goa.MergeErrors(err, goa.MissingFieldError(%s, \"metadata\"))\n", name)
			b.WriteString("\t\t} else {\n")
			fmt.Fprintf(&b, "\t\t\t%s = vals[0]\n", md.VarName)
			b.WriteString("\t\t}\n")
		} else {
			fmt.Fprintf(&b, "\t\tif vals := %s.Get(%s); len(vals) > 0 {\n", mdVar, name)
			fmt.Fprintf(&b, "\t\t\t%s = vals[0]\n", md.VarName)
			b.WriteString("\t\t}\n")
		}
	case md.StringSlice:
		if md.Required {
			fmt.Fprintf(&b, "\t\tif vals := %s.Get(%s); len(vals) == 0 {\n", mdVar, name)
			fmt.Fprintf(&b, "\t\t\terr = goa.MergeErrors(err, goa.MissingFieldError(%s, \"metadata\"))\n", name)
			b.WriteString("\t\t} else {\n")
			fmt.Fprintf(&b, "\t\t\t%s = vals\n", md.VarName)
			b.WriteString("\t\t}\n")
		} else {
			fmt.Fprintf(&b, "\t\t%s = %s.Get(%s)\n", md.VarName, mdVar, name)
		}
	case md.Slice:
		rawVar := md.VarName + "Raw"
		if md.Required {
			fmt.Fprintf(&b, "\t\tif %s := %s.Get(%s); len(%s) == 0 {\n", rawVar, mdVar, name, rawVar)
			fmt.Fprintf(&b, "\t\t\terr = goa.MergeErrors(err, goa.MissingFieldError(%s, \"metadata\"))\n", name)
			b.WriteString("\t\t} else {\n")
			b.WriteString(indent(renderGRPCSliceConversion(md, rawVar), 3))
			b.WriteString("\t\t}\n")
		} else {
			fmt.Fprintf(&b, "\t\tif %s := %s.Get(%s); len(%s) > 0 {\n", rawVar, mdVar, name, rawVar)
			b.WriteString(indent(renderGRPCSliceConversion(md, rawVar), 3))
			b.WriteString("\t\t}\n")
		}
	default:
		rawVar := md.VarName + "Raw"
		if md.Required {
			fmt.Fprintf(&b, "\t\tif vals := %s.Get(%s); len(vals) == 0 {\n", mdVar, name)
			fmt.Fprintf(&b, "\t\t\terr = goa.MergeErrors(err, goa.MissingFieldError(%s, \"metadata\"))\n", name)
			b.WriteString("\t\t} else {\n")
			fmt.Fprintf(&b, "\t\t\t%s = vals[0]\n", rawVar)
			b.WriteString("\n")
			b.WriteString(indent(renderGRPCStringParse(md, rawVar), 3))
			b.WriteString("\t\t}\n")
		} else {
			fmt.Fprintf(&b, "\t\tif vals := %s.Get(%s); len(vals) > 0 {\n", mdVar, name)
			fmt.Fprintf(&b, "\t\t\t%s = vals[0]\n", rawVar)
			b.WriteString("\n")
			b.WriteString(indent(renderGRPCStringParse(md, rawVar), 3))
			b.WriteString("\t\t}\n")
		}
	}
	return b.String()
}

func renderGRPCRequestMetadataDecode(md *MetadataData) string {
	var b strings.Builder
	name := fmt.Sprintf("%q", md.Name)
	switch {
	case md.TypeName == "string" || md.Type.Name() == "any":
		if md.Required {
			fmt.Fprintf(&b, "\t\tif vals := md.Get(%s); len(vals) == 0 {\n", name)
			fmt.Fprintf(&b, "\t\t\terr = goa.MergeErrors(err, goa.MissingFieldError(%s, \"metadata\"))\n", name)
			b.WriteString("\t\t} else {\n")
			fmt.Fprintf(&b, "\t\t\t%s = vals[0]\n", md.VarName)
			b.WriteString("\t\t}\n")
		} else {
			fmt.Fprintf(&b, "\t\tif vals := md.Get(%s); len(vals) > 0 {\n", name)
			if md.Pointer {
				fmt.Fprintf(&b, "\t\t\t%s = &vals[0]\n", md.VarName)
			} else {
				fmt.Fprintf(&b, "\t\t\t%s = vals[0]\n", md.VarName)
			}
			b.WriteString("\t\t}\n")
		}
	case md.StringSlice:
		if md.Required {
			fmt.Fprintf(&b, "\t\tif vals := md.Get(%s); len(vals) == 0 {\n", name)
			fmt.Fprintf(&b, "\t\t\terr = goa.MergeErrors(err, goa.MissingFieldError(%s, \"metadata\"))\n", name)
			b.WriteString("\t\t} else {\n")
			fmt.Fprintf(&b, "\t\t\t%s = vals\n", md.VarName)
			b.WriteString("\t\t}\n")
		} else {
			fmt.Fprintf(&b, "\t\t%s = md.Get(%s)\n", md.VarName, name)
		}
	case md.Slice:
		rawVar := md.VarName + "Raw"
		if md.Required {
			fmt.Fprintf(&b, "\t\tif %s := md.Get(%s); len(%s) == 0 {\n", rawVar, name, rawVar)
			fmt.Fprintf(&b, "\t\t\terr = goa.MergeErrors(err, goa.MissingFieldError(%s, \"metadata\"))\n", name)
			b.WriteString("\t\t} else {\n")
			b.WriteString(indent(renderGRPCSliceConversion(md, rawVar), 3))
			b.WriteString("\t\t}\n")
		} else {
			fmt.Fprintf(&b, "\t\tif %s := md.Get(%s); len(%s) > 0 {\n", rawVar, name, rawVar)
			b.WriteString(indent(renderGRPCSliceConversion(md, rawVar), 3))
			b.WriteString("\t\t}\n")
		}
	default:
		rawVar := md.VarName + "Raw"
		if md.Required {
			fmt.Fprintf(&b, "\t\tif vals := md.Get(%s); len(vals) == 0 {\n", name)
			fmt.Fprintf(&b, "\t\t\terr = goa.MergeErrors(err, goa.MissingFieldError(%s, \"metadata\"))\n", name)
			b.WriteString("\t\t} else {\n")
			fmt.Fprintf(&b, "\t\t\t%s := vals[0]\n", rawVar)
			b.WriteString("\n")
			b.WriteString(indent(renderGRPCStringParse(md, rawVar), 3))
			b.WriteString("\t\t}\n")
		} else {
			fmt.Fprintf(&b, "\t\tif vals := md.Get(%s); len(vals) > 0 {\n", name)
			fmt.Fprintf(&b, "\t\t\t%s := vals[0]\n", rawVar)
			b.WriteString("\n")
			b.WriteString(indent(renderGRPCStringParse(md, rawVar), 3))
			b.WriteString("\t\t}\n")
		}
	}
	return b.String()
}

func renderGRPCMetadataEncode(md *MetadataData, targetVar string) string {
	var b strings.Builder
	switch {
	case md.StringSlice:
		fmt.Fprintf(&b, "\t%s.Append(%q, res.%s...)\n", targetVar, md.Name, md.FieldName)
	case md.Slice:
		fmt.Fprintf(&b, "\tfor _, value := range res.%s {\n", md.FieldName)
		b.WriteString(indent(renderGRPCStringConversion(expr.AsArray(md.Type).ElemType.Type, "valueStr", "value"), 2))
		fmt.Fprintf(&b, "\t\t%s.Append(%q, valueStr)\n", targetVar, md.Name)
		b.WriteString("\t}\n")
	default:
		if md.Pointer {
			fmt.Fprintf(&b, "\tif res.%s != nil {\n", md.FieldName)
		}
		fmt.Fprintf(&b, "\t\t%s.Append(%q, %s)\n", targetVar, md.Name, renderTemplateMetadataValue(md))
		if md.Pointer {
			b.WriteString("\t}\n")
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
		return "fmt.Sprintf(\"%v\", " + pointerPrefix(md.Pointer) + value + ")"
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
		return "fmt.Sprintf(\"%v\", *p." + md.FieldName + ")"
	}
}

func renderGRPCStringConversion(dt expr.DataType, varName, target string) string {
	switch dt.Name() {
	case "boolean":
		return fmt.Sprintf("%s := strconv.FormatBool(%s)\n", varName, target)
	case "int":
		return fmt.Sprintf("%s := strconv.Itoa(%s)\n", varName, target)
	case "int32":
		return fmt.Sprintf("%s := strconv.FormatInt(int64(%s), 10)\n", varName, target)
	case "int64":
		return fmt.Sprintf("%s := strconv.FormatInt(%s, 10)\n", varName, target)
	case "uint":
		return fmt.Sprintf("%s := strconv.FormatUint(uint64(%s), 10)\n", varName, target)
	case "uint32":
		return fmt.Sprintf("%s := strconv.FormatUint(uint64(%s), 10)\n", varName, target)
	case "uint64":
		return fmt.Sprintf("%s := strconv.FormatUint(%s, 10)\n", varName, target)
	case "float32":
		return fmt.Sprintf("%s := strconv.FormatFloat(float64(%s), 'f', -1, 32)\n", varName, target)
	case "float64":
		return fmt.Sprintf("%s := strconv.FormatFloat(%s, 'f', -1, 64)\n", varName, target)
	case "string":
		return fmt.Sprintf("%s := %s\n", varName, target)
	case "bytes":
		return fmt.Sprintf("%s := string(%s)\n", varName, target)
	case "any":
		return fmt.Sprintf("%s := fmt.Sprintf(\"%%v\", %s)\n", varName, target)
	default:
		return fmt.Sprintf("// unsupported type %s for field %s\n", dt.Name(), varName)
	}
}

func renderGRPCStringParse(md *MetadataData, rawVar string) string {
	name := fmt.Sprintf("%q", md.VarName)
	switch md.Type.Name() {
	case "bytes":
		return fmt.Sprintf("%s = []byte(%s)\n", md.VarName, rawVar)
	case "int":
		return fmt.Sprintf("v, err2 := strconv.ParseInt(%s, 10, strconv.IntSize)\nif err2 != nil {\n\terr = goa.MergeErrors(err, goa.InvalidFieldTypeError(%s, %s, \"integer\"))\n}\n%s", rawVar, name, rawVar, renderParsedAssign(md, "int(v)"))
	case "int32":
		return fmt.Sprintf("v, err2 := strconv.ParseInt(%s, 10, 32)\nif err2 != nil {\n\terr = goa.MergeErrors(err, goa.InvalidFieldTypeError(%s, %s, \"integer\"))\n}\n%s", rawVar, name, rawVar, renderParsedAssign(md, "int32(v)"))
	case "int64":
		return fmt.Sprintf("v, err2 := strconv.ParseInt(%s, 10, 64)\nif err2 != nil {\n\terr = goa.MergeErrors(err, goa.InvalidFieldTypeError(%s, %s, \"integer\"))\n}\n%s", rawVar, name, rawVar, renderDirectOrValueAssign(md, "v"))
	case "uint":
		return fmt.Sprintf("v, err2 := strconv.ParseUint(%s, 10, strconv.IntSize)\nif err2 != nil {\n\terr = goa.MergeErrors(err, goa.InvalidFieldTypeError(%s, %s, \"unsigned integer\"))\n}\n%s", rawVar, name, rawVar, renderParsedAssign(md, "uint(v)"))
	case "uint32":
		return fmt.Sprintf("v, err2 := strconv.ParseUint(%s, 10, 32)\nif err2 != nil {\n\terr = goa.MergeErrors(err, goa.InvalidFieldTypeError(%s, %s, \"unsigned integer\"))\n}\n%s", rawVar, name, rawVar, renderParsedAssign(md, "uint32(v)"))
	case "uint64":
		return fmt.Sprintf("v, err2 := strconv.ParseUint(%s, 10, 64)\nif err2 != nil {\n\terr = goa.MergeErrors(err, goa.InvalidFieldTypeError(%s, %s, \"unsigned integer\"))\n}\n%s", rawVar, name, rawVar, renderDirectOrValueAssign(md, "v"))
	case "float32":
		return fmt.Sprintf("v, err2 := strconv.ParseFloat(%s, 32)\nif err2 != nil {\n\terr = goa.MergeErrors(err, goa.InvalidFieldTypeError(%s, %s, \"float\"))\n}\n%s", rawVar, name, rawVar, renderParsedAssign(md, "float32(v)"))
	case "float64":
		return fmt.Sprintf("v, err2 := strconv.ParseFloat(%s, 64)\nif err2 != nil {\n\terr = goa.MergeErrors(err, goa.InvalidFieldTypeError(%s, %s, \"float\"))\n}\n%s", rawVar, name, rawVar, renderDirectOrValueAssign(md, "v"))
	case "boolean":
		return fmt.Sprintf("v, err2 := strconv.ParseBool(%s)\nif err2 != nil {\n\terr = goa.MergeErrors(err, goa.InvalidFieldTypeError(%s, %s, \"boolean\"))\n}\n%s", rawVar, name, rawVar, renderDirectOrValueAssign(md, "v"))
	default:
		return fmt.Sprintf("// unsupported type %s for var %s\n", md.Type.Name(), md.VarName)
	}
}

func renderParsedAssign(md *MetadataData, value string) string {
	if md.Pointer {
		return fmt.Sprintf("pv := %s\n%s = &pv\n", value, md.VarName)
	}
	return fmt.Sprintf("%s = %s\n", md.VarName, value)
}

func renderDirectOrValueAssign(md *MetadataData, value string) string {
	if md.Pointer {
		return fmt.Sprintf("%s = &%s\n", md.VarName, value)
	}
	return fmt.Sprintf("%s = %s\n", md.VarName, value)
}

func renderGRPCSliceConversion(md *MetadataData, rawVar string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s = make(%s, len(%s))\n", md.VarName, md.TypeRef, rawVar)
	fmt.Fprintf(&b, "for i, rv := range %s {\n", rawVar)
	b.WriteString(indent(renderGRPCSliceItemConversion(md), 1))
	b.WriteString("}\n")
	return b.String()
}

func renderGRPCSliceItemConversion(md *MetadataData) string {
	name := fmt.Sprintf("%q", md.VarName)
	elemName := expr.AsArray(md.Type).ElemType.Type.Name()
	switch elemName {
	case "string":
		return fmt.Sprintf("%s[i] = rv\n", md.VarName)
	case "bytes":
		return fmt.Sprintf("%s[i] = []byte(rv)\n", md.VarName)
	case "int":
		return fmt.Sprintf("v, err2 := strconv.ParseInt(rv, 10, strconv.IntSize)\nif err2 != nil {\n\terr = goa.MergeErrors(err, goa.InvalidFieldTypeError(%s, %sRaw, \"array of integers\"))\n}\n%s[i] = int(v)\n", name, md.VarName, md.VarName)
	case "int32":
		return fmt.Sprintf("v, err2 := strconv.ParseInt(rv, 10, 32)\nif err2 != nil {\n\terr = goa.MergeErrors(err, goa.InvalidFieldTypeError(%s, %sRaw, \"array of integers\"))\n}\n%s[i] = int32(v)\n", name, md.VarName, md.VarName)
	case "int64":
		return fmt.Sprintf("v, err2 := strconv.ParseInt(rv, 10, 64)\nif err2 != nil {\n\terr = goa.MergeErrors(err, goa.InvalidFieldTypeError(%s, %sRaw, \"array of integers\"))\n}\n%s[i] = v\n", name, md.VarName, md.VarName)
	case "uint":
		return fmt.Sprintf("v, err2 := strconv.ParseUint(rv, 10, strconv.IntSize)\nif err2 != nil {\n\terr = goa.MergeErrors(err, goa.InvalidFieldTypeError(%s, %sRaw, \"array of unsigned integers\"))\n}\n%s[i] = uint(v)\n", name, md.VarName, md.VarName)
	case "uint32":
		return fmt.Sprintf("v, err2 := strconv.ParseUint(rv, 10, 32)\nif err2 != nil {\n\terr = goa.MergeErrors(err, goa.InvalidFieldTypeError(%s, %sRaw, \"array of unsigned integers\"))\n}\n%s[i] = int32(v)\n", name, md.VarName, md.VarName)
	case "uint64":
		return fmt.Sprintf("v, err2 := strconv.ParseUint(rv, 10, 64)\nif err2 != nil {\n\terr = goa.MergeErrors(err, goa.InvalidFieldTypeError(%s, %sRaw, \"array of unsigned integers\"))\n}\n%s[i] = v\n", name, md.VarName, md.VarName)
	case "float32":
		return fmt.Sprintf("v, err2 := strconv.ParseFloat(rv, 32)\nif err2 != nil {\n\terr = goa.MergeErrors(err, goa.InvalidFieldTypeError(%s, %sRaw, \"array of floats\"))\n}\n%s[i] = float32(v)\n", name, md.VarName, md.VarName)
	case "float64":
		return fmt.Sprintf("v, err2 := strconv.ParseFloat(rv, 64)\nif err2 != nil {\n\terr = goa.MergeErrors(err, goa.InvalidFieldTypeError(%s, %sRaw, \"array of floats\"))\n}\n%s[i] = v\n", name, md.VarName, md.VarName)
	case "boolean":
		return fmt.Sprintf("v, err2 := strconv.ParseBool(rv)\nif err2 != nil {\n\terr = goa.MergeErrors(err, goa.InvalidFieldTypeError(%s, %sRaw, \"array of booleans\"))\n}\n%s[i] = v\n", name, md.VarName, md.VarName)
	case "any":
		return fmt.Sprintf("%s[i] = rv\n", md.VarName)
	default:
		return fmt.Sprintf("// unsupported slice type %s for var %s\n", elemName, md.VarName)
	}
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
