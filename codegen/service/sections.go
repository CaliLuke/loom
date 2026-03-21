package service

import (
	"fmt"
	"strings"

	"goa.design/goa/v3/codegen"
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
	b.WriteString("// Deprecated: Use GoaErrorName - https://github.com/goadesign/goa/issues/3105\n")
	fmt.Fprintf(&b, "func (e %s) ErrorName() string {\n\treturn e.GoaErrorName()\n}\n\n", data.Ref)
	b.WriteString("// GoaErrorName returns the error name.\n")
	fmt.Fprintf(&b, "func (e %s) GoaErrorName() string {\n\treturn %s\n}\n", data.Ref, errorName(data))
	if data.RemedyCode != "" || data.SafeMessage != "" || data.RetryHint != "" {
		b.WriteString("\n// GoaErrorRemedy returns the remediation guidance for the error.\n")
		fmt.Fprintf(&b, "func (e %s) GoaErrorRemedy() *goa.ErrorRemedy {\n", data.Ref)
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
