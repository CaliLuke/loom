package service

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
)

func typeDefinitionSection(name, description, typeName, def string) codegen.Section {
	return codegen.NewRawSection(name, renderTypeDefinition(description, typeName, def))
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
	return codegen.NewRawSection("service-error", renderErrorMethods(data))
}

func validateSection(name string, data *ValidateData) codegen.Section {
	return codegen.NewRawSection(name, renderValidateFunction(data))
}

func viewedTypeMapSection(rtdata []*viewedType) codegen.Section {
	return codegen.NewRawSection("viewed-type-map", renderViewedTypeMap(rtdata))
}

func unionTypeSection(name string, data *UnionTypeData) codegen.Section {
	return codegen.NewRawSection(name, renderUnionType(data))
}

func renderTypeDefinition(description, typeName, def string) string {
	var b strings.Builder
	b.WriteString(codegen.Comment(description))
	b.WriteString("\n")
	fmt.Fprintf(&b, "type %s %s\n", typeName, def)
	return b.String()
}

func renderErrorMethods(data *UserTypeData) string {
	var b strings.Builder
	b.WriteString("// Error returns an error description.\n")
	fmt.Fprintf(&b, "func (e %s) Error() string {\n\treturn %q\n}\n\n", data.Ref, data.Description)
	b.WriteString("// ErrorName returns the error name.\n//\n")
	b.WriteString("// Deprecated: Use LoomErrorName.\n")
	fmt.Fprintf(&b, "func (e %s) ErrorName() string {\n\treturn e.LoomErrorName()\n}\n\n", data.Ref)
	b.WriteString("// LoomErrorName returns the error name.\n")
	fmt.Fprintf(&b, "func (e %s) LoomErrorName() string {\n\treturn %s\n}\n", data.Ref, errorName(data))
	if data.RemedyCode != "" || data.SafeMessage != "" || data.RetryHint != "" {
		b.WriteString("\n// LoomErrorRemedy returns the remediation guidance for the error.\n")
		fmt.Fprintf(&b, "func (e %s) LoomErrorRemedy() *goa.ErrorRemedy {\n", data.Ref)
		b.WriteString("\treturn &goa.ErrorRemedy{\n")
		fmt.Fprintf(&b, "\t\tCode:        %q,\n", data.RemedyCode)
		fmt.Fprintf(&b, "\t\tSafeMessage: %q,\n", data.SafeMessage)
		fmt.Fprintf(&b, "\t\tRetryHint:   %q,\n", data.RetryHint)
		b.WriteString("\t}\n}\n")
	}
	return b.String()
}

func renderValidateFunction(data *ValidateData) string {
	var b strings.Builder
	b.WriteString(codegen.Comment(data.Description))
	b.WriteString("\n")
	fmt.Fprintf(&b, "func %s(result %s) (err error) {\n", data.Name, data.Ref)
	if data.Validate != "" {
		b.WriteString(codegen.Indent(data.Validate, "\t"))
		if !strings.HasSuffix(data.Validate, "\n") {
			b.WriteString("\n")
		}
	} else {
		b.WriteString("\n")
	}
	b.WriteString("\treturn\n}\n")
	return b.String()
}

func renderViewedTypeMap(rtdata []*viewedType) string {
	var b strings.Builder
	b.WriteString("var (\n")
	for _, vt := range rtdata {
		b.WriteString(codegen.Indent(codegen.Comment(fmt.Sprintf("%sMap is a map indexing the attribute names of %s by view name.", vt.Name, vt.Name)), "\t"))
		b.WriteString("\n")
		fmt.Fprintf(&b, "\t%sMap = map[string][]string{\n", vt.Name)
		for _, view := range vt.Views {
			fmt.Fprintf(&b, "\t\t%q: {\n", view.Name)
			for _, attr := range view.Attributes {
				fmt.Fprintf(&b, "\t\t\t%q,\n", attr)
			}
			b.WriteString("\t\t},\n")
		}
		b.WriteString("\t}\n")
	}
	b.WriteString(")\n")
	return b.String()
}

func renderUnionType(data *UnionTypeData) string {
	var b strings.Builder
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
