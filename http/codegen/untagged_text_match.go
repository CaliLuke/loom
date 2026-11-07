package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen/unionnames"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/unionjson"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

// renderTextUnionMatcher captures candidates privately and publishes only the
// unique matching branch. Its raw carrier is supplied by the enclosing adapter.
func renderTextUnionMatcher(element *Element, location string) string {
	var out strings.Builder
	union := expr.AsUnion(element.Type)
	fmt.Fprintf(&out, "matchTextUnion := func() (%s, bool, []int, error) {\nvar result %s\npresent := false\n", element.TypeRef, element.TypeRef)
	renderTextUnionBranches(&out, element, union, element.Codec.Branches, location, "")
	out.WriteString("if !present {\nreturn result, false, nil, nil\n}\n")
	fmt.Fprintf(&out, "matched, err := loomhttp.MatchUntaggedText(%s, branches, %t)\nif err != nil {\nreturn result, true, nil, err\n}\n", parameterWireName(element), element.attribute.UnionClosed())
	if strings.HasPrefix(element.TypeRef, "*") {
		fmt.Fprintf(&out, "result = new(%s)\n", element.TypeName)
	}
	out.WriteString("path := []int{matched}\n")
	renderTextUnionSelection(&out, union, "result", "path", "")
	out.WriteString("return result, true, path, nil\n}\n")
	return out.String()
}

func renderTextUnionBranches(out *strings.Builder, element *Element, union *expr.Union, codecs []httpencoding.BranchCodec, location, prefix string) {
	for index, branch := range union.Values {
		id := fmt.Sprintf("%s%d", prefix, index)
		branchElement := textUnionBranchElement(element, codecs[index])
		ref := element.context.Scope.Ref(branch.Attribute, element.context.Pkg(branch.Attribute))
		fmt.Fprintf(out, "var candidate%s %s\n", id, ref)
		if nested := expr.AsUnion(branch.Attribute.Type); nested != nil {
			renderTextUnionBranches(out, branchElement, nested, codecs[index].Inner, location, id+"_")
			fmt.Fprintf(out, "var path%s []int\nbranch%s := loomhttp.ComposeTextUnion(%s, branches%s_, %t, func(matched int) (bool, error) {\n", id, id, parameterWireName(element), id, branch.Attribute.UnionClosed())
			if strings.HasPrefix(ref, "*") {
				name := element.context.Scope.Name(branch.Attribute, element.context.Pkg(branch.Attribute), false, false)
				fmt.Fprintf(out, "candidate%s = new(%s)\n", id, name)
			}
			fmt.Fprintf(out, "path%s = []int{matched}\n", id)
			renderTextUnionSelection(out, nested, "candidate"+id, "path"+id, id+"_")
			out.WriteString("return true, nil\n})\n")
			continue
		}
		fmt.Fprintf(out, "view%s, viewError%s := %s\npresent = present || view%s.Present || viewError%s != nil\n", id, id, textUnionView(branchElement, location), id, id)
		plan, err := unionjson.Plan(branch.Attribute, func(name string, _ *expr.AttributeExpr) string {
			return expr.ElementName(name)
		}, true, false)
		if err != nil {
			panic(err)
		}
		shape, err := plan.JSONShape()
		if err != nil {
			panic(err)
		}
		unionjson.RenderShape(out, "schema"+id, shape)
		fmt.Fprintf(out, "leaf%s := loomhttp.TextBranch{View: func() (loomhttp.ParamValue, error) {\nreturn view%s, viewError%s\n}, Shape: %s, Member: schema%s.Match, Evaluate: schema%s.Evaluated, Assign: func(raw loomhttp.ParamValue) (bool, error) {\n", id, id, id, textUnionShape(branch.Attribute), id, id)
		fmt.Fprintf(out, "var v %s\nif err := func() (err error) {\n", ref)
		out.WriteString(renderParameterAssignment(branch.Attribute, branch.Attribute, element.context, "raw", "v", element.HTTPName, false))
		branchContext := element.context.Dup()
		representation.ApplyLayout(branchContext, branch.Attribute)
		branchContext, _ = branchContext.UnionBranchContext(branch.Attribute)
		out.WriteString(codegen.AttributeValidationCode(branch.Attribute, nil, branchContext, true, expr.IsAlias(branch.Attribute.Type), "v", element.HTTPName))
		fmt.Fprintf(out, "\nreturn\n}(); err != nil {\nreturn false, nil\n}\ncandidate%s = v\nreturn true, nil\n}}\n", id)
		fmt.Fprintf(out, "branch%s := loomhttp.TextUnionBranch{Schema: leaf%s.Schema, Decode: leaf%s.Decode, Annotations: leaf%s.Annotations}\n", id, id, id, id)
	}
	fmt.Fprintf(out, "branches%s := []loomhttp.TextUnionBranch{\n", prefix)
	for index := range union.Values {
		fmt.Fprintf(out, "branch%s%d,\n", prefix, index)
	}
	out.WriteString("}\n")
}

func renderTextUnionSelection(out *strings.Builder, union *expr.Union, target, path, prefix string) {
	members := unionnames.Resolve(union)
	out.WriteString("switch matched {\n")
	for index, branch := range union.Values {
		id := fmt.Sprintf("%s%d", prefix, index)
		fmt.Fprintf(out, "case %d:\n%s.Set%s(candidate%s)\n", index, target, members[branch.Name].Method, id)
		if expr.AsUnion(branch.Attribute.Type) != nil {
			fmt.Fprintf(out, "%s = append(%s, path%s...)\n", path, path, id)
		}
	}
	out.WriteString("}\n")
}
