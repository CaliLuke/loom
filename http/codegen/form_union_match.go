package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
)

func renderFormUnionMatcher(body *TypeData, contract *expr.HTTPFormEncoding) string {
	var out strings.Builder
	name := body.context.Scope.Name(body.attribute, body.context.Pkg(body.attribute), &codegen.AttributeContext{})
	fmt.Fprintf(&out, "matchFormUnion := func() (%s, []int, error) {\nvar result %s\n", name, name)
	renderFormUnionBranches(&out, body, contract, "")
	fmt.Fprintf(&out, "matched, err := loomhttp.MatchUntaggedText(%q, branches, %t)\nif err != nil {\nreturn result, nil, err\n}\npath := []int{matched}\n", "form body", contract.Body.UnionClosed())
	renderTextUnionSelection(&out, expr.AsUnion(body.attribute.Type), "result", "path", "")
	out.WriteString("return result, path, nil\n}\n")
	return out.String()
}

func renderFormUnionBranches(out *strings.Builder, body *TypeData, contract *expr.HTTPFormEncoding, prefix string) {
	union := expr.AsUnion(body.attribute.Type)
	for index, branch := range union.Values {
		id := fmt.Sprintf("%s%d", prefix, index)
		child := formBranchBody(body, branch.Attribute)
		captured := contract.Branches[index]
		validation, pointer := child.context.UnionBranchContext(branch.Attribute)
		ref := child.context.Scope.Ref(branch.Attribute, child.context.Pkg(branch.Attribute))
		fmt.Fprintf(out, "var candidate%s %s\n", id, ref)
		if nested := expr.AsUnion(branch.Attribute.Type); nested != nil {
			renderFormUnionBranches(out, child, captured, id+"_")
			fmt.Fprintf(out, "var path%s []int\nbranch%s := loomhttp.ComposeTextUnion(%q, branches%s_, %t, func(matched int) (bool, error) {\n", id, id, "form body", id, captured.Body.UnionClosed())
			if pointer {
				fmt.Fprintf(out, "candidate%s = new(%s)\n", id, child.context.Scope.Name(branch.Attribute, child.context.Pkg(branch.Attribute), &codegen.AttributeContext{}))
			}
			fmt.Fprintf(out, "path%s = []int{matched}\n", id)
			renderTextUnionSelection(out, nested, "candidate"+id, "path"+id, id+"_")
			out.WriteString("return true, nil\n})\n")
			continue
		}
		renderFormRootProjection(out, child, captured, id)
		fmt.Fprintf(out, "branch%s := loomhttp.TextUnionBranch{Schema: func() (bool, error) {\nif projectionError%s != nil || !projected%s {\nreturn false, projectionError%s\n}\nreturn projection%s.Schema(formSchema%s.Match, %t)\n}, Decode: func() (bool, error) {\nreturn projection%s.Schema(func(jsontext.Value) (bool, error) {\n", id, id, id, id, id, id, captured.ObjectOpenness == expr.ObjectClosed, id)
		styled, ordinary := formFieldData(child.attribute, captured.Fields)
		for _, field := range append(styled, ordinary...) {
			if field.Required {
				fmt.Fprintf(out, "if len(projection%s.FieldClaims[%q]) == 0 {\nreturn false, nil\n}\n", id, field.HTTPName)
			}
		}
		fmt.Fprintf(out, "var v %s\n", ref)
		if pointer {
			fmt.Fprintf(out, "v = new(%s)\n", child.context.Scope.Name(branch.Attribute, child.context.Pkg(branch.Attribute), &codegen.AttributeContext{}))
		}
		out.WriteString("if err := func() (err error) {\n")
		out.WriteString(renderFormLeafDecode(child, captured, "v"))
		out.WriteString(codegen.AttributeValidationCode(branch.Attribute, nil, validation, true, expr.IsAlias(branch.Attribute.Type), "v", "form body"))
		fmt.Fprintf(out, "\nreturn\n}(); err != nil {\nreturn false, nil\n}\ncandidate%s = v\nreturn true, nil\n}, %t)\n}, Annotations: func() (loomhttp.TextUnionAnnotations, error) {\nreturn projection%s.Annotations(formSchema%s.Evaluated, %t)\n}}\n", id, captured.ObjectOpenness == expr.ObjectClosed, id, id, captured.ObjectOpenness == expr.ObjectOpen)
	}
	fmt.Fprintf(out, "branches%s := []loomhttp.TextUnionBranch{\n", prefix)
	for index := range union.Values {
		fmt.Fprintf(out, "branch%s%d,\n", prefix, index)
	}
	out.WriteString("}\n")
}

func formBranchBody(parent *TypeData, attribute *expr.AttributeExpr) *TypeData {
	context := parent.context.Dup()
	representation.ApplyLayout(context, attribute)
	return &TypeData{attribute: attribute, context: context, Name: context.Scope.Name(attribute, context.Pkg(attribute), &codegen.AttributeContext{})}
}

func renderFormLeafDecode(body *TypeData, contract *expr.HTTPFormEncoding, target string) string {
	if expr.AsMap(body.attribute.Type) != nil {
		return renderFormMapDecode(body, contract, target)
	}
	styled, ordinary := formFieldData(body.attribute, contract.Fields)
	return "if fieldError := func() error {\n" + renderFormFieldsDecode(body, ordinary, styled, target) + "}(); fieldError != nil {\nreturn fieldError\n}\n"
}
