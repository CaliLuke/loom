package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen/unionnames"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

func renderFormContentUnionEncode(element *Element, source string) string {
	var out strings.Builder
	out.WriteString("{\n")
	if element.FieldPointer {
		fmt.Fprintf(&out, "if %s != nil {\n", parameterServiceValue(element, source))
	}
	out.WriteString("contentValues := make(map[string][]string)\nvar selected []int\nif contentError := func() error {\n")
	renderFormContentEncodingChoice(&out, element, expr.AsUnion(element.Type), element.Codec.Branches, parameterServiceValue(element, source))
	out.WriteString("}(); contentError != nil {\nreturn contentError\n}\n{\nvar contentBuilder loomhttp.RawPairsBuilder\nparameterFormPairs, contentError := loomhttp.ParseRawPairs(contentBuilder.Encode(contentValues))\nif contentError != nil {\nreturn contentError\n}\nparameterForm := contentValues\n")
	out.WriteString(renderFormContentUnionMatcher(element))
	fmt.Fprintf(&out, "_, present, matchedPath, contentError := matchFormContent()\nif contentError != nil {\nreturn contentError\n}\nif !present && %t {\nreturn fmt.Errorf(%q)\n}\nif present {\nif len(selected) != len(matchedPath) {\nreturn fmt.Errorf(\"form content union changed selected branch\")\n}\nfor index := range selected {\nif selected[index] != matchedPath[index] {\nreturn fmt.Errorf(\"form content union changed selected branch\")\n}\n}\n}\n}\nif values, present := contentValues[%s]; present {\nparameterForm[%s] = values\n}\n", element.Required, "required form field "+element.HTTPName+" has no encoded items", parameterWireName(element), parameterWireName(element))
	if element.FieldPointer {
		if element.Required {
			fmt.Fprintf(&out, "} else {\nreturn loom.MissingFieldError(%s, \"body\")\n", parameterWireName(element))
		}
		out.WriteString("}\n")
	}
	out.WriteString("}\n")
	return out.String()
}

func renderFormContentEncodingChoice(out *strings.Builder, element *Element, union *expr.Union, codecs []httpencoding.BranchCodec, value string) {
	members := unionnames.Resolve(union)
	for index, branch := range union.Values {
		codec := codecs[index].Codec
		fmt.Fprintf(out, "if v, ok := %s.As%s(); ok {\nselected = append(selected, %d)\n", value, members[branch.Name].Method, index)
		if nested := expr.AsUnion(branch.Attribute.Type); nested != nil {
			context := element.context.Dup()
			representation.ApplyLayout(context, branch.Attribute)
			_, pointer := context.UnionBranchContext(branch.Attribute)
			if pointer {
				out.WriteString("if v == nil {\nreturn fmt.Errorf(\"selected form content union branch is nil\")\n}\n")
			}
			renderFormContentEncodingChoice(out, element, nested, codec.Branches, "v")
		} else if codec.Style == httpencoding.JSON {
			fmt.Fprintf(out, "if _, contentError := loomhttp.EncodeJSONField(contentValues, %s, v, %s); contentError != nil {\nreturn contentError\n}\nreturn nil\n", parameterWireName(element), formApplicationLiteral(codec.Application))
		} else {
			fmt.Fprintf(out, "if _, contentError := loomhttp.EncodeFormValue(contentValues, %s, v); contentError != nil {\nreturn contentError\n}\nreturn nil\n", parameterWireName(element))
		}
		out.WriteString("}\n")
	}
	out.WriteString("return fmt.Errorf(\"form content union has no selected branch\")\n")
}
