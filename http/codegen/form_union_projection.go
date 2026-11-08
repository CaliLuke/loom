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

func renderFormRootProjection(out *strings.Builder, body *TypeData, contract *expr.HTTPFormEncoding, id string) {
	shape, err := representation.FormJSONShape(body.attribute, contract, body.context)
	if err != nil {
		panic(fmt.Errorf("form root schema: %w", err))
	}
	unionjson.RenderShape(out, "formSchema"+id, shape)
	fmt.Fprintf(out, "projection%s, projected%s, projectionError%s := ", id, id, id)
	if mapping := expr.AsMap(body.attribute.Type); mapping != nil {
		out.WriteString("loomhttp.FormMapJSON(parameterFormPairs, func(formKey string) (jsontext.Value, bool, error) {\n")
		renderFormFieldProjection(out, mapping.ElemType, *contract.Element, nil, "formKey", body.context)
		out.WriteString("})\n")
		return
	}
	styled, ordinary := formFieldData(body.attribute, contract.Fields)
	byName := make(map[string]*FormFieldData, len(styled)+len(ordinary))
	for _, field := range append(styled, ordinary...) {
		byName[field.AttributeName] = field
	}
	out.WriteString("loomhttp.FormObjectJSON(parameterFormPairs, []loomhttp.FormField{\n")
	for _, field := range *expr.AsObject(body.attribute.Type) {
		captured := contract.Fields[field.Name]
		if captured.Omitted {
			continue
		}
		data := byName[expr.AttributeName(field.Name)]
		fmt.Fprintf(out, "{Name: %q, Keys: %s, View: func() (jsontext.Value, bool, error) {\n", captured.Name, formKeySetLiteral(captured.Keys))
		renderFormFieldProjection(out, field.Attribute, captured.Codec, data, strconv.Quote(captured.Name), body.context)
		out.WriteString("}},\n")
	}
	out.WriteString("})\n")
}

func renderFormFieldProjection(out *strings.Builder, attribute *expr.AttributeExpr, codec httpencoding.Codec, data *FormFieldData, name string, context *codegen.AttributeContext) {
	if codec.Style == httpencoding.JSON {
		fmt.Fprintf(out, "return loomhttp.FormJSONValue(parameterFormPairs, %s, %s)\n", name, formgen.Application(codec.Application))
		return
	}
	element := &Element{attribute: attribute, Codec: codec, context: context}
	if data != nil {
		element.HTTPName, element.Namespace = data.HTTPName, data.Namespace
	}
	if union := expr.AsUnion(attribute.Type); union != nil {
		formValueUnion(out, attribute, element, union, codec.Branches, name)
		return
	}
	fmt.Fprintf(out, "return loomhttp.FormTextValue(loomhttp.TextBranch{View: func() (loomhttp.ParamValue, error) {\nreturn loomhttp.DecodePairsParam(parameterFormPairs, %s, %s, %s, %s)\n}, Shape: %s})\n", name, parameterCodecLiteral(codec), parameterKindLiteral(codec), parameterNamespaceLiteral(element), textUnionShape(attribute))
}

func formValueUnion(out *strings.Builder, attribute *expr.AttributeExpr, element *Element, union *expr.Union, codecs []httpencoding.BranchCodec, name string) {
	for index, branch := range union.Values {
		shape, err := representation.FormFieldJSONShape(branch.Attribute, codecs[index].Codec, element.context)
		if err != nil {
			panic(err)
		}
		unionjson.RenderShape(out, fmt.Sprintf("fieldSchema%d", index), shape)
	}
	fmt.Fprintf(out, "return loomhttp.FormTextUnionValue(%s, []loomhttp.FormValueBranch{\n", name)
	for index, branch := range union.Values {
		branchElement := textUnionBranchElement(element, codecs[index])
		out.WriteString("{View: func() (jsontext.Value, bool, error) {\n")
		if branchElement.Codec.Style == httpencoding.JSON {
			fmt.Fprintf(out, "return loomhttp.FormJSONValue(parameterFormPairs, %s, %s)\n", name, formgen.Application(branchElement.Codec.Application))
		} else if nested := expr.AsUnion(branch.Attribute.Type); nested != nil {
			formValueUnion(out, branch.Attribute, branchElement, nested, codecs[index].Inner, name)
		} else {
			fmt.Fprintf(out, "return loomhttp.FormTextValue(loomhttp.TextBranch{View: func() (loomhttp.ParamValue, error) {\nreturn loomhttp.DecodePairsParam(parameterFormPairs, %s, %s, %s, %s)\n}, Shape: %s})\n", name, parameterCodecLiteral(branchElement.Codec), parameterKindLiteral(branchElement.Codec), parameterNamespaceLiteral(branchElement), textUnionShape(branch.Attribute))
		}
		fmt.Fprintf(out, "}, Member: fieldSchema%d.Match, Evaluate: fieldSchema%d.Evaluated},\n", index, index)
	}
	fmt.Fprintf(out, "}, %t)\n", attribute.UnionClosed())
}

func formKeySetLiteral(keys httpencoding.KeySet) string {
	quoted := func(names []string) string {
		values := make([]string, len(names))
		for index, name := range names {
			values[index] = strconv.Quote(name)
		}
		return "[]string{" + strings.Join(values, ",") + "}"
	}
	return fmt.Sprintf("loomhttp.FormKeySet{Literals: %s, Prefixes: %s, Residual: %t}", quoted(keys.Literals), quoted(keys.Prefixes), keys.Residual)
}
