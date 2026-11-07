package codegen

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/codegen/unionnames"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

func parameterWireName(element *Element) string {
	if element.wireNameExpr != "" {
		return element.wireNameExpr
	}
	return strconv.Quote(element.HTTPName)
}

func textUnionBranchElement(element *Element, branch httpencoding.BranchCodec) *Element {
	result := *element
	result.Codec = branch.Codec
	if branch.Codec.Shape == httpencoding.Object && (branch.Codec.Style == httpencoding.DeepObject || branch.Codec.Explode && (branch.Codec.Style == httpencoding.Form || branch.Codec.Style == httpencoding.CookieStyle)) {
		result.Namespace.OwnedLiterals = append([]string{}, branch.Keys.Literals...)
	}
	return &result
}

func textUnionView(element *Element, location string) string {
	codec, kind, namespace := parameterCodecLiteral(element.Codec), parameterKindLiteral(element.Codec), parameterNamespaceLiteral(element)
	switch location {
	case "query", "form":
		return fmt.Sprintf("loomhttp.DecodePairsParam(parameterPairs, %s, %s, %s, %s)", parameterWireName(element), codec, kind, namespace)
	case "header", "response-header":
		return fmt.Sprintf("loomhttp.DecodeHeaderParam(%q, parameterHeaders.Values(%q), len(parameterHeaders.Values(%q)) > 0, %s, %s)", element.HTTPName, element.HTTPName, element.HTTPName, codec, kind)
	case "path":
		return fmt.Sprintf("loomhttp.DecodePathParam(%q, parameterPath, %s, %s)", element.HTTPName, codec, kind)
	case "cookie":
		return fmt.Sprintf("loomhttp.DecodeCookieParam(parameterCookies, %q, %s, %s, %s)", element.HTTPName, codec, kind, namespace)
	default:
		panic("unsupported text union location " + location)
	}
}

func textUnionShape(attribute *expr.AttributeExpr) string {
	base, _ := transportUnderlyingType(attribute.Type)
	switch actual := base.(type) {
	case *expr.Array:
		return "loomhttp.ParamBranchShape{Kind: loomhttp.ParamList, Element: " + textUnionScalar(actual.ElemType) + "}"
	case *expr.Map:
		return "loomhttp.ParamBranchShape{Kind: loomhttp.ParamPairs, MapValue: " + textUnionScalar(actual.ElemType) + "}"
	case *expr.Object:
		var out strings.Builder
		out.WriteString("loomhttp.ParamBranchShape{Kind: loomhttp.ParamPairs, Properties: map[string]loomhttp.ParamScalarKind{\n")
		for _, field := range *actual {
			fmt.Fprintf(&out, "%q: %s,\n", expr.ElementName(field.Name), textUnionScalar(field.Attribute))
		}
		out.WriteString("}}")
		return out.String()
	default:
		return "loomhttp.ParamBranchShape{Kind: loomhttp.ParamScalar, Element: " + textUnionScalar(attribute) + "}"
	}
}

func textUnionScalar(attribute *expr.AttributeExpr) string {
	base, _ := transportUnderlyingType(attribute.Type)
	switch base.Kind() {
	case expr.BooleanKind:
		return "loomhttp.ParamBoolean"
	case expr.IntKind, expr.Int32Kind, expr.Int64Kind, expr.UIntKind, expr.UInt32Kind, expr.UInt64Kind:
		return "loomhttp.ParamInteger"
	case expr.Float32Kind, expr.Float64Kind:
		return "loomhttp.ParamNumber"
	case expr.BytesKind:
		return "loomhttp.ParamBytes"
	case expr.StringKind:
		return "loomhttp.ParamString"
	default:
		panic("unsupported text union scalar " + base.Name())
	}
}

func renderTextUnionDecode(element *Element, location string) string {
	var out strings.Builder
	switch location {
	case "header":
		out.WriteString("parameterHeaders := r.Header\n")
	case "response-header":
		out.WriteString("parameterHeaders := resp.Header\n")
	case "cookie":
		out.WriteString("parameterCookies := r.Cookies()\n")
	case "path":
		fmt.Fprintf(&out, "parameterPath := mux.RawVars(r)[%q]\n", element.HTTPName)
	}
	out.WriteString(renderTextUnionMatcher(element, location))
	out.WriteString("matchedValue, present, _, matchError := matchTextUnion()\nif matchError != nil {\nreturn loom.DecodePayloadError(matchError.Error())\n}\nif present {\n")
	fmt.Fprintf(&out, "%s = matchedValue\n", element.VarName)
	if element.Required {
		fmt.Fprintf(&out, "} else {\nreturn loom.MissingFieldError(%q, %q)\n", element.Name, location)
	}
	out.WriteString("}\nreturn nil\n")
	return out.String()
}

func renderTextUnionEncode(element *Element, source, location string) string {
	var out strings.Builder
	union := expr.AsUnion(element.Type)
	codecs := element.Codec.Branches
	out.WriteString("{\n")
	guard := element.FieldPointer || element.FieldName == "" && element.Pointer
	if guard {
		fmt.Fprintf(&out, "if %s != nil {\n", parameterServiceValue(element, source))
	}
	out.WriteString("var selected []int\nvar parameterValue loomhttp.ParamValue\nvar parameterCodec loomhttp.ParamCodec\n")
	withNamespace := location == "query" || location == "form" || location == "cookie"
	if withNamespace {
		out.WriteString("var parameterNamespace loomhttp.ParamNamespace\n")
	}
	renderTextUnionEncodingChoice(&out, element, union, codecs, parameterServiceValue(element, source), location, nil)
	out.WriteString("if len(selected) == 0 {\nreturn fmt.Errorf(\"text union has no selected branch\")\n}\n")
	switch location {
	case "query", "form":
		fmt.Fprintf(&out, "var localBuilder loomhttp.RawPairsBuilder\nif err := localBuilder.Add(%s, parameterCodec, parameterValue, parameterNamespace); err != nil {\nreturn err\n}\nparameterPairs, err := loomhttp.ParseRawPairs(localBuilder.Encode(nil))\nif err != nil {\nreturn err\n}\n", parameterWireName(element))
	case "header", "response-header":
		fmt.Fprintf(&out, "encoded, err := loomhttp.EncodeHeaderParam(%q, parameterCodec, parameterValue)\nif err != nil {\nreturn err\n}\nparameterHeaders := make(http.Header)\nparameterHeaders.Set(%q, encoded)\n", element.HTTPName, element.HTTPName)
	case "cookie":
		fmt.Fprintf(&out, "localRequest := &http.Request{Header: make(http.Header)}\nif err := loomhttp.AddRequestCookieParam(localRequest, %q, parameterCodec, parameterValue, parameterNamespace); err != nil {\nreturn err\n}\nparameterCookies := localRequest.Cookies()\n", element.HTTPName)
	case "path":
		fmt.Fprintf(&out, "parameterPath, err := loomhttp.EncodePathParam(%q, parameterCodec, parameterValue)\nif err != nil {\nreturn err\n}\n", element.HTTPName)
	default:
		panic("unsupported text union encoder " + location)
	}
	out.WriteString(renderTextUnionMatcher(element, location))
	fmt.Fprintf(&out, "_, present, matched, matchError := matchTextUnion()\nif matchError != nil {\nreturn matchError\n}\nif !present && %t {\nreturn fmt.Errorf(\"required text union is absent\")\n}\nif present {\nif len(matched) != len(selected) {\nreturn fmt.Errorf(\"text union changed selected branch\")\n}\nfor index := range selected {\nif selected[index] != matched[index] {\nreturn fmt.Errorf(\"text union changed selected branch\")\n}\n}\n}\n", element.Required)
	switch location {
	case "query", "form":
		fmt.Fprintf(&out, "if err := parameterBuilder.Add(%s, parameterCodec, parameterValue, parameterNamespace); err != nil {\nreturn err\n}\n", parameterWireName(element))
	case "header":
		fmt.Fprintf(&out, "req.Header.Set(%q, encoded)\n", element.HTTPName)
	case "response-header":
		fmt.Fprintf(&out, "w.Header().Set(%q, encoded)\n", element.HTTPName)
	case "cookie":
		out.WriteString("for _, cookie := range parameterCookies {\nreq.AddCookie(cookie)\n}\n")
	case "path":
		out.WriteString("textUnionPath = parameterPath\n")
	}
	if guard {
		if element.Required {
			out.WriteString("} else {\nreturn fmt.Errorf(\"required text union is absent\")\n")
		}
		out.WriteString("}\n")
	}
	out.WriteString("}\n")
	return out.String()
}

func renderTextUnionEncodingChoice(out *strings.Builder, element *Element, union *expr.Union, codecs []httpencoding.BranchCodec, source, location string, ancestors []string) {
	members := unionnames.Resolve(union)
	for index, branch := range union.Values {
		branchElement := textUnionBranchElement(element, codecs[index])
		path := append(append([]string(nil), ancestors...), fmt.Sprint(index))
		fmt.Fprintf(out, "if v, ok := %s.As%s(); ok {\n", source, members[branch.Name].Method)
		if nested := expr.AsUnion(branch.Attribute.Type); nested != nil {
			ref := element.context.Scope.Ref(branch.Attribute, element.context.Pkg(branch.Attribute))
			if strings.HasPrefix(ref, "*") {
				out.WriteString("if v == nil {\nreturn fmt.Errorf(\"nested text union is absent\")\n}\n")
			}
			renderTextUnionEncodingChoice(out, branchElement, nested, codecs[index].Inner, "v", location, path)
		} else {
			fmt.Fprintf(out, "selected = []int{%s}\n", strings.Join(path, ", "))
			out.WriteString(renderParameterValue(branch.Attribute, branch.Attribute, element.context, "v", "branchValue", false))
			fmt.Fprintf(out, "parameterValue = branchValue\nparameterCodec = %s\n", parameterCodecLiteral(branchElement.Codec))
			if location == "query" || location == "form" || location == "cookie" {
				fmt.Fprintf(out, "parameterNamespace = %s\n", parameterNamespaceLiteral(branchElement))
			}
		}
		out.WriteString("}\n")
	}
}

func renderTextUnionPath(arg *InitArgData, segment string) string {
	element := *arg.parameter
	attribute := *element.AttributeData
	attribute.FieldName = ""
	element.AttributeData = &attribute
	var out strings.Builder
	fmt.Fprintf(&out, "%s, parameterError := func() (string, error) {\nvar textUnionPath string\nif err := func() error {\n", segment)
	out.WriteString(renderTextUnionEncode(&element, arg.VarName, "path"))
	out.WriteString("return nil\n}(); err != nil {\nreturn \"\", err\n}\nreturn textUnionPath, nil\n}()\nif parameterError != nil {\nreturn \"\", parameterError\n}\n")
	return out.String()
}
