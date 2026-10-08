//nolint:errcheck // Generator helpers write only to in-memory builders.
package service

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/uniongen"
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
		decl := stmt.Type().Id(typeName)
		if isAliasTypeDef(def) {
			decl.Op("=")
		}
		decl.Add(codegen.Expr(def))
		stmt.Line()
	})
}

// isAliasTypeDef reports whether a user type with the Go type definition def
// is generated as an alias of def rather than as a defined type.
func isAliasTypeDef(def string) bool {
	return def == "loom.JSONValue"
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
		uniongen.Emit(stmt, &data.Type)
	})
}

func renderErrorMethods(data *UserTypeData) string {
	var b strings.Builder
	b.WriteString("// Error returns an error description.\n")
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
	b.WriteString("// LoomErrorName returns the error name.\n")
	fmt.Fprintf(&b, "func (e %s) LoomErrorName() string {\n\treturn %s\n}\n", data.Ref, errorName(data))
	if data.RemedyCode != "" || data.SafeMessage != "" || data.RetryHint != "" {
		b.WriteString("\n// LoomErrorRemedy returns the remediation guidance for the error.\n")
		fmt.Fprintf(&b, "func (e %s) LoomErrorRemedy() *loom.ErrorRemedy {\n", data.Ref)
		b.WriteString("\treturn &loom.ErrorRemedy{\n")
		fmt.Fprintf(&b, "\t\tCode:        %q,\n", data.RemedyCode)
		fmt.Fprintf(&b, "\t\tSafeMessage: %q,\n", data.SafeMessage)
		fmt.Fprintf(&b, "\t\tRetryHint:   %q,\n", data.RetryHint)
		b.WriteString("\t}\n}\n")
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
