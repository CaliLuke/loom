package codegen

import (
	"fmt"
	"regexp"
	"strings"

	"goa.design/goa/v3/codegen"
	servicecodegen "goa.design/goa/v3/codegen/service"
)

var blankLineBeforeReturnRE = regexp.MustCompile(`\n[ \t]*\n(\treturn body)`)

func typeDeclSection(name string, data *TypeData) codegen.Section {
	return codegen.NewRawSection(name, renderTypeDecl(data))
}

func renderTypeDecl(data *TypeData) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(codegen.Comment(data.Description))
	b.WriteString("\n")
	fmt.Fprintf(&b, "type %s %s\n", data.VarName, data.Def)
	if data.FlatFormUnionField == "" {
		return b.String()
	}
	b.WriteString("\n")
	b.WriteString("// MarshalFormValues marshals the synthetic request body wrapper using the\n")
	b.WriteString("// wrapped union field at the top level.\n")
	fmt.Fprintf(&b, "func (body %s) MarshalFormValues(values url.Values, prefix string) error {\n", data.VarName)
	fmt.Fprintf(&b, "\treturn body.%s.MarshalFormValues(values, prefix)\n", data.FlatFormUnionField)
	b.WriteString("}\n\n")
	b.WriteString("// UnmarshalFormValues unmarshals the synthetic request body wrapper using the\n")
	b.WriteString("// wrapped union field at the top level.\n")
	fmt.Fprintf(&b, "func (body *%s) UnmarshalFormValues(values url.Values, prefix string) error {\n", data.VarName)
	fmt.Fprintf(&b, "\treturn (&body.%s).UnmarshalFormValues(values, prefix)\n", data.FlatFormUnionField)
	b.WriteString("}\n")
	return b.String()
}

func unionTypeSection(name string, data *servicecodegen.UnionTypeData) codegen.Section {
	return codegen.NewRawSection(name, renderHTTPUnionType(data))
}

func renderHTTPUnionType(data *servicecodegen.UnionTypeData) string {
	var b strings.Builder
	b.WriteString("\n")
	for _, field := range data.Fields {
		if !field.EmitPrimitiveAlias {
			continue
		}
		fmt.Fprintf(&b, "type %s %s\n\n", field.FieldType, field.PrimitiveAliasType)
	}

	fmt.Fprintf(&b, "// %s is a sum-type union.\n", data.Name)
	fmt.Fprintf(&b, "type %s struct {\n", data.Name)
	fmt.Fprintf(&b, "\tkind %s\n", data.KindName)
	for _, field := range data.Fields {
		fmt.Fprintf(&b, "\t%s %s\n", field.FieldName, field.FieldType)
	}
	b.WriteString("}\n\n")

	fmt.Fprintf(&b, "// %s enumerates the union variants for %s.\n", data.KindName, data.Name)
	fmt.Fprintf(&b, "type %s string\n\n", data.KindName)

	b.WriteString("const (\n")
	for _, field := range data.Fields {
		fmt.Fprintf(&b, "\t// %s identifies the %s branch of the union.\n", field.KindConst, field.Name)
		fmt.Fprintf(&b, "\t%s %s = %q\n", field.KindConst, data.KindName, field.TypeTag)
	}
	b.WriteString(")\n\n")

	fmt.Fprintf(&b, "// Kind returns the discriminator value of the union.\nfunc (u %s) Kind() %s {\n\treturn u.kind\n}\n\n", data.Name, data.KindName)

	for _, field := range data.Fields {
		fmt.Fprintf(&b, "// New%s%s constructs a %s with the %s branch set.\n", data.Name, field.FieldName, data.Name, field.Name)
		fmt.Fprintf(&b, "func New%s%s(v %s) %s {\n", data.Name, field.FieldName, field.FieldType, data.Name)
		fmt.Fprintf(&b, "\treturn %s{\n\t\tkind:      %s,\n\t\t%s: v,\n\t}\n}\n\n", data.Name, field.KindConst, field.FieldName)

		fmt.Fprintf(&b, "// As%s returns the value of the %s branch if set.\n", field.FieldName, field.Name)
		fmt.Fprintf(&b, "func (u %s) As%s() (_ %s, ok bool) {\n", data.Name, field.FieldName, field.FieldType)
		fmt.Fprintf(&b, "\tif u.kind != %s {\n\t\treturn\n\t}\n", field.KindConst)
		fmt.Fprintf(&b, "\treturn u.%s, true\n}\n\n", field.FieldName)

		fmt.Fprintf(&b, "// Set%s sets the %s branch of the union.\n", field.FieldName, field.Name)
		fmt.Fprintf(&b, "func (u *%s) Set%s(v %s) {\n", data.Name, field.FieldName, field.FieldType)
		fmt.Fprintf(&b, "\tu.kind = %s\n\tu.%s = v\n}\n\n", field.KindConst, field.FieldName)
	}

	fmt.Fprintf(&b, "// Validate ensures the union discriminant is valid.\nfunc (u %s) Validate() error {\n", data.Name)
	b.WriteString("\tswitch u.kind {\n")
	fmt.Fprintf(&b, "\tcase %q:\n", "")
	fmt.Fprintf(&b, "\t\treturn goa.InvalidEnumValueError(%q, %q, []any{\n", data.TypeKey, "")
	for _, field := range data.Fields {
		fmt.Fprintf(&b, "\t\t\tstring(%s),\n", field.KindConst)
	}
	b.WriteString("\t\t})\n")
	for _, field := range data.Fields {
		fmt.Fprintf(&b, "\tcase %s:\n\t\treturn nil\n", field.KindConst)
	}
	b.WriteString("\tdefault:\n")
	fmt.Fprintf(&b, "\t\treturn goa.InvalidEnumValueError(%q, u.kind, []any{\n", data.TypeKey)
	for _, field := range data.Fields {
		fmt.Fprintf(&b, "\t\t\tstring(%s),\n", field.KindConst)
	}
	b.WriteString("\t\t})\n\t}\n}\n\n")

	fmt.Fprintf(&b, "// MarshalJSON marshals the union into the canonical {type,value} JSON shape.\nfunc (u %s) MarshalJSON() ([]byte, error) {\n", data.Name)
	b.WriteString("\tif err := u.Validate(); err != nil {\n\t\treturn nil, err\n\t}\n")
	b.WriteString("\tvar (\n\t\tvalue any\n\t)\n")
	b.WriteString("\tswitch u.kind {\n")
	for _, field := range data.Fields {
		fmt.Fprintf(&b, "\tcase %s:\n\t\tvalue = u.%s\n", field.KindConst, field.FieldName)
	}
	fmt.Fprintf(&b, "\tdefault:\n\t\treturn nil, fmt.Errorf(\"unexpected %s discriminant %%q\", u.kind)\n\t}\n", data.Name)
	fmt.Fprintf(&b, "\treturn json.Marshal(struct {\n\t\tType  string `json:\"%s\"`\n\t\tValue any    `json:\"%s\"`\n\t}{\n", data.TypeKey, data.ValueKey)
	b.WriteString("\t\tType:  string(u.kind),\n\t\tValue: value,\n\t})\n}\n\n")

	fmt.Fprintf(&b, "// MarshalFormValues marshals the union into application/x-www-form-urlencoded\n// values using the discriminator field plus flattened object fields for\n// object-shaped branches and the canonical {type,value} form shape for scalar\n// branches.\nfunc (u %s) MarshalFormValues(values url.Values, prefix string) error {\n", data.Name)
	b.WriteString("\tif err := u.Validate(); err != nil {\n\t\treturn err\n\t}\n")
	fmt.Fprintf(&b, "\tvalues.Set(goahttp.FormChildKey(prefix, %q), string(u.kind))\n", data.TypeKey)
	b.WriteString("\tswitch u.kind {\n")
	for _, field := range data.Fields {
		fmt.Fprintf(&b, "\tcase %s:\n", field.KindConst)
		if field.FlatFormObject {
			fmt.Fprintf(&b, "\t\t_, err := goahttp.EncodeFormValue(values, prefix, u.%s)\n", field.FieldName)
		} else {
			fmt.Fprintf(&b, "\t\t_, err := goahttp.EncodeFormValue(values, goahttp.FormChildKey(prefix, %q), u.%s)\n", data.ValueKey, field.FieldName)
		}
		b.WriteString("\t\treturn err\n")
	}
	fmt.Fprintf(&b, "\tdefault:\n\t\treturn fmt.Errorf(\"unexpected %s discriminant %%q\", u.kind)\n\t}\n}\n\n", data.Name)

	fmt.Fprintf(&b, "// UnmarshalFormValues unmarshals the union from application/x-www-form-urlencoded\n// values using the discriminator field plus flattened object fields for\n// object-shaped branches and the canonical {type,value} form shape for scalar\n// branches.\nfunc (u *%s) UnmarshalFormValues(values url.Values, prefix string) error {\n", data.Name)
	fmt.Fprintf(&b, "\ttypeKey := goahttp.FormChildKey(prefix, %q)\n", data.TypeKey)
	if data.HasScalarFormBranch {
		fmt.Fprintf(&b, "\tvalueKey := goahttp.FormChildKey(prefix, %q)\n", data.ValueKey)
	}
	b.WriteString("\trawType := values.Get(typeKey)\n")
	b.WriteString("\tif rawType == \"\" {\n")
	fmt.Fprintf(&b, "\t\treturn goa.MissingFieldError(%q, \"body\")\n\t}\n", data.TypeKey)
	b.WriteString("\tswitch rawType {\n")
	for _, field := range data.Fields {
		fmt.Fprintf(&b, "\tcase string(%s):\n\t\tvar v %s\n", field.KindConst, field.FieldType)
		if field.FlatFormObject {
			b.WriteString("\t\tseen, err := goahttp.DecodeFormValue(values, prefix, &v)\n")
		} else {
			b.WriteString("\t\tseen, err := goahttp.DecodeFormValue(values, valueKey, &v)\n")
		}
		b.WriteString("\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n")
		b.WriteString("\t\tif !seen {\n")
		if field.FlatFormObjectAllowsEmpty {
			fmt.Fprintf(&b, "\t\t\tv = %s\n", field.EmptyValueExpr)
		} else {
			fmt.Fprintf(&b, "\t\t\treturn goa.MissingFieldError(%q, \"body\")\n", data.ValueKey)
		}
		b.WriteString("\t\t}\n")
		fmt.Fprintf(&b, "\t\tu.kind = %s\n\t\tu.%s = v\n", field.KindConst, field.FieldName)
	}
	fmt.Fprintf(&b, "\tdefault:\n\t\treturn fmt.Errorf(\"unexpected %s type %%q\", rawType)\n\t}\n\treturn nil\n}\n\n", data.Name)

	fmt.Fprintf(&b, "// UnmarshalJSON unmarshals the union from the canonical {type,value} JSON shape.\nfunc (u *%s) UnmarshalJSON(data []byte) error {\n", data.Name)
	fmt.Fprintf(&b, "\tvar raw struct {\n\t\tType  string          `json:\"%s\"`\n\t\tValue json.RawMessage `json:\"%s\"`\n\t}\n", data.TypeKey, data.ValueKey)
	b.WriteString("\tif err := json.Unmarshal(data, &raw); err != nil {\n\t\treturn err\n\t}\n")
	b.WriteString("\tswitch raw.Type {\n")
	for _, field := range data.Fields {
		fmt.Fprintf(&b, "\tcase string(%s):\n\t\tvar v %s\n", field.KindConst, field.FieldType)
		b.WriteString("\t\tif err := json.Unmarshal(raw.Value, &v); err != nil {\n\t\t\treturn err\n\t\t}\n")
		fmt.Fprintf(&b, "\t\tu.kind = %s\n\t\tu.%s = v\n", field.KindConst, field.FieldName)
	}
	fmt.Fprintf(&b, "\tdefault:\n\t\treturn fmt.Errorf(\"unexpected %s type %%q\", raw.Type)\n\t}\n\treturn nil\n}\n", data.Name)
	return b.String()
}

func bodyInitSection(name string, init *InitData, client bool) codegen.Section {
	return codegen.NewRawSection(name, renderBodyInit(init, client))
}

func renderBodyInit(init *InitData, client bool) string {
	var args []*InitArgData
	code := strings.TrimRight(init.ServerCode, "\n\t ")
	if client {
		args = init.ClientArgs
		code = strings.TrimRight(init.ClientCode, "\n\t ")
	} else {
		args = init.ServerArgs
	}
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(codegen.Comment(init.Description))
	b.WriteString("\n")
	fmt.Fprintf(&b, "func %s(", init.Name)
	for _, arg := range args {
		fmt.Fprintf(&b, "%s %s, ", arg.VarName, arg.TypeRef)
	}
	fmt.Fprintf(&b, ") %s {\n", init.ReturnTypeRef)
	if code != "" {
		fmt.Fprintf(&b, "\t%s\n", code)
	}
	b.WriteString("\treturn body\n}\n")
	return blankLineBeforeReturnRE.ReplaceAllString(b.String(), "\n$1")
}

func typeInitSection(name string, init *InitData, client bool) codegen.Section {
	return codegen.NewRawSection(name, renderTypeInit(init, client))
}

func renderTypeInit(init *InitData, client bool) string {
	var (
		args []*InitArgData
		code string
		typ  string
	)
	if client {
		args = init.ClientArgs
		code = strings.TrimRight(init.ClientCode, "\n\t ")
		typ = "client"
	} else {
		args = init.ServerArgs
		code = strings.TrimRight(init.ServerCode, "\n\t ")
		typ = "server"
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(codegen.Comment(init.Description))
	b.WriteString("\n")
	fmt.Fprintf(&b, "func %s(", init.Name)
	for _, arg := range args {
		fmt.Fprintf(&b, "%s %s, ", arg.VarName, arg.TypeRef)
	}
	fmt.Fprintf(&b, ") %s {\n", init.ReturnTypeRef)
	if code != "" {
		fmt.Fprintf(&b, "\t%s\n", code)
		if init.ReturnTypeAttribute != "" {
			fmt.Fprintf(&b, "\tres := &%s{\n", init.ReturnTypeName)
			if init.ReturnIsPrimitivePointer {
				fmt.Fprintf(&b, "\t\t%s: &v,\n", init.ReturnTypeAttribute)
			} else {
				fmt.Fprintf(&b, "\t\t%s: v,\n", init.ReturnTypeAttribute)
			}
			b.WriteString("\t}\n")
		}
	}
	if init.ReturnIsStruct && code == "" {
		if init.ReturnTypeAttribute != "" {
			fmt.Fprintf(&b, "\tres := &%s{}\n", init.ReturnTypeName)
		} else {
			fmt.Fprintf(&b, "\tv := &%s{}\n", init.ReturnTypeName)
		}
	}
	fieldInitCode := strings.TrimRight(fieldCode(init, typ), "\n\t ")
	if fieldInitCode != "" {
		fmt.Fprintf(&b, "\t%s\n", fieldInitCode)
	}
	if code != "" || fieldInitCode != "" {
		b.WriteString("\n")
	}
	if init.ReturnTypeAttribute != "" {
		b.WriteString("\treturn res\n}\n")
	} else {
		b.WriteString("\treturn v\n}\n")
	}
	return b.String()
}

func validateSection(name string, data *TypeData) codegen.Section {
	return codegen.NewRawSection(name, renderHTTPValidate(data))
}

func renderHTTPValidate(data *TypeData) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(codegen.Comment(fmt.Sprintf("Validate%s runs the validations defined on %s", data.VarName, data.Name)))
	b.WriteString("\n")
	fmt.Fprintf(&b, "func Validate%s(body %s) (err error) {\n", data.VarName, data.Ref)
	if data.ValidateDef != "" {
		fmt.Fprintf(&b, "\t%s\n", data.ValidateDef)
	}
	b.WriteString("\treturn\n}\n")
	return b.String()
}
