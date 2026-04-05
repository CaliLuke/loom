package codegen

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	servicecodegen "github.com/CaliLuke/loom/codegen/service"
)

func typeDeclSection(name string, data *TypeData) codegen.Section {
	return codegen.MustJenniferSection(name, func(stmt *jen.Statement) {
		addHTTPDocOrBlank(stmt, data.Description)
		stmt.Type().Id(data.VarName).Add(codegen.Expr(data.Def))
		if data.FlatFormUnionField == "" {
			stmt.Line()
			return
		}

		stmt.Line()
		codegen.CommentBlock(stmt, "MarshalFormValues marshals the synthetic request body wrapper using the wrapped union field at the top level.")
		stmt.Func().
			Params(jen.Id("body").Id(data.VarName)).
			Id("MarshalFormValues").
			Params(jen.Id("values").Qual("net/url", "Values"), jen.Id("prefix").String()).
			Params(jen.Error()).
			Block(
				jen.Return(jen.Id("body").Dot(data.FlatFormUnionField).Dot("MarshalFormValues").Call(jen.Id("values"), jen.Id("prefix"))),
			)

		stmt.Line()
		codegen.CommentBlock(stmt, "UnmarshalFormValues unmarshals the synthetic request body wrapper using the wrapped union field at the top level.")
		stmt.Func().
			Params(jen.Id("body").Op("*").Id(data.VarName)).
			Id("UnmarshalFormValues").
			Params(jen.Id("values").Qual("net/url", "Values"), jen.Id("prefix").String()).
			Params(jen.Error()).
			Block(
				jen.Return(
					jen.Op("(&").Id("body").Dot(data.FlatFormUnionField).Op(")").
						Dot("UnmarshalFormValues").
						Call(jen.Id("values"), jen.Id("prefix")),
				),
			)
		stmt.Line()
	})
}

func renderTypeDecl(data *TypeData) string {
	var b sourceBuilder
	b.Add("\n")
	b.Add(codegen.Comment(data.Description))
	b.Add("\n")
	b.Addf("type %s %s\n", data.VarName, data.Def)
	if data.FlatFormUnionField == "" {
		return b.String()
	}
	b.Add("\n")
	b.Add("// MarshalFormValues marshals the synthetic request body wrapper using the\n")
	b.Add("// wrapped union field at the top level.\n")
	b.Addf("func (body %s) MarshalFormValues(values url.Values, prefix string) error {\n", data.VarName)
	b.Addf("\treturn body.%s.MarshalFormValues(values, prefix)\n", data.FlatFormUnionField)
	b.Add("}\n\n")
	b.Add("// UnmarshalFormValues unmarshals the synthetic request body wrapper using the\n")
	b.Add("// wrapped union field at the top level.\n")
	b.Addf("func (body *%s) UnmarshalFormValues(values url.Values, prefix string) error {\n", data.VarName)
	b.Addf("\treturn (&body.%s).UnmarshalFormValues(values, prefix)\n", data.FlatFormUnionField)
	b.Add("}\n")
	return b.String()
}

func unionTypeSection(name string, data *servicecodegen.UnionTypeData) codegen.Section {
	return codegen.MustRenderSection(name, func() string {
		return renderHTTPUnionType(data)
	})
}

func renderHTTPUnionType(data *servicecodegen.UnionTypeData) string {
	var b sourceBuilder
	b.Add("\n")
	writeHTTPUnionTypeScaffold(&b, data)
	writeHTTPUnionVariantMethods(&b, data)
	writeHTTPUnionValidationMethods(&b, data)
	writeHTTPUnionMarshalJSONMethod(&b, data)
	writeHTTPUnionFormMethods(&b, data)
	writeHTTPUnionUnmarshalJSONMethod(&b, data)
	return b.String()
}

func writeHTTPUnionTypeScaffold(b *sourceBuilder, data *servicecodegen.UnionTypeData) {
	for _, field := range data.Fields {
		if !field.EmitPrimitiveAlias {
			continue
		}
		b.Addf("type %s %s\n\n", field.FieldType, field.PrimitiveAliasType)
	}
	b.Addf("// %s is a sum-type union.\n", data.Name)
	b.Addf("type %s struct {\n", data.Name)
	b.Addf("\tkind %s\n", data.KindName)
	for _, field := range data.Fields {
		b.Addf("\t%s %s\n", field.FieldName, field.FieldType)
	}
	b.Add("}\n\n")
	b.Addf("// %s enumerates the union variants for %s.\n", data.KindName, data.Name)
	b.Addf("type %s string\n\n", data.KindName)
	b.Add("const (\n")
	for _, field := range data.Fields {
		b.Addf("\t// %s identifies the %s branch of the union.\n", field.KindConst, field.Name)
		b.Addf("\t%s %s = %q\n", field.KindConst, data.KindName, field.TypeTag)
	}
	b.Add(")\n\n")
	b.Addf("// Kind returns the discriminator value of the union.\nfunc (u %s) Kind() %s {\n\treturn u.kind\n}\n\n", data.Name, data.KindName)
}

func writeHTTPUnionVariantMethods(b *sourceBuilder, data *servicecodegen.UnionTypeData) {
	for _, field := range data.Fields {
		b.Addf("// New%s%s constructs a %s with the %s branch set.\n", data.Name, field.FieldName, data.Name, field.Name)
		b.Addf("func New%s%s(v %s) %s {\n", data.Name, field.FieldName, field.FieldType, data.Name)
		b.Addf("\treturn %s{\n\t\tkind:      %s,\n\t\t%s: v,\n\t}\n}\n\n", data.Name, field.KindConst, field.FieldName)
		b.Addf("// As%s returns the value of the %s branch if set.\n", field.FieldName, field.Name)
		b.Addf("func (u %s) As%s() (_ %s, ok bool) {\n", data.Name, field.FieldName, field.FieldType)
		b.Addf("\tif u.kind != %s {\n\t\treturn\n\t}\n", field.KindConst)
		b.Addf("\treturn u.%s, true\n}\n\n", field.FieldName)
		b.Addf("// Set%s sets the %s branch of the union.\n", field.FieldName, field.Name)
		b.Addf("func (u *%s) Set%s(v %s) {\n", data.Name, field.FieldName, field.FieldType)
		b.Addf("\tu.kind = %s\n\tu.%s = v\n}\n\n", field.KindConst, field.FieldName)
	}
}

func writeHTTPUnionValidationMethods(b *sourceBuilder, data *servicecodegen.UnionTypeData) {
	b.Addf("// Validate ensures the union discriminant is valid.\nfunc (u %s) Validate() error {\n", data.Name)
	b.Add("\tswitch u.kind {\n")
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
	b.Add("\t\t})\n\t}\n}\n\n")
}

func writeHTTPUnionMarshalJSONMethod(b *sourceBuilder, data *servicecodegen.UnionTypeData) {
	b.Addf("// MarshalJSON marshals the union into the canonical {type,value} JSON shape.\nfunc (u %s) MarshalJSON() ([]byte, error) {\n", data.Name)
	b.Add("\tif err := u.Validate(); err != nil {\n\t\treturn nil, err\n\t}\n")
	b.Add("\tvar (\n\t\tvalue any\n\t)\n")
	b.Add("\tswitch u.kind {\n")
	for _, field := range data.Fields {
		b.Addf("\tcase %s:\n\t\tvalue = u.%s\n", field.KindConst, field.FieldName)
	}
	b.Addf("\tdefault:\n\t\treturn nil, fmt.Errorf(\"unexpected %s discriminant %%q\", u.kind)\n\t}\n", data.Name)
	b.Addf("\treturn json.Marshal(struct {\n\t\tType  string `json:\"%s\"`\n\t\tValue any    `json:\"%s\"`\n\t}{\n", data.TypeKey, data.ValueKey)
	b.Add("\t\tType:  string(u.kind),\n\t\tValue: value,\n\t})\n}\n\n")
}

func writeHTTPUnionUnmarshalJSONMethod(b *sourceBuilder, data *servicecodegen.UnionTypeData) {
	b.Addf("// UnmarshalJSON unmarshals the union from the canonical {type,value} JSON shape.\nfunc (u *%s) UnmarshalJSON(data []byte) error {\n", data.Name)
	b.Addf("\tvar raw struct {\n\t\tType  string          `json:\"%s\"`\n\t\tValue json.RawMessage `json:\"%s\"`\n\t}\n", data.TypeKey, data.ValueKey)
	b.Add("\tif err := json.Unmarshal(data, &raw); err != nil {\n\t\treturn err\n\t}\n")
	b.Add("\tswitch raw.Type {\n")
	for _, field := range data.Fields {
		b.Addf("\tcase string(%s):\n\t\tvar v %s\n", field.KindConst, field.FieldType)
		b.Add("\t\tif err := json.Unmarshal(raw.Value, &v); err != nil {\n\t\t\treturn err\n\t\t}\n")
		b.Addf("\t\tu.kind = %s\n\t\tu.%s = v\n", field.KindConst, field.FieldName)
	}
	b.Addf("\tdefault:\n\t\treturn fmt.Errorf(\"unexpected %s type %%q\", raw.Type)\n\t}\n\treturn nil\n}\n", data.Name)
}

func writeHTTPUnionFormMethods(b *sourceBuilder, data *servicecodegen.UnionTypeData) {
	b.Addf("// MarshalFormValues marshals the union into application/x-www-form-urlencoded\n// values using the discriminator field plus flattened object fields for\n// object-shaped branches and the canonical {type,value} form shape for scalar\n// branches.\nfunc (u %s) MarshalFormValues(values url.Values, prefix string) error {\n", data.Name)
	b.Add("\tif err := u.Validate(); err != nil {\n\t\treturn err\n\t}\n")
	b.Addf("\tvalues.Set(loomhttp.FormChildKey(prefix, %q), string(u.kind))\n", data.TypeKey)
	b.Add("\tswitch u.kind {\n")
	for _, field := range data.Fields {
		b.Addf("\tcase %s:\n", field.KindConst)
		if field.FlatFormObject {
			b.Addf("\t\t_, err := loomhttp.EncodeFormValue(values, prefix, u.%s)\n", field.FieldName)
		} else {
			b.Addf("\t\t_, err := loomhttp.EncodeFormValue(values, loomhttp.FormChildKey(prefix, %q), u.%s)\n", data.ValueKey, field.FieldName)
		}
		b.Add("\t\treturn err\n")
	}
	b.Addf("\tdefault:\n\t\treturn fmt.Errorf(\"unexpected %s discriminant %%q\", u.kind)\n\t}\n}\n\n", data.Name)
	b.Addf("// UnmarshalFormValues unmarshals the union from application/x-www-form-urlencoded\n// values using the discriminator field plus flattened object fields for\n// object-shaped branches and the canonical {type,value} form shape for scalar\n// branches.\nfunc (u *%s) UnmarshalFormValues(values url.Values, prefix string) error {\n", data.Name)
	b.Addf("\ttypeKey := loomhttp.FormChildKey(prefix, %q)\n", data.TypeKey)
	if data.HasScalarFormBranch {
		b.Addf("\tvalueKey := loomhttp.FormChildKey(prefix, %q)\n", data.ValueKey)
	}
	b.Add("\trawType := values.Get(typeKey)\n")
	b.Add("\tif rawType == \"\" {\n")
	b.Addf("\t\treturn loom.MissingFieldError(%q, \"body\")\n\t}\n", data.TypeKey)
	b.Add("\tswitch rawType {\n")
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
	b.Addf("\tdefault:\n\t\treturn fmt.Errorf(\"unexpected %s type %%q\", rawType)\n\t}\n\treturn nil\n}\n\n", data.Name)
}

func bodyInitSection(name string, init *InitData, client bool) codegen.Section {
	return codegen.MustJenniferSection(name, func(stmt *jen.Statement) {
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

func renderBodyInit(init *InitData, client bool) string {
	args, code := initRenderData(init, client)
	var b sourceBuilder
	b.Add("\n")
	b.Add(codegen.Comment(init.Description))
	b.Add("\n")
	b.Addf("func %s(", init.Name)
	for _, arg := range args {
		b.Addf("%s %s, ", arg.VarName, arg.TypeRef)
	}
	b.Addf(") %s {\n", init.ReturnTypeRef)
	if code != "" {
		b.Addf("\t%s\n", code)
	}
	b.Add("\treturn body\n}\n")
	return stripBlankLineBeforeBodyReturn(b.String())
}

func typeInitSection(name string, init *InitData, client bool) codegen.Section {
	return codegen.MustJenniferSection(name, func(stmt *jen.Statement) {
		args, code := initRenderData(init, client)
		typ := initRenderTarget(client)
		fieldInitCode := strings.TrimRight(fieldCode(init, typ), "\n\t ")

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
						}
						appendHTTPRawBlock(group, fmt.Sprintf(
							"res := &%s{\n\t%s: %s,\n}",
							init.ReturnTypeName,
							init.ReturnTypeAttribute,
							valueExpr,
						))
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

func renderTypeInit(init *InitData, client bool) string {
	args, code := initRenderData(init, client)
	typ := initRenderTarget(client)

	var b sourceBuilder
	b.Add("\n")
	b.Add(codegen.Comment(init.Description))
	b.Add("\n")
	b.Addf("func %s(", init.Name)
	for _, arg := range args {
		b.Addf("%s %s, ", arg.VarName, arg.TypeRef)
	}
	b.Addf(") %s {\n", init.ReturnTypeRef)
	if code != "" {
		b.Addf("\t%s\n", code)
		if init.ReturnTypeAttribute != "" {
			b.Addf("\tres := &%s{\n", init.ReturnTypeName)
			if init.ReturnIsPrimitivePointer {
				b.Addf("\t\t%s: &v,\n", init.ReturnTypeAttribute)
			} else {
				b.Addf("\t\t%s: v,\n", init.ReturnTypeAttribute)
			}
			b.Add("\t}\n")
		}
	}
	if init.ReturnIsStruct && code == "" {
		if init.ReturnTypeAttribute != "" {
			b.Addf("\tres := &%s{}\n", init.ReturnTypeName)
		} else {
			b.Addf("\tv := &%s{}\n", init.ReturnTypeName)
		}
	}
	fieldInitCode := strings.TrimRight(fieldCode(init, typ), "\n\t ")
	if fieldInitCode != "" {
		b.Addf("\t%s\n", fieldInitCode)
	}
	if code != "" || fieldInitCode != "" {
		b.Add("\n")
	}
	if init.ReturnTypeAttribute != "" {
		b.Add("\treturn res\n}\n")
	} else {
		b.Add("\treturn v\n}\n")
	}
	return b.String()
}

func validateSection(name string, data *TypeData) codegen.Section {
	return codegen.MustJenniferSection(name, func(stmt *jen.Statement) {
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

func renderHTTPValidate(data *TypeData) string {
	var b sourceBuilder
	b.Add("\n")
	b.Add(codegen.Comment(fmt.Sprintf("Validate%s runs the validations defined on %s", data.VarName, data.Name)))
	b.Add("\n")
	b.Addf("func Validate%s(body %s) (err error) {\n", data.VarName, data.Ref)
	if data.ValidateDef != "" {
		b.Addf("\t%s\n", data.ValidateDef)
	}
	b.Add("\treturn\n}\n")
	return b.String()
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

func stripBlankLineBeforeBodyReturn(code string) string {
	return strings.ReplaceAll(code, "\n\n\treturn body", "\n\treturn body")
}

func addHTTPDocOrBlank(stmt *jen.Statement, description string) {
	if strings.TrimSpace(description) == "" {
		stmt.Line()
		return
	}
	codegen.Doc(stmt, description)
}
