package codegen

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/internal/formgen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/unionjson"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

func multipartTextCodec(attribute *expr.AttributeExpr, codec httpencoding.MultipartCodec) httpencoding.Codec {
	if codec.StyleCodec != nil {
		return *codec.StyleCodec
	}
	return httpencoding.Codec{Style: httpencoding.Form, Shape: expr.HTTPValueShape(attribute).Shape, Explode: codec.Application == httpencoding.Items}
}

func multipartFieldData(body *TypeData, contract *expr.HTTPMultipartEncoding) map[string]*FormFieldData {
	fields := make(map[string]httpencoding.FieldCodec, len(contract.Fields))
	for _, field := range *expr.AsObject(body.attribute.Type) {
		captured := contract.Fields[field.Name]
		fields[field.Name] = httpencoding.FieldCodec{Name: captured.Name, Omitted: captured.Omitted, Codec: multipartTextCodec(field.Attribute, captured.Codec), Keys: captured.Codec.Keys}
	}
	styled, ordinary := formFieldData(body.attribute, fields)
	result := make(map[string]*FormFieldData, len(fields))
	for _, field := range append(styled, ordinary...) {
		result[field.AttributeName] = field
	}
	return result
}

func multipartValueUnion(attribute *expr.AttributeExpr, codec httpencoding.MultipartCodec) bool {
	return expr.AsUnion(attribute.Type) != nil && (codec.Kind == httpencoding.BranchPart || codec.StyleCodec != nil)
}

func multipartBranchCodec(codec httpencoding.MultipartCodec, index int) httpencoding.MultipartCodec {
	if codec.Kind == httpencoding.BranchPart {
		return codec.Branches[index]
	}
	result := codec
	branch := codec.StyleCodec.Branches[index]
	result.StyleCodec, result.Keys = &branch.Codec, branch.Keys
	return result
}

func renderMultipartValueProjection(out *strings.Builder, attribute *expr.AttributeExpr, codec httpencoding.MultipartCodec, field *FormFieldData, context *codegen.AttributeContext) {
	name := strconv.Quote(field.HTTPName)
	if multipartValueUnion(attribute, codec) {
		union := expr.AsUnion(attribute.Type)
		for index, branch := range union.Values {
			shape, err := representation.MultipartFieldJSONShape(branch.Attribute, multipartBranchCodec(codec, index), context)
			if err != nil {
				panic(err)
			}
			unionjson.RenderShape(out, fmt.Sprintf("partSchema%d", index), shape)
		}
		fmt.Fprintf(out, "return loomhttp.FormTextUnionValue(%s, []loomhttp.FormValueBranch{\n", name)
		for index, branch := range union.Values {
			out.WriteString("{View: func() (jsontext.Value, bool, error) {\n")
			renderMultipartValueProjection(out, branch.Attribute, multipartBranchCodec(codec, index), field, context)
			fmt.Fprintf(out, "}, Member: partSchema%d.Match, Evaluate: partSchema%d.Evaluated},\n", index, index)
		}
		fmt.Fprintf(out, "}, %t)\n", attribute.UnionClosed())
		return
	}
	switch codec.Kind {
	case httpencoding.JSONPart:
		fmt.Fprintf(out, "return loomhttp.MultipartJSONValue(multipartForm, %s, %s)\n", name, formgen.Application(codec.Application))
	case httpencoding.BinaryPart:
		fmt.Fprintf(out, "return loomhttp.MultipartBinaryValue(multipartForm, %s, %s)\n", name, formgen.Application(codec.Application))
	default:
		text := multipartTextCodec(attribute, codec)
		element := &Element{Namespace: field.Namespace}
		if text.Shape == httpencoding.Object && text.Explode {
			element.Namespace.OwnedLiterals = append([]string{}, codec.Keys.Literals...)
		}
		fmt.Fprintf(out, "return loomhttp.FormTextValue(loomhttp.TextBranch{View: func() (loomhttp.ParamValue, error) {\nreturn loomhttp.DecodeMultipartParam(multipartForm, %s, %s, %s, %s)\n}, Shape: %s})\n", name, parameterCodecLiteral(text), parameterKindLiteral(text), parameterNamespaceLiteral(element), textUnionShape(attribute))
	}
}

func renderMultipartFieldDecode(out *strings.Builder, body *TypeData, field *FormFieldData, codec httpencoding.MultipartCodec, target string) {
	if multipartValueUnion(field.Attribute, codec) {
		out.WriteString("{\n")
		out.WriteString(renderMultipartFieldMatcher(body, field, codec))
		value := "matched"
		if !formUnionElement(body, field).FieldPointer {
			value = "*matched"
		}
		fmt.Fprintf(out, "matched, present, _, partError := matchMultipartField()\nif partError != nil {\nreturn partError\n}\nif present {\n%s = %s\n", target, value)
		if field.Required {
			fmt.Fprintf(out, "} else {\nreturn loom.MissingFieldError(%q, \"body\")\n", field.HTTPName)
		}
		out.WriteString("}\n}\n")
		return
	}
	out.WriteString("{\n")
	name := strconv.Quote(field.HTTPName)
	switch codec.Kind {
	case httpencoding.JSONPart:
		fmt.Fprintf(out, "partPresent, partError := loomhttp.DecodeMultipartJSON(multipartForm, %s, &%s, %t, %s)\n", name, target, expr.AllowsNull(field.Attribute), formgen.Application(codec.Application))
	case httpencoding.BinaryPart:
		fmt.Fprintf(out, "partPresent, partError := loomhttp.DecodeMultipartBinary(multipartForm, %s, &%s, %s)\n", name, target, formgen.Application(codec.Application))
	default:
		text := multipartTextCodec(field.Attribute, codec)
		element := &Element{Namespace: field.Namespace}
		if text.Shape == httpencoding.Object && text.Explode {
			element.Namespace.OwnedLiterals = append([]string{}, codec.Keys.Literals...)
		}
		fmt.Fprintf(out, "parameterValue, partError := loomhttp.DecodeMultipartParam(multipartForm, %s, %s, %s, %s)\npartPresent := parameterValue.Present\n", name, parameterCodecLiteral(text), parameterKindLiteral(text), parameterNamespaceLiteral(element))
		out.WriteString("if partError == nil && partPresent {\n")
		key, physical := body.attribute.FindAttribute(field.AttributeName)
		out.WriteString(renderParameterAssignment(field.Attribute, physical, body.context, "parameterValue", target, field.HTTPName, body.context.IsPrimitivePointer(key, body.attribute)))
		out.WriteString("}\n")
	}
	out.WriteString("if partError != nil {\nreturn partError\n}\n")
	if field.Required {
		fmt.Fprintf(out, "if !partPresent {\nreturn loom.MissingFieldError(%q, \"body\")\n}\n", field.HTTPName)
	} else {
		out.WriteString("_ = partPresent\n")
	}
	out.WriteString("}\n")
}

func renderMultipartLeafDecode(body *TypeData, contract *expr.HTTPMultipartEncoding, target string) string {
	var out strings.Builder
	fields := multipartFieldData(body, contract)
	for _, field := range *expr.AsObject(body.attribute.Type) {
		captured := contract.Fields[field.Name]
		if captured.Omitted {
			continue
		}
		data := fields[expr.AttributeName(field.Name)]
		if captured.Derived != nil {
			fmt.Fprintf(&out, "{\nderivedJSON, derived, derivedError := %s\nif derivedError != nil {\nreturn derivedError\n}\nif derived {\nif derivedError := json.Unmarshal(derivedJSON, &%s.%s, loom.JSONOptions()); derivedError != nil {\nreturn derivedError\n}\n} else {\n", multipartMetadataView(contract, captured), target, data.FieldName)
		}
		renderMultipartFieldDecode(&out, body, data, captured.Codec, target+"."+data.FieldName)
		if captured.Derived != nil {
			out.WriteString("}\n}\n")
		}
	}
	return out.String()
}

func multipartMetadataView(contract *expr.HTTPMultipartEncoding, field expr.HTTPMultipartFieldEncoding) string {
	header := "filename"
	if field.Derived.Kind == expr.HTTPMultipartContentType {
		header = "Content-Type"
	}
	return fmt.Sprintf("loomhttp.MultipartMetadataJSON(multipartForm, %q, %q, %q)", field.Name, contract.Fields[field.Derived.File].Name, header)
}
