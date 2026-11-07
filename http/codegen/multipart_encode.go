package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen/unionnames"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

func renderMultipartEncode(body *TypeData, contract *expr.HTTPMultipartEncoding) string {
	var out strings.Builder
	out.WriteString("multipartForm := &loomhttp.MultipartForm{}\n")
	if len(contract.Branches) == 0 {
		out.WriteString(renderMultipartLeafEncode(body, contract, "body"))
	} else {
		out.WriteString("var selected []int\nif partError := func() error {\n")
		renderMultipartEncodingBranches(&out, body, contract, "body")
		out.WriteString("}(); partError != nil {\nreturn nil, partError\n}\n")
		out.WriteString(renderMultipartMatcher(body, contract))
		out.WriteString("_, matched, partError := matchMultipart()\nif partError != nil {\nreturn nil, partError\n}\n")
		out.WriteString(multipartPathCheck("selected", "matched", "return nil, "))
	}
	out.WriteString("return multipartForm, nil\n")
	return out.String()
}

func renderMultipartEncodingBranches(out *strings.Builder, body *TypeData, contract *expr.HTTPMultipartEncoding, source string) {
	members := unionnames.Resolve(expr.AsUnion(body.attribute.Type))
	for index, branch := range expr.AsUnion(body.attribute.Type).Values {
		child := formBranchBody(body, branch.Attribute)
		fmt.Fprintf(out, "if v, ok := %s.As%s(); ok {\nselected = append(selected, %d)\n", source, members[branch.Name].Method, index)
		_, pointer := child.context.UnionBranchContext(branch.Attribute)
		if pointer {
			out.WriteString("if v == nil {\nreturn fmt.Errorf(\"selected multipart branch is nil\")\n}\n")
		}
		if len(contract.Branches[index].Branches) != 0 {
			renderMultipartEncodingBranches(out, child, contract.Branches[index], "v")
		} else {
			out.WriteString("if _, partError := func() (*loomhttp.MultipartForm, error) {\n")
			out.WriteString(renderMultipartLeafEncode(child, contract.Branches[index], "v"))
			out.WriteString("return multipartForm, nil\n}(); partError != nil {\nreturn partError\n}\n")
			out.WriteString("return nil\n")
		}
		out.WriteString("}\n")
	}
	out.WriteString("return fmt.Errorf(\"multipart union has no selected branch\")\n")
}

func renderMultipartLeafEncode(body *TypeData, contract *expr.HTTPMultipartEncoding, source string) string {
	var out strings.Builder
	fields := multipartFieldData(body, contract)
	for _, field := range *expr.AsObject(body.attribute.Type) {
		captured := contract.Fields[field.Name]
		if captured.Omitted {
			continue
		}
		data := fields[expr.AttributeName(field.Name)]
		value := source + "." + data.FieldName
		if multipartValueUnion(field.Attribute, captured.Codec) {
			out.WriteString(renderMultipartFieldEncode(body, data, captured.Codec, source))
			continue
		}
		guard := ""
		if body.context.FieldPresence(expr.NewObjectFields(body.attribute), field.Name, field.Attribute) != codegen.NativePresence {
			guard = value + ".Present()"
		} else if expr.AsUnion(field.Attribute.Type) != nil && expr.HasNativeJSONPresence(field.Attribute) {
			guard = value + ".Kind() != \"\""
		}
		out.WriteString("{\n")
		if guard != "" {
			fmt.Fprintf(&out, "if %s {\n", guard)
		}
		renderMultipartLeafFieldEncode(&out, body, data, captured.Codec, value)
		if guard != "" {
			if data.Required {
				fmt.Fprintf(&out, "} else {\nreturn nil, loom.MissingFieldError(%q, \"body\")\n", data.HTTPName)
			}
			out.WriteString("}\n")
		}
		out.WriteString("}\n")
	}
	return out.String()
}

func renderMultipartLeafFieldEncode(out *strings.Builder, body *TypeData, field *FormFieldData, codec httpencoding.MultipartCodec, value string) {
	switch codec.Kind {
	case httpencoding.JSONPart, httpencoding.BinaryPart:
		method := "AddJSON"
		if codec.Kind == httpencoding.BinaryPart {
			method = "AddBinary"
		}
		fmt.Fprintf(out, "partPresent, partError := multipartForm.%s(%q, %q, %s, %s)\nif partError != nil {\nreturn nil, partError\n}\n", method, field.HTTPName, codec.ContentType, value, formApplicationLiteral(codec.Application))
	default:
		key, physical := body.attribute.FindAttribute(field.AttributeName)
		out.WriteString(renderParameterValue(field.Attribute, physical, body.context, value, "parameterValue", body.context.IsPrimitivePointer(key, body.attribute)))
		text := multipartTextCodec(field.Attribute, codec)
		fmt.Fprintf(out, "partPresent := parameterValue.Present\nif partError := multipartForm.AddParam(%q, %q, %s, parameterValue, %s); partError != nil {\nreturn nil, partError\n}\n", field.HTTPName, codec.ContentType, parameterCodecLiteral(text), parameterNamespaceLiteral(&Element{Namespace: field.Namespace}))
		if parameterEmptyHasNoKey(text) {
			fmt.Fprintf(out, "partPresent = loomhttp.MultipartFieldPresent(multipartForm, %s, %s)\n", formKeySetLiteral(codec.Keys), parameterNamespaceLiteral(&Element{Namespace: field.Namespace}))
		}
	}
	if field.Required {
		fmt.Fprintf(out, "if !partPresent {\nreturn nil, loom.MissingFieldError(%q, \"body\")\n}\n", field.HTTPName)
	} else {
		out.WriteString("_ = partPresent\n")
	}
}

func multipartPathCheck(selected, matched, failure string) string {
	return fmt.Sprintf("if len(%s) != len(%s) {\n%sfmt.Errorf(\"multipart encoding changed selected branch\")\n}\nfor index := range %s {\nif %s[index] != %s[index] {\n%sfmt.Errorf(\"multipart encoding changed selected branch\")\n}\n}\n", selected, matched, failure, selected, selected, matched, failure)
}
