package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen/unionnames"
	"github.com/CaliLuke/loom/expr"
)

func renderFormUnionEncode(body *TypeData, contract *expr.HTTPFormEncoding, optional bool) string {
	var out strings.Builder
	out.WriteString("var selected []int\nparameterEncoded, parameterError := func() (string, error) {\n")
	renderFormUnionEncodeBranches(&out, body, contract, "body")
	out.WriteString("}()\nif parameterError != nil {\nreturn \"\", parameterError\n}\n")
	if optional {
		out.WriteString("if parameterEncoded == \"\" {\nreturn parameterEncoded, nil\n}\n")
	}
	out.WriteString("parameterFormPairs, parameterError := loomhttp.ParseRawPairs(parameterEncoded)\nif parameterError != nil {\nreturn \"\", parameterError\n}\nparameterForm, parameterError := url.ParseQuery(parameterEncoded)\nif parameterError != nil {\nreturn \"\", parameterError\n}\n")
	out.WriteString(renderFormUnionMatcher(body, contract))
	out.WriteString("_, matchedPath, parameterError := matchFormUnion()\nif parameterError != nil {\nreturn \"\", parameterError\n}\nif len(selected) != len(matchedPath) {\nreturn \"\", fmt.Errorf(\"encoded form selects a different union branch\")\n}\nfor index := range selected {\nif selected[index] != matchedPath[index] {\nreturn \"\", fmt.Errorf(\"encoded form selects a different union branch\")\n}\n}\nreturn parameterEncoded, nil\n")
	return out.String()
}

func renderFormUnionEncodeBranches(out *strings.Builder, body *TypeData, contract *expr.HTTPFormEncoding, value string) {
	union := expr.AsUnion(body.attribute.Type)
	members := unionnames.Resolve(union)
	for index, branch := range union.Values {
		child := formBranchBody(body, branch.Attribute)
		captured := contract.Branches[index]
		fmt.Fprintf(out, "if v, ok := %s.As%s(); ok {\nselected = append(selected, %d)\n", value, members[branch.Name].Method, index)
		_, pointer := child.context.UnionBranchContext(branch.Attribute)
		if pointer {
			out.WriteString("if v == nil {\nreturn \"\", fmt.Errorf(\"selected form union branch is nil\")\n}\n")
		}
		switch {
		case expr.AsUnion(branch.Attribute.Type) != nil:
			renderFormUnionEncodeBranches(out, child, captured, "v")
		case expr.AsMap(branch.Attribute.Type) != nil:
			out.WriteString(renderFormMapEncode(child, captured, "v"))
		default:
			styled, ordinary := formFieldData(child.attribute, captured.Fields)
			out.WriteString(renderFormFieldsEncode(child, ordinary, styled, "v"))
		}
		out.WriteString("}\n")
	}
	out.WriteString("return \"\", fmt.Errorf(\"form union has no selected branch\")\n")
}
