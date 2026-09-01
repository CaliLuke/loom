package codegen

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	servicecodegen "github.com/CaliLuke/loom/codegen/service"
)

func typeDeclSection(name string, data *TypeData) codegen.Section {
	return codegen.NewJenniferSection(name, func(stmt *jen.Statement) {
		addHTTPDocOrBlank(stmt, data.Description)
		stmt.Type().Id(data.VarName).Add(codegen.Expr(data.Def))
		if data.FlatFormUnionField == "" {
			stmt.Line()
			return
		}

		stmt.Line()
		codegen.CommentBlock(stmt, "MarshalFormValues marshals the synthetic request body wrapper using the wrapped union field at the top level.")
		marshal := stmt.Func().
			Params(jen.Id("body").Id(data.VarName)).
			Id("MarshalFormValues").
			Params(jen.Id("values").Qual("net/url", "Values"), jen.Id("prefix").String()).
			Params(jen.Error())
		if data.FlatFormUnionPointer {
			marshal.Block(
				jen.If(jen.Id("body").Dot(data.FlatFormUnionField).Op("==").Nil()).Block(jen.Return(jen.Nil())),
				jen.Return(jen.Id("body").Dot(data.FlatFormUnionField).Dot("MarshalFormValues").Call(jen.Id("values"), jen.Id("prefix"))),
			)
		} else {
			marshal.Block(
				jen.Return(jen.Id("body").Dot(data.FlatFormUnionField).Dot("MarshalFormValues").Call(jen.Id("values"), jen.Id("prefix"))),
			)
		}

		stmt.Line()
		codegen.CommentBlock(stmt, "UnmarshalFormValues unmarshals the synthetic request body wrapper using the wrapped union field at the top level.")
		unmarshal := stmt.Func().
			Params(jen.Id("body").Op("*").Id(data.VarName)).
			Id("UnmarshalFormValues").
			Params(jen.Id("values").Qual("net/url", "Values"), jen.Id("prefix").String()).
			Params(jen.Error())
		if data.FlatFormUnionPointer {
			unmarshal.Block(
				jen.If(
					jen.Id("values").Dot("Get").Call(
						jen.Id("loomhttp").Dot("FormChildKey").Call(jen.Id("prefix"), jen.Lit(data.FlatFormUnionTypeKey)),
					).Op("==").Lit(""),
				).Block(jen.Return(jen.Nil())),
				jen.Var().Id("value").Add(codegen.Expr(strings.TrimPrefix(data.FlatFormUnionRef, "*"))),
				jen.If(jen.Err().Op(":=").Id("value").Dot("UnmarshalFormValues").Call(jen.Id("values"), jen.Id("prefix")), jen.Err().Op("!=").Nil()).Block(
					jen.Return(jen.Err()),
				),
				jen.Id("body").Dot(data.FlatFormUnionField).Op("=").Op("&").Id("value"),
				jen.Return(jen.Nil()),
			)
		} else {
			unmarshal.Block(
				jen.Return(
					jen.Op("(&").Id("body").Dot(data.FlatFormUnionField).Op(")").
						Dot("UnmarshalFormValues").
						Call(jen.Id("values"), jen.Id("prefix")),
				),
			)
		}
		stmt.Line()
	})
}

func unionTypeSection(name string, data *servicecodegen.UnionTypeData) codegen.Section {
	return codegen.NewJenniferSection(name, func(stmt *jen.Statement) {
		addHTTPUnionTypeSection(stmt, data)
	})
}

func addHTTPUnionTypeSection(stmt *jen.Statement, data *servicecodegen.UnionTypeData) {
	addHTTPUnionAliasTypes(stmt, data)
	addHTTPUnionStructType(stmt, data)
	addHTTPUnionKindType(stmt, data)
	addHTTPUnionKindConsts(stmt, data)
	addHTTPUnionKindMethod(stmt, data)
	addHTTPUnionVariantMethods(stmt, data)
	addHTTPUnionValidateMethod(stmt, data)
	addHTTPUnionMarshalJSONMethod(stmt, data)
	addHTTPUnionMarshalFormMethod(stmt, data)
	addHTTPUnionUnmarshalFormMethod(stmt, data)
	addHTTPUnionUnmarshalJSONMethod(stmt, data)
}

func addHTTPUnionAliasTypes(stmt *jen.Statement, data *servicecodegen.UnionTypeData) {
	for _, field := range data.Fields {
		if !field.EmitPrimitiveAlias {
			continue
		}
		stmt.Type().Id(field.FieldType).Add(codegen.Expr(field.PrimitiveAliasType))
		stmt.Line()
	}
}

func addHTTPUnionStructType(stmt *jen.Statement, data *servicecodegen.UnionTypeData) {
	codegen.Doc(stmt, fmt.Sprintf("%s is a sum-type union.", data.Name))
	stmt.Type().Id(data.Name).StructFunc(func(group *jen.Group) {
		group.Id("kind").Id(data.KindName)
		for _, field := range data.Fields {
			group.Id(field.FieldName).Id(field.FieldType)
		}
	})
	stmt.Line()
}

func addHTTPUnionKindType(stmt *jen.Statement, data *servicecodegen.UnionTypeData) {
	codegen.Doc(stmt, fmt.Sprintf("%s enumerates the union variants for %s.", data.KindName, data.Name))
	stmt.Type().Id(data.KindName).String()
	stmt.Line()
}

func addHTTPUnionKindConsts(stmt *jen.Statement, data *servicecodegen.UnionTypeData) {
	stmt.Const().DefsFunc(func(group *jen.Group) {
		for _, field := range data.Fields {
			group.Comment(fmt.Sprintf("%s identifies the %s branch of the union.", field.KindConst, field.Name))
			group.Id(field.KindConst).Id(data.KindName).Op("=").Lit(field.TypeTag)
		}
	})
	stmt.Line()
}

func addHTTPUnionKindMethod(stmt *jen.Statement, data *servicecodegen.UnionTypeData) {
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

func addHTTPUnionVariantMethods(stmt *jen.Statement, data *servicecodegen.UnionTypeData) {
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

func addHTTPUnionValidateMethod(stmt *jen.Statement, data *servicecodegen.UnionTypeData) {
	codegen.Doc(stmt, "Validate ensures the union discriminant is valid.")
	stmt.Func().
		Params(jen.Id("u").Id(data.Name)).
		Id("Validate").
		Params().
		Error().
		BlockFunc(func(group *jen.Group) {
			addRawHTTPUnionBlock(group, renderHTTPUnionValidateBody(data))
		})
	stmt.Line()
}

func addHTTPUnionMarshalJSONMethod(stmt *jen.Statement, data *servicecodegen.UnionTypeData) {
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
			addRawHTTPUnionBlock(group, renderHTTPUnionMarshalJSONBody(data))
		})
	stmt.Line()
}

func addHTTPUnionMarshalFormMethod(stmt *jen.Statement, data *servicecodegen.UnionTypeData) {
	addHTTPUnionFormMethodComment(stmt, "MarshalFormValues", "marshals")
	stmt.Func().
		Params(jen.Id("u").Id(data.Name)).
		Id("MarshalFormValues").
		Params(jen.Id("values").Qual("net/url", "Values"), jen.Id("prefix").String()).
		Error().
		BlockFunc(func(group *jen.Group) {
			addRawHTTPUnionBlock(group, renderHTTPUnionMarshalFormBody(data))
		})
	stmt.Line()
}

func addHTTPUnionUnmarshalFormMethod(stmt *jen.Statement, data *servicecodegen.UnionTypeData) {
	addHTTPUnionFormMethodComment(stmt, "UnmarshalFormValues", "unmarshals")
	stmt.Func().
		Params(jen.Id("u").Op("*").Id(data.Name)).
		Id("UnmarshalFormValues").
		Params(jen.Id("values").Qual("net/url", "Values"), jen.Id("prefix").String()).
		Error().
		BlockFunc(func(group *jen.Group) {
			addRawHTTPUnionBlock(group, renderHTTPUnionUnmarshalFormBody(data))
		})
	stmt.Line()
}

func addHTTPUnionUnmarshalJSONMethod(stmt *jen.Statement, data *servicecodegen.UnionTypeData) {
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
			addRawHTTPUnionBlock(group, renderHTTPUnionUnmarshalJSONBody(data))
		})
	stmt.Line()
}

func renderHTTPUnionValidateBody(data *servicecodegen.UnionTypeData) string {
	var b sourceBuilder
	b.Add("switch u.kind {\n")
	b.Addf("\tcase %q:\n", "")
	b.Addf("\t\treturn loom.InvalidEnumValueError(%q, %q, []any{\n", data.TypeKey, "")
	for _, field := range data.Fields {
		b.Addf("\t\t\tstring(%s),\n", field.KindConst)
	}
	b.Add("\t\t})\n")
	for _, field := range data.Fields {
		b.Addf("\tcase %s:\n\t\treturn nil\n", field.KindConst)
	}
	b.Add("\tdefault:\n")
	b.Addf("\t\treturn loom.InvalidEnumValueError(%q, u.kind, []any{\n", data.TypeKey)
	for _, field := range data.Fields {
		b.Addf("\t\t\tstring(%s),\n", field.KindConst)
	}
	b.Add("\t\t})\n\t}")
	return b.String()
}

func renderHTTPUnionMarshalJSONBody(data *servicecodegen.UnionTypeData) string {
	if data.Untagged {
		return renderHTTPUntaggedUnionMarshalJSONBody(data)
	}
	var b sourceBuilder
	b.Add("if err := u.Validate(); err != nil {\n\treturn nil, err\n}\n")
	b.Add("var (\n\tvalue any\n)\n")
	b.Add("switch u.kind {\n")
	for _, field := range data.Fields {
		b.Addf("\tcase %s:\n\t\tvalue = u.%s\n", field.KindConst, field.FieldName)
	}
	b.Addf("\tdefault:\n\t\treturn nil, fmt.Errorf(\"unexpected %s discriminant %%q\", u.kind)\n\t}\n", data.Name)
	b.Addf("return json.Marshal(struct {\n\tType  string `json:\"%s\"`\n\tValue any    `json:\"%s\"`\n}{\n", data.TypeKey, data.ValueKey)
	b.Add("\tType:  string(u.kind),\n\tValue: value,\n}, json.Deterministic(true))")
	return b.String()
}

func renderHTTPUntaggedUnionMarshalJSONBody(data *servicecodegen.UnionTypeData) string {
	var b sourceBuilder
	b.Add("if err := u.Validate(); err != nil {\n\treturn nil, err\n}\n")
	b.Add("switch u.kind {\n")
	for _, field := range data.Fields {
		b.Addf(
			"\tcase %s:\n\t\treturn json.Marshal(u.%s, json.Deterministic(true))\n",
			field.KindConst,
			field.FieldName,
		)
	}
	b.Addf("\tdefault:\n\t\treturn nil, fmt.Errorf(\"unexpected %s discriminant %%q\", u.kind)\n\t}", data.Name)
	return b.String()
}

func renderHTTPUnionMarshalFormBody(data *servicecodegen.UnionTypeData) string {
	var b sourceBuilder
	b.Add("if err := u.Validate(); err != nil {\n\treturn err\n}\n")
	b.Addf("values.Set(loomhttp.FormChildKey(prefix, %q), string(u.kind))\n", data.TypeKey)
	b.Add("switch u.kind {\n")
	for _, field := range data.Fields {
		b.Addf("\tcase %s:\n", field.KindConst)
		if field.FlatFormObject {
			b.Addf("\t\t_, err := loomhttp.EncodeFormValue(values, prefix, u.%s)\n", field.FieldName)
		} else {
			b.Addf("\t\t_, err := loomhttp.EncodeFormValue(values, loomhttp.FormChildKey(prefix, %q), u.%s)\n", data.ValueKey, field.FieldName)
		}
		b.Add("\t\treturn err\n")
	}
	b.Addf("\tdefault:\n\t\treturn fmt.Errorf(\"unexpected %s discriminant %%q\", u.kind)\n\t}", data.Name)
	return b.String()
}

func renderHTTPUnionUnmarshalFormBody(data *servicecodegen.UnionTypeData) string {
	var b sourceBuilder
	b.Addf("typeKey := loomhttp.FormChildKey(prefix, %q)\n", data.TypeKey)
	if data.HasScalarFormBranch {
		b.Addf("valueKey := loomhttp.FormChildKey(prefix, %q)\n", data.ValueKey)
	}
	b.Add("rawType := values.Get(typeKey)\n")
	b.Add("if rawType == \"\" {\n")
	b.Addf("\treturn loom.MissingFieldError(%q, \"body\")\n}\n", data.TypeKey)
	b.Add("switch rawType {\n")
	for _, field := range data.Fields {
		b.Addf("\tcase string(%s):\n\t\tvar v %s\n", field.KindConst, field.FieldType)
		if field.FlatFormObject {
			b.Add("\t\tseen, err := loomhttp.DecodeFormValue(values, prefix, &v)\n")
		} else {
			b.Add("\t\tseen, err := loomhttp.DecodeFormValue(values, valueKey, &v)\n")
		}
		b.Add("\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n")
		b.Add("\t\tif !seen {\n")
		if field.FlatFormObjectAllowsEmpty {
			b.Addf("\t\t\tv = %s\n", field.EmptyValueExpr)
		} else {
			b.Addf("\t\t\treturn loom.MissingFieldError(%q, \"body\")\n", data.ValueKey)
		}
		b.Add("\t\t}\n")
		b.Addf("\t\tu.kind = %s\n\t\tu.%s = v\n", field.KindConst, field.FieldName)
	}
	b.Add("\tdefault:\n")
	b.Addf("\t\treturn loom.InvalidEnumValueError(%q, rawType, []any{\n", data.TypeKey)
	for _, field := range data.Fields {
		b.Addf("\t\t\tstring(%s),\n", field.KindConst)
	}
	b.Add("\t\t})\n\t}\nreturn nil")
	return b.String()
}

func renderHTTPUnionUnmarshalJSONBody(data *servicecodegen.UnionTypeData) string {
	if data.Untagged {
		return renderHTTPUntaggedUnionUnmarshalJSONBody(data)
	}
	var b sourceBuilder
	b.Addf("var raw struct {\n\tType  string         `json:\"%s\"`\n\tValue jsontext.Value `json:\"%s\"`\n}\n", data.TypeKey, data.ValueKey)
	b.Add("if err := json.Unmarshal(data, &raw); err != nil {\n\treturn err\n}\n")
	b.Add("switch raw.Type {\n")
	for _, field := range data.Fields {
		b.Addf("\tcase string(%s):\n\t\tvar v %s\n", field.KindConst, field.FieldType)
		b.Add("\t\tif len(raw.Value) == 0 || string(raw.Value) == \"null\" {\n")
		b.Addf("\t\t\treturn loom.MissingFieldError(%q, \"body\")\n\t\t}\n", data.ValueKey)
		b.Add("\t\tif err := json.Unmarshal(raw.Value, &v); err != nil {\n\t\t\treturn err\n\t\t}\n")
		b.Addf("\t\tu.kind = %s\n\t\tu.%s = v\n", field.KindConst, field.FieldName)
	}
	b.Add("\tdefault:\n")
	b.Addf("\t\treturn loom.InvalidEnumValueError(%q, raw.Type, []any{\n", data.TypeKey)
	for _, field := range data.Fields {
		b.Addf("\t\t\tstring(%s),\n", field.KindConst)
	}
	b.Add("\t\t})\n\t}\nreturn nil")
	return b.String()
}

func renderHTTPUntaggedUnionUnmarshalJSONBody(data *servicecodegen.UnionTypeData) string {
	var b sourceBuilder
	b.Add("var rawObject map[string]jsontext.Value\n")
	b.Add("if err := json.Unmarshal(data, &rawObject); err != nil {\n\treturn err\n}\n")
	b.Addf("if rawObject == nil {\n\treturn fmt.Errorf(\"decode %s: untagged union matched 0 branches\")\n}\n", data.Name)
	b.Add("matches := 0\n")
	b.Addf("var matched %s\n", data.Name)
	for _, field := range data.Fields {
		b.Addf("{\n\teligible := true\n\tvar v %s\n", field.FieldType)
		for _, name := range field.RequiredFields {
			b.Addf("\tif _, ok := rawObject[%q]; !ok {\n\t\teligible = false\n\t}\n", name)
		}
		for _, name := range field.NonNullableFields {
			b.Addf("\tif value, ok := rawObject[%q]; ok {\n\t\tvar decoded any\n\t\tif err := json.Unmarshal(value, &decoded); err != nil || decoded == nil {\n\t\t\teligible = false\n\t\t}\n\t}\n", name)
		}
		if field.RejectUnknownJSONFields {
			b.Add("\tfor name := range rawObject {\n\t\tswitch name {\n")
			for _, name := range field.JSONFields {
				b.Addf("\t\tcase %q:\n", name)
			}
			b.Add("\t\tdefault:\n\t\t\teligible = false\n\t\t}\n\t}\n")
		}
		b.Add("\tfiltered := make(map[string]jsontext.Value)\n")
		for _, name := range field.JSONFields {
			b.Addf("\tif value, ok := rawObject[%q]; ok {\n\t\tfiltered[%q] = value\n\t}\n", name, name)
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
			b.Addf("\t\tif err := %s; err == nil {\n", field.ValidateRef)
			validated = true
		}
		indent := "\t\t"
		if validated {
			indent = "\t\t\t"
		}
		b.Addf("%smatches++\n%smatched.kind = %s\n%smatched.%s = v\n", indent, indent, field.KindConst, indent, field.FieldName)
		if validated {
			b.Add("\t\t}\n")
		}
		b.Add("\t}\n\t}\n\t}\n}\n")
	}
	b.Addf("if matches != 1 {\n\treturn fmt.Errorf(\"decode %s: untagged union matched %%d branches\", matches)\n}\n", data.Name)
	b.Add("*u = matched\nreturn nil")
	return b.String()
}

func addRawHTTPUnionBlock(group *jen.Group, code string) {
	if strings.TrimSpace(code) == "" {
		return
	}
	group.Add(codegen.Expr(strings.TrimRight(code, "\n")))
}

func addHTTPUnionFormMethodComment(stmt *jen.Statement, methodName, verb string) {
	preposition := "into"
	if verb == "unmarshals" {
		preposition = "from"
	}
	stmt.Comment(methodName + " " + verb + " the union " + preposition + " application/x-www-form-urlencoded").Line()
	stmt.Comment("values using the discriminator field plus flattened object fields for").Line()
	stmt.Comment("object-shaped branches and the canonical {type,value} form shape for scalar").Line()
	stmt.Comment("branches.").Line()
}

func bodyInitSection(name string, init *InitData, client bool) codegen.Section {
	return codegen.NewJenniferSection(name, func(stmt *jen.Statement) {
		args, code := initRenderData(init, client)
		stmt.Line()
		codegen.Doc(stmt, init.Description)
		stmt.Func().
			Id(init.Name).
			ParamsFunc(func(group *jen.Group) {
				for _, arg := range args {
					group.Id(arg.VarName).Add(codegen.TypeRef(arg.TypeRef))
				}
			}).
			Add(codegen.TypeRef(init.ReturnTypeRef)).
			BlockFunc(func(group *jen.Group) {
				if code != "" {
					appendHTTPRawBlock(group, code)
				}
				group.Return(jen.Id("body"))
			})
		stmt.Line()
	})
}

func typeInitSection(name string, init *InitData, client bool) codegen.Section {
	return codegen.NewJenniferSection(name, func(stmt *jen.Statement) {
		args, code := initRenderData(init, client)
		typ := initRenderTarget(client)
		fieldInitCode := ""
		if !init.SkipFieldInit {
			fieldInitCode = strings.TrimRight(fieldCode(init, typ), "\n\t ")
		}

		stmt.Line()
		codegen.Doc(stmt, init.Description)
		stmt.Func().
			Id(init.Name).
			ParamsFunc(func(group *jen.Group) {
				for _, arg := range args {
					group.Id(arg.VarName).Add(codegen.TypeRef(arg.TypeRef))
				}
			}).
			Add(codegen.TypeRef(init.ReturnTypeRef)).
			BlockFunc(func(group *jen.Group) {
				if code != "" {
					appendHTTPRawBlock(group, code)
					if init.ReturnTypeAttribute != "" {
						valueExpr := "v"
						if init.ReturnIsPrimitivePointer {
							valueExpr = "&v"
						} else if init.ReturnIsUnionValue {
							valueExpr = "*v"
						}
						group.Id("res").Op(":=").Op("&").Id(init.ReturnTypeName).CustomFunc(jen.Options{
							Open:      "{",
							Close:     "}",
							Separator: ",",
							Multi:     true,
						}, func(values *jen.Group) {
							values.Id(init.ReturnTypeAttribute).Op(":").Add(codegen.Expr(valueExpr))
						})
					}
				} else if init.ReturnIsStruct {
					if init.ReturnTypeAttribute != "" {
						group.Id("res").Op(":=").Op("&").Id(init.ReturnTypeName).Values()
					} else {
						group.Id("v").Op(":=").Op("&").Id(init.ReturnTypeName).Values()
					}
				}
				if fieldInitCode != "" {
					appendHTTPRawBlock(group, fieldInitCode)
				}
				if code != "" || fieldInitCode != "" {
					group.Line()
				}
				if init.ReturnTypeAttribute != "" {
					group.Return(jen.Id("res"))
					return
				}
				group.Return(jen.Id("v"))
			})
		stmt.Line()
	})
}

func validateSection(name string, data *TypeData) codegen.Section {
	return codegen.NewJenniferSection(name, func(stmt *jen.Statement) {
		stmt.Line()
		codegen.Doc(stmt, fmt.Sprintf("Validate%s runs the validations defined on %s", data.VarName, data.Name))
		stmt.Func().
			Id("Validate" + data.VarName).
			Params(jen.Id("body").Add(codegen.TypeRef(data.Ref))).
			Params(jen.Id("err").Error()).
			BlockFunc(func(group *jen.Group) {
				if data.ValidateDef != "" {
					appendHTTPRawBlock(group, data.ValidateDef)
				}
				group.Return()
			})
		stmt.Line()
	})
}

func initRenderData(init *InitData, client bool) ([]*InitArgData, string) {
	if client {
		return init.ClientArgs, strings.TrimRight(init.ClientCode, "\n\t ")
	}
	return init.ServerArgs, strings.TrimRight(init.ServerCode, "\n\t ")
}

func initRenderTarget(client bool) string {
	if client {
		return "client"
	}
	return "server"
}

func addHTTPDocOrBlank(stmt *jen.Statement, description string) {
	if strings.TrimSpace(description) == "" {
		stmt.Line()
		return
	}
	codegen.Doc(stmt, description)
}
