package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/unionjson"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

func renderMultipartFieldMatcher(body *TypeData, field *FormFieldData, codec httpencoding.MultipartCodec) string {
	element := formUnionElement(body, field)
	var out strings.Builder
	fmt.Fprintf(&out, "matchMultipartField := func() (%s, bool, []int, error) {\nvar result %s\nif !loomhttp.MultipartFieldPresent(multipartForm, %s, %s) {\nreturn result, false, nil, nil\n}\n", element.TypeRef, element.TypeRef, formKeySetLiteral(codec.Keys), parameterNamespaceLiteral(element))
	renderMultipartFieldBranches(&out, body, field, codec, "")
	fmt.Fprintf(&out, "matched, err := loomhttp.MatchUntaggedText(%q, branches, %t)\nif err != nil {\nreturn result, true, nil, err\n}\n", field.HTTPName, field.Attribute.UnionClosed())
	if strings.HasPrefix(element.TypeRef, "*") {
		fmt.Fprintf(&out, "result = new(%s)\n", element.TypeName)
	}
	out.WriteString("path := []int{matched}\n")
	renderTextUnionSelection(&out, expr.AsUnion(field.Attribute.Type), "result", "path", "")
	out.WriteString("return result, true, path, nil\n}\n")
	return out.String()
}

func renderMultipartFieldBranches(out *strings.Builder, body *TypeData, field *FormFieldData, codec httpencoding.MultipartCodec, prefix string) {
	union := expr.AsUnion(field.Attribute.Type)
	for index, branch := range union.Values {
		id := fmt.Sprintf("%s%d", prefix, index)
		childCodec := multipartBranchCodec(codec, index)
		context := body.context.Dup()
		representation.ApplyLayout(context, branch.Attribute)
		validation, pointer := context.UnionBranchContext(branch.Attribute)
		ref := context.Scope.Ref(branch.Attribute, context.Pkg(branch.Attribute))
		fmt.Fprintf(out, "var candidate%s %s\n", id, ref)
		childField := *field
		childField.Attribute, childField.Required = branch.Attribute, false
		if expr.AsUnion(branch.Attribute.Type) != nil {
			renderMultipartFieldBranches(out, body, &childField, childCodec, id+"_")
			fmt.Fprintf(out, "var path%s []int\nbranch%s := loomhttp.ComposeTextUnion(%q, branches%s_, %t, func(matched int) (bool, error) {\n", id, id, field.HTTPName, id, branch.Attribute.UnionClosed())
			if pointer {
				fmt.Fprintf(out, "candidate%s = new(%s)\n", id, context.Scope.Name(branch.Attribute, context.Pkg(branch.Attribute), false, false))
			}
			fmt.Fprintf(out, "path%s = []int{matched}\n", id)
			renderTextUnionSelection(out, expr.AsUnion(branch.Attribute.Type), "candidate"+id, "path"+id, id+"_")
			out.WriteString("return true, nil\n})\n")
			continue
		}
		shape, err := representation.MultipartFieldJSONShape(branch.Attribute, childCodec, context)
		if err != nil {
			panic(err)
		}
		unionjson.RenderShape(out, "partSchema"+id, shape)
		fmt.Fprintf(out, "partView%s, partMatch%s, partError%s := func() (jsontext.Value, bool, error) {\n", id, id, id)
		renderMultipartValueProjection(out, branch.Attribute, childCodec, &childField, context)
		out.WriteString("}()\n")
		fmt.Fprintf(out, "branch%s := loomhttp.TextUnionBranch{Schema: func() (bool, error) {\nif partError%s != nil || !partMatch%s || partView%s == nil {\nreturn false, partError%s\n}\nreturn partSchema%s.Match(partView%s)\n}, Decode: func() (bool, error) {\nvar v %s\nif err := func() (err error) {\n", id, id, id, id, id, id, id, ref)
		childBody := &TypeData{attribute: &expr.AttributeExpr{Type: &expr.Object{{Name: field.AttributeName, Attribute: branch.Attribute}}, Validation: &expr.ValidationExpr{Required: []string{field.AttributeName}}}, context: validation}
		childField.Required = true
		renderMultipartFieldDecode(out, childBody, &childField, childCodec, "v")
		out.WriteString(codegen.AttributeValidationCode(branch.Attribute, nil, validation, true, expr.IsAlias(branch.Attribute.Type), "v", field.HTTPName))
		fmt.Fprintf(out, "\nreturn\n}(); err != nil {\nreturn false, nil\n}\ncandidate%s = v\nreturn true, nil\n}, Annotations: func() (loomhttp.TextUnionAnnotations, error) {\n", id)
		if childCodec.StyleCodec != nil && (expr.IsObject(branch.Attribute.Type) || expr.IsMap(branch.Attribute.Type)) {
			element := &Element{Namespace: field.Namespace}
			if childCodec.StyleCodec.Shape == httpencoding.Object && childCodec.StyleCodec.Explode {
				element.Namespace.OwnedLiterals = append([]string{}, childCodec.Keys.Literals...)
			}
			fmt.Fprintf(out, "return (loomhttp.TextBranch{View: func() (loomhttp.ParamValue, error) {\nreturn loomhttp.DecodeMultipartParam(multipartForm, %q, %s, %s, %s)\n}, Shape: %s, Evaluate: partSchema%s.Evaluated}).Annotations()\n", field.HTTPName, parameterCodecLiteral(*childCodec.StyleCodec), parameterKindLiteral(*childCodec.StyleCodec), parameterNamespaceLiteral(element), textUnionShape(branch.Attribute), id)
		} else {
			out.WriteString("return loomhttp.TextUnionAnnotations{}, nil\n")
		}
		out.WriteString("}}\n")
	}
	fmt.Fprintf(out, "branches%s := []loomhttp.TextUnionBranch{\n", prefix)
	for index := range union.Values {
		fmt.Fprintf(out, "branch%s%d,\n", prefix, index)
	}
	out.WriteString("}\n")
}
