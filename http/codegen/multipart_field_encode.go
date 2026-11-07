package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen/unionnames"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

func renderMultipartFieldEncode(body *TypeData, field *FormFieldData, codec httpencoding.MultipartCodec, source string) string {
	var out strings.Builder
	element := formUnionElement(body, field)
	value := source + "." + field.FieldName
	guard := value + ".Kind() != \"\""
	if element.FieldPointer {
		guard = value + " != nil"
	}
	fmt.Fprintf(&out, "{\nif %s {\nouterMultipart := multipartForm\nmultipartForm := &loomhttp.MultipartForm{}\nvar selected []int\nif _, partError := func() (*loomhttp.MultipartForm, error) {\n", guard)
	renderMultipartFieldEncodingBranches(&out, body, field, codec, value)
	out.WriteString("}(); partError != nil {\nreturn nil, partError\n}\n")
	out.WriteString(renderMultipartFieldMatcher(body, field, codec))
	fmt.Fprintf(&out, "_, present, matched, partError := matchMultipartField()\nif partError != nil {\nreturn nil, partError\n}\nif !present && %t {\nreturn nil, loom.MissingFieldError(%q, \"body\")\n}\nif present {\n", field.Required, field.HTTPName)
	out.WriteString(multipartPathCheck("selected", "matched", "return nil, "))
	out.WriteString("}\nouterMultipart.Parts = append(outerMultipart.Parts, multipartForm.Parts...)\n")
	if field.Required {
		fmt.Fprintf(&out, "} else {\nreturn nil, loom.MissingFieldError(%q, \"body\")\n", field.HTTPName)
	}
	out.WriteString("}\n}\n")
	return out.String()
}

func renderMultipartFieldEncodingBranches(out *strings.Builder, body *TypeData, field *FormFieldData, codec httpencoding.MultipartCodec, value string) {
	members := unionnames.Resolve(expr.AsUnion(field.Attribute.Type))
	for index, branch := range expr.AsUnion(field.Attribute.Type).Values {
		fmt.Fprintf(out, "if v, ok := %s.As%s(); ok {\nselected = append(selected, %d)\n", value, members[branch.Name].Method, index)
		childCodec := multipartBranchCodec(codec, index)
		context := body.context.Dup()
		representation.ApplyLayout(context, branch.Attribute)
		context, pointer := context.UnionBranchContext(branch.Attribute)
		if pointer {
			out.WriteString("if v == nil {\nreturn nil, fmt.Errorf(\"selected multipart field branch is nil\")\n}\n")
		}
		child := *field
		child.Attribute = branch.Attribute
		childBody := &TypeData{attribute: &expr.AttributeExpr{Type: &expr.Object{{Name: field.AttributeName, Attribute: branch.Attribute}}, Validation: &expr.ValidationExpr{Required: []string{field.AttributeName}}}, context: context}
		if expr.AsUnion(branch.Attribute.Type) != nil {
			renderMultipartFieldEncodingBranches(out, childBody, &child, childCodec, "v")
		} else {
			renderMultipartLeafFieldEncode(out, childBody, &child, childCodec, "v")
			out.WriteString("return multipartForm, nil\n")
		}
		out.WriteString("}\n")
	}
	out.WriteString("return nil, fmt.Errorf(\"multipart field has no selected branch\")\n")
}
