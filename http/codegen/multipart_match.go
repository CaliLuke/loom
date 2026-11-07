package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/unionjson"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
)

func renderMultipartMatcher(body *TypeData, contract *expr.HTTPMultipartEncoding) string {
	var out strings.Builder
	name := body.context.Scope.Name(body.attribute, body.context.Pkg(body.attribute), &codegen.AttributeContext{})
	fmt.Fprintf(&out, "matchMultipart := func() (%s, []int, error) {\nvar result %s\n", name, name)
	renderMultipartBranches(&out, body, contract, "")
	fmt.Fprintf(&out, "matched, err := loomhttp.MatchUntaggedText(%q, branches, %t)\nif err != nil {\nreturn result, nil, err\n}\npath := []int{matched}\n", "multipart body", contract.Body.UnionClosed())
	renderTextUnionSelection(&out, expr.AsUnion(body.attribute.Type), "result", "path", "")
	out.WriteString("return result, path, nil\n}\n")
	return out.String()
}

func renderMultipartBranches(out *strings.Builder, body *TypeData, contract *expr.HTTPMultipartEncoding, prefix string) {
	union := expr.AsUnion(body.attribute.Type)
	for index, branch := range union.Values {
		id := fmt.Sprintf("%s%d", prefix, index)
		child := formBranchBody(body, branch.Attribute)
		captured := contract.Branches[index]
		validation, pointer := child.context.UnionBranchContext(branch.Attribute)
		ref := child.context.Scope.Ref(branch.Attribute, child.context.Pkg(branch.Attribute))
		fmt.Fprintf(out, "var candidate%s %s\n", id, ref)
		if nested := expr.AsUnion(branch.Attribute.Type); nested != nil {
			renderMultipartBranches(out, child, captured, id+"_")
			fmt.Fprintf(out, "var path%s []int\nbranch%s := loomhttp.ComposeTextUnion(%q, branches%s_, %t, func(matched int) (bool, error) {\n", id, id, "multipart body", id, captured.Body.UnionClosed())
			if pointer {
				fmt.Fprintf(out, "candidate%s = new(%s)\n", id, child.Name)
			}
			fmt.Fprintf(out, "path%s = []int{matched}\n", id)
			renderTextUnionSelection(out, nested, "candidate"+id, "path"+id, id+"_")
			out.WriteString("return true, nil\n})\n")
			continue
		}
		renderMultipartProjection(out, child, captured, id)
		fmt.Fprintf(out, "branch%s := loomhttp.TextUnionBranch{Schema: func() (bool, error) {\nif projectionError%s != nil || !projected%s {\nreturn false, projectionError%s\n}\nreturn projection%s.Schema(partSchema%s.Match, %t)\n}, Decode: func() (bool, error) {\nreturn projection%s.Schema(func(jsontext.Value) (bool, error) {\nvar v %s\n", id, id, id, id, id, id, captured.ObjectOpenness == expr.ObjectClosed, id, ref)
		if pointer {
			fmt.Fprintf(out, "v = new(%s)\n", child.Name)
		}
		out.WriteString("if err := func() (err error) {\n")
		out.WriteString(renderMultipartLeafDecode(child, captured, "v"))
		out.WriteString(codegen.AttributeValidationCode(branch.Attribute, nil, validation, true, expr.IsAlias(branch.Attribute.Type), "v", "multipart body"))
		fmt.Fprintf(out, "\nreturn\n}(); err != nil {\nreturn false, nil\n}\ncandidate%s = v\nreturn true, nil\n}, %t)\n}, Annotations: func() (loomhttp.TextUnionAnnotations, error) {\nreturn projection%s.Annotations(partSchema%s.Evaluated, %t)\n}}\n", id, captured.ObjectOpenness == expr.ObjectClosed, id, id, captured.ObjectOpenness == expr.ObjectOpen)
	}
	fmt.Fprintf(out, "branches%s := []loomhttp.TextUnionBranch{\n", prefix)
	for index := range union.Values {
		fmt.Fprintf(out, "branch%s%d,\n", prefix, index)
	}
	out.WriteString("}\n")
}

func renderMultipartProjection(out *strings.Builder, body *TypeData, contract *expr.HTTPMultipartEncoding, id string) {
	shape, err := representation.MultipartJSONShape(body.attribute, contract, body.context)
	if err != nil {
		panic(err)
	}
	unionjson.RenderShape(out, "partSchema"+id, shape)
	fmt.Fprintf(out, "projection%s, projected%s, projectionError%s := loomhttp.MultipartObjectJSON(multipartForm, []loomhttp.FormField{\n", id, id, id)
	fields := multipartFieldData(body, contract)
	for _, field := range *expr.AsObject(body.attribute.Type) {
		captured := contract.Fields[field.Name]
		if captured.Omitted {
			continue
		}
		fmt.Fprintf(out, "{Name: %q, Keys: %s, View: func() (jsontext.Value, bool, error) {\n", captured.Name, formKeySetLiteral(captured.Codec.Keys))
		if captured.Derived != nil {
			fmt.Fprintf(out, "if raw, derived, err := %s; err != nil || derived {\nreturn raw, err == nil, err\n}\n", multipartMetadataView(contract, captured))
		}
		renderMultipartValueProjection(out, field.Attribute, captured.Codec, fields[expr.AttributeName(field.Name)], body.context)
		out.WriteString("}},\n")
	}
	out.WriteString("})\n")
}
