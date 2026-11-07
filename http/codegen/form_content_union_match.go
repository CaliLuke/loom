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

// renderFormContentUnionMatcher keeps the repeated field immutable while every
// alternative applies its captured item grammar in independent value domains.
func renderFormContentUnionMatcher(element *Element) string {
	var out strings.Builder
	fmt.Fprintf(&out, "matchFormContent := func() (%s, bool, []int, error) {\nvar result %s\nif len(parameterForm[%s]) == 0 {\nreturn result, false, nil, nil\n}\n", element.TypeRef, element.TypeRef, parameterWireName(element))
	renderFormContentBranches(&out, element, expr.AsUnion(element.Type), element.Codec.Branches, "")
	fmt.Fprintf(&out, "matched, err := loomhttp.MatchUntaggedText(%s, branches, %t)\nif err != nil {\nreturn result, true, nil, err\n}\n", parameterWireName(element), element.attribute.UnionClosed())
	if strings.HasPrefix(element.TypeRef, "*") {
		fmt.Fprintf(&out, "result = new(%s)\n", element.TypeName)
	}
	out.WriteString("path := []int{matched}\n")
	renderTextUnionSelection(&out, expr.AsUnion(element.Type), "result", "path", "")
	out.WriteString("return result, true, path, nil\n}\n")
	return out.String()
}

func renderFormContentBranches(out *strings.Builder, element *Element, union *expr.Union, codecs []httpencoding.BranchCodec, prefix string) {
	for index, branch := range union.Values {
		id := fmt.Sprintf("%s%d", prefix, index)
		codec := codecs[index].Codec
		context := element.context.Dup()
		representation.ApplyLayout(context, branch.Attribute)
		validation, pointer := context.UnionBranchContext(branch.Attribute)
		ref := context.Scope.Ref(branch.Attribute, context.Pkg(branch.Attribute))
		fmt.Fprintf(out, "var candidate%s %s\n", id, ref)
		if nested := expr.AsUnion(branch.Attribute.Type); nested != nil {
			renderFormContentBranches(out, element, nested, codec.Branches, id+"_")
			fmt.Fprintf(out, "var path%s []int\nbranch%s := loomhttp.ComposeTextUnion(%s, branches%s_, %t, func(matched int) (bool, error) {\n", id, id, parameterWireName(element), id, branch.Attribute.UnionClosed())
			if pointer {
				fmt.Fprintf(out, "candidate%s = new(%s)\n", id, context.Scope.Name(branch.Attribute, context.Pkg(branch.Attribute), &codegen.AttributeContext{}))
			}
			fmt.Fprintf(out, "path%s = []int{matched}\n", id)
			renderTextUnionSelection(out, nested, "candidate"+id, "path"+id, id+"_")
			out.WriteString("return true, nil\n})\n")
			continue
		}
		if codec.Application != httpencoding.Items || expr.AsArray(branch.Attribute.Type) == nil {
			panic("content union alternative requires a captured array item codec")
		}
		shape, err := representation.FormFieldJSONShape(branch.Attribute, codec, context)
		if err != nil {
			panic(err)
		}
		unionjson.RenderShape(out, "contentSchema"+id, shape)
		fmt.Fprintf(out, "contentView%s, contentMatch%s, contentError%s := func() (jsontext.Value, bool, error) {\n", id, id, id)
		renderFormFieldProjection(out, branch.Attribute, codec, nil, parameterWireName(element), context)
		out.WriteString("}()\n")
		fmt.Fprintf(out, "branch%s := loomhttp.TextUnionBranch{Schema: func() (bool, error) {\nif contentError%s != nil || !contentMatch%s {\nreturn false, contentError%s\n}\nreturn contentSchema%s.Match(contentView%s)\n}, Decode: func() (bool, error) {\nvar v %s\nif err := func() (err error) {\n", id, id, id, id, id, id, ref)
		if codec.Style == httpencoding.JSON {
			fmt.Fprintf(out, "if _, decodeError := loomhttp.DecodeJSONField(parameterForm, %s, &v, false, %s); decodeError != nil {\nreturn decodeError\n}\n", parameterWireName(element), formApplicationLiteral(codec.Application))
		} else {
			fmt.Fprintf(out, "if _, decodeError := loomhttp.DecodeFormValue(parameterForm, %s, &v); decodeError != nil {\nreturn decodeError\n}\n", parameterWireName(element))
		}
		out.WriteString(codegen.AttributeValidationCode(branch.Attribute, nil, validation, true, expr.IsAlias(branch.Attribute.Type), "v", element.HTTPName))
		fmt.Fprintf(out, "\nreturn\n}(); err != nil {\nreturn false, nil\n}\ncandidate%s = v\nreturn true, nil\n}, Annotations: func() (loomhttp.TextUnionAnnotations, error) {\nreturn loomhttp.TextUnionAnnotations{}, nil\n}}\n", id)
	}
	fmt.Fprintf(out, "branches%s := []loomhttp.TextUnionBranch{\n", prefix)
	for index := range union.Values {
		fmt.Fprintf(out, "branch%s%d,\n", prefix, index)
	}
	out.WriteString("}\n")
}
