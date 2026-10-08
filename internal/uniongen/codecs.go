//nolint:errcheck // Generator helpers write only to in-memory builders.
package uniongen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
)

// ValidateBody renders the body of the union Validate method.
func ValidateBody(data *Type) string {
	var b strings.Builder
	b.WriteString("switch u.kind {\n")
	fmt.Fprintf(&b, "\tcase %q:\n", "")
	fmt.Fprintf(&b, "\t\treturn loom.InvalidEnumValueError(%q, %q, []any{\n", data.TypeKey, "")
	for _, field := range data.Fields {
		fmt.Fprintf(&b, "\t\t\tstring(%s),\n", field.KindConst)
	}
	b.WriteString("\t\t})\n")
	for _, field := range data.Fields {
		fmt.Fprintf(&b, "\tcase %s:\n\t\treturn nil\n", field.KindConst)
	}
	b.WriteString("\tdefault:\n")
	fmt.Fprintf(&b, "\t\treturn loom.InvalidEnumValueError(%q, u.kind, []any{\n", data.TypeKey)
	for _, field := range data.Fields {
		fmt.Fprintf(&b, "\t\t\tstring(%s),\n", field.KindConst)
	}
	b.WriteString("\t\t})\n\t}")
	return b.String()
}

// MarshalJSONBody renders the body of the union MarshalJSON method.
func MarshalJSONBody(data *Type) string {
	if data.Untagged {
		return data.JSON.Marshal()
	}
	var b strings.Builder
	b.WriteString("if err := u.Validate(); err != nil {\n\treturn nil, err\n}\n")
	b.WriteString("var (\n\tvalue any\n)\n")
	b.WriteString("switch u.kind {\n")
	for _, field := range data.Fields {
		fmt.Fprintf(&b, "\tcase %s:\n\t\tvalue = u.%s\n", field.KindConst, field.FieldName)
	}
	fmt.Fprintf(&b, "\tdefault:\n\t\treturn nil, fmt.Errorf(\"unexpected %s discriminant %%q\", u.kind)\n\t}\n", data.Name)
	fmt.Fprintf(&b, "return json.Marshal(struct {\n\tType  string%s\n\tValue any   %s\n}{\n", codegen.StructTag(map[string]string{"json": data.TypeKey}), codegen.StructTag(map[string]string{"json": data.ValueKey}))
	b.WriteString("\tType:  string(u.kind),\n\tValue: value,\n}, loom.JSONOptions(), json.Deterministic(true))")
	return b.String()
}

// MarshalFormBody renders the body of the union MarshalForm method.
func MarshalFormBody(data *Type) string {
	var b strings.Builder
	b.WriteString("if err := u.Validate(); err != nil {\n\treturn err\n}\n")
	fmt.Fprintf(&b, "values.Set(loomhttp.FormChildKey(prefix, %q), string(u.kind))\n", data.TypeKey)
	b.WriteString("switch u.kind {\n")
	for _, field := range data.Fields {
		fmt.Fprintf(&b, "\tcase %s:\n", field.KindConst)
		if field.FlatFormObject {
			fmt.Fprintf(&b, "\t\t_, err := loomhttp.EncodeFormValue(values, prefix, u.%s)\n", field.FieldName)
		} else {
			fmt.Fprintf(&b, "\t\t_, err := loomhttp.EncodeFormValue(values, loomhttp.FormChildKey(prefix, %q), u.%s)\n", data.ValueKey, field.FieldName)
		}
		b.WriteString("\t\treturn err\n")
	}
	fmt.Fprintf(&b, "\tdefault:\n\t\treturn fmt.Errorf(\"unexpected %s discriminant %%q\", u.kind)\n\t}", data.Name)
	return b.String()
}

// UnmarshalFormBody renders the body of the union UnmarshalForm method.
func UnmarshalFormBody(data *Type) string {
	var b strings.Builder
	fmt.Fprintf(&b, "typeKey := loomhttp.FormChildKey(prefix, %q)\n", data.TypeKey)
	if data.HasScalarFormBranch {
		fmt.Fprintf(&b, "valueKey := loomhttp.FormChildKey(prefix, %q)\n", data.ValueKey)
	}
	b.WriteString("rawType := values.Get(typeKey)\n")
	b.WriteString("if rawType == \"\" {\n")
	fmt.Fprintf(&b, "\treturn loom.MissingFieldError(%q, \"body\")\n}\n", data.TypeKey)
	b.WriteString("switch rawType {\n")
	for _, field := range data.Fields {
		fmt.Fprintf(&b, "\tcase string(%s):\n\t\tvar v %s\n", field.KindConst, field.FieldType)
		if field.FlatFormObject {
			b.WriteString("\t\tseen, err := loomhttp.DecodeFormValue(values, prefix, &v)\n")
		} else {
			b.WriteString("\t\tseen, err := loomhttp.DecodeFormValue(values, valueKey, &v)\n")
		}
		b.WriteString("\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n")
		b.WriteString("\t\tif !seen {\n")
		if field.FlatFormObjectAllowsEmpty {
			fmt.Fprintf(&b, "\t\t\tv = %s\n", field.EmptyValueExpr)
		} else {
			fmt.Fprintf(&b, "\t\t\treturn loom.MissingFieldError(%q, \"body\")\n", data.ValueKey)
		}
		b.WriteString("\t\t}\n")
		fmt.Fprintf(&b, "\t\tu.kind = %s\n\t\tu.%s = v\n", field.KindConst, field.FieldName)
	}
	b.WriteString("\tdefault:\n")
	fmt.Fprintf(&b, "\t\treturn loom.InvalidEnumValueError(%q, rawType, []any{\n", data.TypeKey)
	for _, field := range data.Fields {
		fmt.Fprintf(&b, "\t\t\tstring(%s),\n", field.KindConst)
	}
	b.WriteString("\t\t})\n\t}\nreturn nil")
	return b.String()
}

// UnmarshalJSONBody renders the body of the union UnmarshalJSON method.
func UnmarshalJSONBody(data *Type) string {
	if data.Untagged {
		return data.JSON.Unmarshal()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "var raw struct {\n\tType  string        %s\n\tValue jsontext.Value%s\n}\n", codegen.StructTag(map[string]string{"json": data.TypeKey}), codegen.StructTag(map[string]string{"json": data.ValueKey}))
	b.WriteString("if err := json.Unmarshal(data, &raw); err != nil {\n\treturn err\n}\n")
	b.WriteString("switch raw.Type {\n")
	for _, field := range data.Fields {
		fmt.Fprintf(&b, "\tcase string(%s):\n\t\tvar v %s\n", field.KindConst, field.FieldType)
		b.WriteString("\t\tif len(raw.Value) == 0 || string(raw.Value) == \"null\" {\n")
		fmt.Fprintf(&b, "\t\t\treturn loom.MissingFieldError(%q, \"body\")\n\t\t}\n", data.ValueKey)
		b.WriteString("\t\tif err := json.Unmarshal(raw.Value, &v, loom.JSONOptions()); err != nil {\n\t\t\treturn err\n\t\t}\n")
		fmt.Fprintf(&b, "\t\tu.kind = %s\n\t\tu.%s = v\n", field.KindConst, field.FieldName)
	}
	b.WriteString("\tdefault:\n")
	fmt.Fprintf(&b, "\t\treturn loom.InvalidEnumValueError(%q, raw.Type, []any{\n", data.TypeKey)
	for _, field := range data.Fields {
		fmt.Fprintf(&b, "\t\t\tstring(%s),\n", field.KindConst)
	}
	b.WriteString("\t\t})\n\t}\nreturn nil")
	return b.String()
}
