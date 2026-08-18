//nolint:errcheck // Generator helpers write only to in-memory builders.
package service

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

type errorMessageField struct {
	Name    string
	Pointer bool
	Alias   bool
}

var errorMessageAttributeNames = []string{"message", "detail", "title", "error", "reason"}

func typeDefinitionSection(name, description, typeName, def string) codegen.Section {
	return codegen.NewJenniferSection(name, func(stmt *jen.Statement) {
		if strings.TrimSpace(description) != "" {
			codegen.Doc(stmt, description)
		} else {
			stmt.Line()
		}
		stmt.Type().Id(typeName).Add(codegen.Expr(def))
		stmt.Line()
	})
}

func payloadSection(method *MethodData) codegen.Section {
	return typeDefinitionSection("service-payload", method.PayloadDesc, method.Payload, method.PayloadDef)
}

func streamingPayloadSection(method *MethodData) codegen.Section {
	return typeDefinitionSection("service-streaming-payload", method.StreamingPayloadDesc, method.StreamingPayload, method.StreamingPayloadDef)
}

func resultSection(name, resultName, resultDesc, resultDef string) codegen.Section {
	return typeDefinitionSection(name, resultDesc, resultName, resultDef)
}

func userTypeSection(name string, data *UserTypeData) codegen.Section {
	return typeDefinitionSection(name, data.Description, data.VarName, data.Def)
}

func errorSection(data *UserTypeData) codegen.Section {
	return codegen.NewJenniferSection("service-error", func(stmt *jen.Statement) {
		stmt.Add(codegen.Expr(strings.TrimSpace(renderErrorMethods(data))))
		stmt.Line()
	})
}

func validateSection(name string, data *ValidateData) codegen.Section {
	return codegen.NewJenniferSection(name, func(stmt *jen.Statement) {
		codegen.Doc(stmt, data.Description)
		stmt.Func().
			Id(data.Name).
			Params(jen.Id("result").Add(codegen.TypeRef(data.Ref))).
			Params(jen.Id("err").Error()).
			BlockFunc(func(group *jen.Group) {
				if data.Validate != "" {
					group.Add(codegen.Expr(strings.TrimRight(codegen.Indent(data.Validate, "\t"), "\n")))
				} else {
					group.Line()
				}
				group.Return()
			})
		stmt.Line()
	})
}

func viewedTypeMapSection(rtdata []*viewedType) codegen.Section {
	return codegen.NewJenniferSection("viewed-type-map", func(stmt *jen.Statement) {
		stmt.Add(codegen.Expr(strings.TrimSpace(renderViewedTypeMap(rtdata))))
		stmt.Line()
	})
}

func unionTypeSection(name string, data *UnionTypeData) codegen.Section {
	return codegen.NewJenniferSection(name, func(stmt *jen.Statement) {
		addUnionTypeSection(stmt, data)
	})
}

func renderErrorMethods(data *UserTypeData) string {
	var b sourceBuilder
	b.Add("// Error returns an error description.\n")
	fmt.Fprintf(&b, "func (e %s) Error() string {\n", data.Ref)
	for _, field := range errorMessageFields(data) {
		accessor := "e." + field.Name
		value := accessor
		if field.Pointer {
			value = "*" + accessor
		}
		if field.Alias {
			value = "string(" + value + ")"
		}
		condition := value + " != \"\""
		if field.Pointer {
			condition = accessor + " != nil && " + condition
		}
		if strings.HasPrefix(data.Ref, "*") {
			condition = "e != nil && " + condition
		}
		fmt.Fprintf(&b, "\tif %s {\n\t\treturn %s\n\t}\n", condition, value)
	}
	fmt.Fprintf(&b, "\treturn %q\n}\n\n", data.Description)
	b.Add("// LoomErrorName returns the error name.\n")
	fmt.Fprintf(&b, "func (e %s) LoomErrorName() string {\n\treturn %s\n}\n", data.Ref, errorName(data))
	if data.RemedyCode != "" || data.SafeMessage != "" || data.RetryHint != "" {
		b.Add("\n// LoomErrorRemedy returns the remediation guidance for the error.\n")
		fmt.Fprintf(&b, "func (e %s) LoomErrorRemedy() *loom.ErrorRemedy {\n", data.Ref)
		b.Add("\treturn &loom.ErrorRemedy{\n")
		fmt.Fprintf(&b, "\t\tCode:        %q,\n", data.RemedyCode)
		fmt.Fprintf(&b, "\t\tSafeMessage: %q,\n", data.SafeMessage)
		fmt.Fprintf(&b, "\t\tRetryHint:   %q,\n", data.RetryHint)
		b.Add("\t}\n}\n")
	}
	return b.String()
}

func errorMessageFields(data *UserTypeData) []errorMessageField {
	object := expr.AsObject(data.Type)
	if object == nil {
		return nil
	}
	parent := data.Type.Attribute()
	fields := make([]errorMessageField, 0, len(errorMessageAttributeNames))
	for _, preferredName := range errorMessageAttributeNames {
		for _, attribute := range *object {
			if _, isErrorName := attribute.Attribute.Meta["struct:error:name"]; isErrorName {
				continue
			}
			if !strings.EqualFold(attribute.Name, preferredName) ||
				!isStringType(attribute.Attribute.Type) {
				continue
			}
			fields = append(fields, errorMessageField{
				Name:    codegen.GoifyAtt(attribute.Attribute, attribute.Name, true),
				Pointer: parent.IsPrimitivePointer(attribute.Name, true),
				Alias:   expr.IsAlias(attribute.Attribute.Type),
			})
		}
	}
	return fields
}

func isStringType(dataType expr.DataType) bool {
	switch actual := dataType.(type) {
	case expr.Primitive:
		return actual.Kind() == expr.StringKind
	case expr.UserType:
		return isStringType(actual.Attribute().Type)
	default:
		return false
	}
}

func renderViewedTypeMap(rtdata []*viewedType) string {
	var b sourceBuilder
	b.Add("var (\n")
	for _, vt := range rtdata {
		b.Add(codegen.Indent(codegen.Comment(fmt.Sprintf("%sMap is a map indexing the attribute names of %s by view name.", vt.Name, vt.Name)), "\t"))
		b.Add("\n")
		fmt.Fprintf(&b, "\t%sMap = map[string][]string{\n", vt.Name)
		for _, view := range vt.Views {
			fmt.Fprintf(&b, "\t\t%q: {\n", view.Name)
			for _, attr := range view.Attributes {
				fmt.Fprintf(&b, "\t\t\t%q,\n", attr)
			}
			b.Add("\t\t},\n")
		}
		b.Add("\t}\n")
	}
	b.Add(")\n")
	return b.String()
}

func addUnionTypeSection(stmt *jen.Statement, data *UnionTypeData) {
	addUnionAliasTypes(stmt, data)
	addUnionStructType(stmt, data)
	addUnionKindType(stmt, data)
	addUnionKindConsts(stmt, data)
	addUnionKindMethod(stmt, data)
	addUnionVariantMethods(stmt, data)
	addUnionValidateMethod(stmt, data)
	addUnionMarshalJSONMethod(stmt, data)
	addUnionMarshalFormMethod(stmt, data)
	addUnionUnmarshalFormMethod(stmt, data)
	addUnionUnmarshalJSONMethod(stmt, data)
}

func addUnionAliasTypes(stmt *jen.Statement, data *UnionTypeData) {
	for _, field := range data.Fields {
		if !field.EmitPrimitiveAlias {
			continue
		}
		stmt.Type().Id(field.FieldType).Add(codegen.Expr(field.PrimitiveAliasType))
		stmt.Line()
	}
}

func addUnionStructType(stmt *jen.Statement, data *UnionTypeData) {
	codegen.Doc(stmt, fmt.Sprintf("%s is a sum-type union.", data.Name))
	stmt.Type().Id(data.Name).StructFunc(func(group *jen.Group) {
		group.Id("kind").Id(data.KindName)
		for _, field := range data.Fields {
			group.Id(field.FieldName).Id(field.FieldType)
		}
	})
	stmt.Line()
}

func addUnionKindType(stmt *jen.Statement, data *UnionTypeData) {
	codegen.Doc(stmt, fmt.Sprintf("%s enumerates the union variants for %s.", data.KindName, data.Name))
	stmt.Type().Id(data.KindName).String()
	stmt.Line()
}

func addUnionKindConsts(stmt *jen.Statement, data *UnionTypeData) {
	stmt.Const().DefsFunc(func(group *jen.Group) {
		for _, field := range data.Fields {
			group.Comment(fmt.Sprintf("%s identifies the %s branch of the union.", field.KindConst, field.Name))
			group.Id(field.KindConst).Id(data.KindName).Op("=").Lit(field.TypeTag)
		}
	})
	stmt.Line()
}

func addUnionKindMethod(stmt *jen.Statement, data *UnionTypeData) {
	codegen.Doc(stmt, "Kind returns the discriminator value of the union.")
	stmt.Func().
		Params(jen.Id("u").Id(data.Name)).
		Id("Kind").
		Params().
		Id(data.KindName).
		Block(
			jen.Return(jen.Id("u").Dot("kind")),
		)
	stmt.Line()
}

func addUnionVariantMethods(stmt *jen.Statement, data *UnionTypeData) {
	for _, field := range data.Fields {
		codegen.Doc(stmt, fmt.Sprintf("New%s%s constructs a %s with the %s branch set.", data.Name, field.FieldName, data.Name, field.Name))
		stmt.Func().
			Id("New" + data.Name + field.FieldName).
			Params(jen.Id("v").Id(field.FieldType)).
			Id(data.Name).
			BlockFunc(func(group *jen.Group) {
				group.Return(
					jen.Id(data.Name).CustomFunc(jen.Options{
						Open:      "{",
						Close:     "}",
						Separator: ",",
						Multi:     true,
					}, func(values *jen.Group) {
						values.Id("kind").Op(":").Id(field.KindConst)
						values.Id(field.FieldName).Op(":").Id("v")
					}),
				)
			})
		stmt.Line()

		codegen.Doc(stmt, fmt.Sprintf("As%s returns the value of the %s branch if set.", field.FieldName, field.Name))
		stmt.Func().
			Params(jen.Id("u").Id(data.Name)).
			Id("As"+field.FieldName).
			Params().
			Params(jen.Id("_").Id(field.FieldType), jen.Id("ok").Bool()).
			Block(
				jen.If(jen.Id("u").Dot("kind").Op("!=").Id(field.KindConst)).Block(
					jen.Return(),
				),
				jen.Return(jen.Id("u").Dot(field.FieldName), jen.True()),
			)
		stmt.Line()

		codegen.Doc(stmt, fmt.Sprintf("Set%s sets the %s branch of the union.", field.FieldName, field.Name))
		stmt.Func().
			Params(jen.Id("u").Op("*").Id(data.Name)).
			Id("Set"+field.FieldName).
			Params(jen.Id("v").Id(field.FieldType)).
			Block(
				jen.Id("u").Dot("kind").Op("=").Id(field.KindConst),
				jen.Id("u").Dot(field.FieldName).Op("=").Id("v"),
			)
		stmt.Line()
	}
}

func addUnionValidateMethod(stmt *jen.Statement, data *UnionTypeData) {
	codegen.Doc(stmt, "Validate ensures the union discriminant is valid.")
	stmt.Func().
		Params(jen.Id("u").Id(data.Name)).
		Id("Validate").
		Params().
		Error().
		BlockFunc(func(group *jen.Group) {
			addRawUnionBlock(group, renderUnionValidateBody(data))
		})
	stmt.Line()
}

func addUnionMarshalJSONMethod(stmt *jen.Statement, data *UnionTypeData) {
	description := "MarshalJSON marshals the union into the canonical {type,value} JSON shape."
	if data.Untagged {
		description = "MarshalJSON marshals the selected union branch directly."
	}
	codegen.Doc(stmt, description)
	stmt.Func().
		Params(jen.Id("u").Id(data.Name)).
		Id("MarshalJSON").
		Params().
		Params(jen.Index().Byte(), jen.Error()).
		BlockFunc(func(group *jen.Group) {
			addRawUnionBlock(group, renderUnionMarshalJSONBody(data))
		})
	stmt.Line()
}

func addUnionMarshalFormMethod(stmt *jen.Statement, data *UnionTypeData) {
	addUnionFormMethodComment(stmt, "MarshalFormValues", "marshals")
	stmt.Func().
		Params(jen.Id("u").Id(data.Name)).
		Id("MarshalFormValues").
		Params(jen.Id("values").Qual("net/url", "Values"), jen.Id("prefix").String()).
		Error().
		BlockFunc(func(group *jen.Group) {
			addRawUnionBlock(group, renderUnionMarshalFormBody(data))
		})
	stmt.Line()
}

func addUnionUnmarshalFormMethod(stmt *jen.Statement, data *UnionTypeData) {
	addUnionFormMethodComment(stmt, "UnmarshalFormValues", "unmarshals")
	stmt.Func().
		Params(jen.Id("u").Op("*").Id(data.Name)).
		Id("UnmarshalFormValues").
		Params(jen.Id("values").Qual("net/url", "Values"), jen.Id("prefix").String()).
		Error().
		BlockFunc(func(group *jen.Group) {
			addRawUnionBlock(group, renderUnionUnmarshalFormBody(data))
		})
	stmt.Line()
}

func addUnionUnmarshalJSONMethod(stmt *jen.Statement, data *UnionTypeData) {
	description := "UnmarshalJSON unmarshals the union from the canonical {type,value} JSON shape."
	if data.Untagged {
		description = "UnmarshalJSON selects the single valid untagged union branch."
	}
	stmt.Comment(description).Line()
	stmt.Func().
		Params(jen.Id("u").Op("*").Id(data.Name)).
		Id("UnmarshalJSON").
		Params(jen.Id("data").Index().Byte()).
		Error().
		BlockFunc(func(group *jen.Group) {
			addRawUnionBlock(group, renderUnionUnmarshalJSONBody(data))
		})
	stmt.Line()
}

func renderUnionValidateBody(data *UnionTypeData) string {
	var b sourceBuilder
	b.Add("switch u.kind {\n")
	fmt.Fprintf(&b, "\tcase %q:\n", "")
	fmt.Fprintf(&b, "\t\treturn loom.InvalidEnumValueError(%q, %q, []any{\n", data.TypeKey, "")
	for _, field := range data.Fields {
		fmt.Fprintf(&b, "\t\t\tstring(%s),\n", field.KindConst)
	}
	b.Add("\t\t})\n")
	for _, field := range data.Fields {
		fmt.Fprintf(&b, "\tcase %s:\n\t\treturn nil\n", field.KindConst)
	}
	b.Add("\tdefault:\n")
	fmt.Fprintf(&b, "\t\treturn loom.InvalidEnumValueError(%q, u.kind, []any{\n", data.TypeKey)
	for _, field := range data.Fields {
		fmt.Fprintf(&b, "\t\t\tstring(%s),\n", field.KindConst)
	}
	b.Add("\t\t})\n\t}")
	return b.String()
}

func renderUnionMarshalJSONBody(data *UnionTypeData) string {
	if data.Untagged {
		return renderUntaggedUnionMarshalJSONBody(data)
	}
	var b sourceBuilder
	b.Add("if err := u.Validate(); err != nil {\n\treturn nil, err\n}\n")
	b.Add("var (\n\tvalue any\n)\n")
	b.Add("switch u.kind {\n")
	for _, field := range data.Fields {
		fmt.Fprintf(&b, "\tcase %s:\n\t\tvalue = u.%s\n", field.KindConst, field.FieldName)
	}
	fmt.Fprintf(&b, "\tdefault:\n\t\treturn nil, fmt.Errorf(\"unexpected %s discriminant %%q\", u.kind)\n\t}\n", data.Name)
	fmt.Fprintf(&b, "return json.Marshal(struct {\n\tType  string `json:\"%s\"`\n\tValue any    `json:\"%s\"`\n}{\n", data.TypeKey, data.ValueKey)
	b.Add("\tType:  string(u.kind),\n\tValue: value,\n})")
	return b.String()
}

func renderUntaggedUnionMarshalJSONBody(data *UnionTypeData) string {
	var b sourceBuilder
	b.Add("if err := u.Validate(); err != nil {\n\treturn nil, err\n}\n")
	b.Add("switch u.kind {\n")
	for _, field := range data.Fields {
		fmt.Fprintf(&b, "\tcase %s:\n\t\treturn json.Marshal(u.%s)\n", field.KindConst, field.FieldName)
	}
	fmt.Fprintf(&b, "\tdefault:\n\t\treturn nil, fmt.Errorf(\"unexpected %s discriminant %%q\", u.kind)\n\t}", data.Name)
	return b.String()
}

func renderUnionMarshalFormBody(data *UnionTypeData) string {
	var b sourceBuilder
	b.Add("if err := u.Validate(); err != nil {\n\treturn err\n}\n")
	fmt.Fprintf(&b, "values.Set(loomhttp.FormChildKey(prefix, %q), string(u.kind))\n", data.TypeKey)
	b.Add("switch u.kind {\n")
	for _, field := range data.Fields {
		fmt.Fprintf(&b, "\tcase %s:\n", field.KindConst)
		if field.FlatFormObject {
			fmt.Fprintf(&b, "\t\t_, err := loomhttp.EncodeFormValue(values, prefix, u.%s)\n", field.FieldName)
		} else {
			fmt.Fprintf(&b, "\t\t_, err := loomhttp.EncodeFormValue(values, loomhttp.FormChildKey(prefix, %q), u.%s)\n", data.ValueKey, field.FieldName)
		}
		b.Add("\t\treturn err\n")
	}
	fmt.Fprintf(&b, "\tdefault:\n\t\treturn fmt.Errorf(\"unexpected %s discriminant %%q\", u.kind)\n\t}", data.Name)
	return b.String()
}

func renderUnionUnmarshalFormBody(data *UnionTypeData) string {
	var b sourceBuilder
	fmt.Fprintf(&b, "typeKey := loomhttp.FormChildKey(prefix, %q)\n", data.TypeKey)
	if data.HasScalarFormBranch {
		fmt.Fprintf(&b, "valueKey := loomhttp.FormChildKey(prefix, %q)\n", data.ValueKey)
	}
	b.Add("rawType := values.Get(typeKey)\n")
	b.Add("if rawType == \"\" {\n")
	fmt.Fprintf(&b, "\treturn loom.MissingFieldError(%q, \"body\")\n}\n", data.TypeKey)
	b.Add("switch rawType {\n")
	for _, field := range data.Fields {
		fmt.Fprintf(&b, "\tcase string(%s):\n\t\tvar v %s\n", field.KindConst, field.FieldType)
		if field.FlatFormObject {
			b.Add("\t\tseen, err := loomhttp.DecodeFormValue(values, prefix, &v)\n")
		} else {
			b.Add("\t\tseen, err := loomhttp.DecodeFormValue(values, valueKey, &v)\n")
		}
		b.Add("\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n")
		b.Add("\t\tif !seen {\n")
		if field.FlatFormObjectAllowsEmpty {
			fmt.Fprintf(&b, "\t\t\tv = %s\n", field.EmptyValueExpr)
		} else {
			fmt.Fprintf(&b, "\t\t\treturn loom.MissingFieldError(%q, \"body\")\n", data.ValueKey)
		}
		b.Add("\t\t}\n")
		fmt.Fprintf(&b, "\t\tu.kind = %s\n\t\tu.%s = v\n", field.KindConst, field.FieldName)
	}
	b.Add("\tdefault:\n")
	fmt.Fprintf(&b, "\t\treturn loom.InvalidEnumValueError(%q, rawType, []any{\n", data.TypeKey)
	for _, field := range data.Fields {
		fmt.Fprintf(&b, "\t\t\tstring(%s),\n", field.KindConst)
	}
	b.Add("\t\t})\n\t}\nreturn nil")
	return b.String()
}

func renderUnionUnmarshalJSONBody(data *UnionTypeData) string {
	if data.Untagged {
		return renderUntaggedUnionUnmarshalJSONBody(data)
	}
	var b sourceBuilder
	fmt.Fprintf(&b, "var raw struct {\n\tType  string          `json:\"%s\"`\n\tValue json.RawMessage `json:\"%s\"`\n}\n", data.TypeKey, data.ValueKey)
	b.Add("if err := json.Unmarshal(data, &raw); err != nil {\n\treturn err\n}\n")
	b.Add("switch raw.Type {\n")
	for _, field := range data.Fields {
		fmt.Fprintf(&b, "\tcase string(%s):\n\t\tvar v %s\n", field.KindConst, field.FieldType)
		b.Add("\t\tif len(raw.Value) == 0 || string(raw.Value) == \"null\" {\n")
		fmt.Fprintf(&b, "\t\t\treturn loom.MissingFieldError(%q, \"body\")\n\t\t}\n", data.ValueKey)
		b.Add("\t\tif err := json.Unmarshal(raw.Value, &v); err != nil {\n\t\t\treturn err\n\t\t}\n")
		fmt.Fprintf(&b, "\t\tu.kind = %s\n\t\tu.%s = v\n", field.KindConst, field.FieldName)
	}
	b.Add("\tdefault:\n")
	fmt.Fprintf(&b, "\t\treturn loom.InvalidEnumValueError(%q, raw.Type, []any{\n", data.TypeKey)
	for _, field := range data.Fields {
		fmt.Fprintf(&b, "\t\t\tstring(%s),\n", field.KindConst)
	}
	b.Add("\t\t})\n\t}\nreturn nil")
	return b.String()
}

func renderUntaggedUnionUnmarshalJSONBody(data *UnionTypeData) string {
	var b sourceBuilder
	b.Add("var rawObject map[string]json.RawMessage\n")
	b.Add("if err := json.Unmarshal(data, &rawObject); err != nil {\n\treturn err\n}\n")
	fmt.Fprintf(&b, "if rawObject == nil {\n\treturn fmt.Errorf(\"decode %s: untagged union matched 0 branches\")\n}\n", data.Name)
	b.Add("matches := 0\n")
	fmt.Fprintf(&b, "var matched %s\n", data.Name)
	for _, field := range data.Fields {
		fmt.Fprintf(&b, "{\n\teligible := true\n\tvar v %s\n", field.FieldType)
		for _, name := range field.RequiredFields {
			fmt.Fprintf(&b, "\tif _, ok := rawObject[%q]; !ok {\n\t\teligible = false\n\t}\n", name)
		}
		for _, name := range field.NonNullableFields {
			fmt.Fprintf(&b, "\tif value, ok := rawObject[%q]; ok {\n\t\tvar decoded any\n\t\tif err := json.Unmarshal(value, &decoded); err != nil || decoded == nil {\n\t\t\teligible = false\n\t\t}\n\t}\n", name)
		}
		if field.RejectUnknownJSONFields {
			b.Add("\tfor name := range rawObject {\n\t\tswitch name {\n")
			for _, name := range field.JSONFields {
				fmt.Fprintf(&b, "\t\tcase %q:\n", name)
			}
			b.Add("\t\tdefault:\n\t\t\teligible = false\n\t\t}\n\t}\n")
		}
		b.Add("\tfiltered := make(map[string]json.RawMessage)\n")
		for _, name := range field.JSONFields {
			fmt.Fprintf(&b, "\tif value, ok := rawObject[%q]; ok {\n\t\tfiltered[%q] = value\n\t}\n", name, name)
		}
		b.Add("\tif eligible {\n")
		b.Add("\tcandidateData, marshalErr := json.Marshal(filtered)\n")
		b.Add("\tif marshalErr == nil {\n")
		b.Add("\tif err := json.Unmarshal(candidateData, &v); err == nil {\n")
		validated := false
		if field.ValidateCode != "" {
			b.Add("\t\tbranchErr := func() (err error) {\n")
			b.Add(codegen.Indent(field.ValidateCode, "\t\t\t"))
			b.Add("\n")
			b.Add("\t\t\treturn\n\t\t}()\n")
			b.Add("\t\tif branchErr == nil {\n")
			validated = true
		} else if field.ValidateRef != "" {
			fmt.Fprintf(&b, "\t\tif err := %s; err == nil {\n", field.ValidateRef)
			validated = true
		}
		indent := "\t\t"
		if validated {
			indent = "\t\t\t"
		}
		fmt.Fprintf(&b, "%smatches++\n%smatched.kind = %s\n%smatched.%s = v\n", indent, indent, field.KindConst, indent, field.FieldName)
		if validated {
			b.Add("\t\t}\n")
		}
		b.Add("\t}\n\t}\n\t}\n}\n")
	}
	fmt.Fprintf(&b, "if matches != 1 {\n\treturn fmt.Errorf(\"decode %s: untagged union matched %%d branches\", matches)\n}\n", data.Name)
	b.Add("*u = matched\nreturn nil")
	return b.String()
}

func addRawUnionBlock(group *jen.Group, code string) {
	if strings.TrimSpace(code) == "" {
		return
	}
	group.Add(codegen.Expr(strings.TrimRight(code, "\n")))
}

func addUnionFormMethodComment(stmt *jen.Statement, methodName, verb string) {
	preposition := "into"
	if verb == "unmarshals" {
		preposition = "from"
	}
	stmt.Comment(methodName + " " + verb + " the union " + preposition + " application/x-www-form-urlencoded").Line()
	stmt.Comment("values using the discriminator field plus flattened object fields for").Line()
	stmt.Comment("object-shaped branches and the canonical {type,value} form shape for scalar").Line()
	stmt.Comment("branches.").Line()
}
