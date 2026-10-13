//nolint:errcheck // Generator helpers write only to in-memory builders.
package codegen

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/expr"
)

func renderGRPCMetadataAppend(md *MetadataData, root string) string {
	var b strings.Builder
	value := fieldSelector(root, md.FieldName)
	switch {
	case md.StringSlice:
		fmt.Fprintf(&b, "\tfor _, value := range %s {\n", value)
		fmt.Fprintf(&b, "\t\t(*md).Append(%q, value)\n", md.Name)
		b.WriteString("\t}\n")
	case md.Slice:
		fmt.Fprintf(&b, "\tfor _, value := range %s {\n", value)
		b.WriteString(indent(renderGRPCMetadataElement(md, "valueStr", "value"), 2))
		fmt.Fprintf(&b, "\t\t(*md).Append(%q, valueStr)\n", md.Name)
		b.WriteString("\t}\n")
	default:
		if md.Pointer {
			fmt.Fprintf(&b, "\tif %s != nil {\n", value)
		}
		if md.Bearer.Framed {
			fmt.Fprintf(&b, "\t\t(*md).Append(%q, \"Bearer \"+%s)\n", md.Name, renderMetadataSingleValue(md, value))
		} else {
			fmt.Fprintf(&b, "\t\t(*md).Append(%q, %s)\n", md.Name, renderMetadataSingleValue(md, value))
		}
		if md.Pointer {
			b.WriteString("\t}\n")
		}
	}
	return b.String()
}

// renderGRPCMetadataDecode owns presence before any type-specific parsing.
// A present textual zero or empty string never takes the absence branch, and
// malformed supplied values retain their parser error instead of defaulting.
func renderGRPCMetadataDecode(md *MetadataData, mdVar string, namedResults bool) string {
	var b strings.Builder
	name := renderJen(jen.Lit(md.Name))
	fmt.Fprintf(&b, "\t\tif vals := %s.Get(%s); len(vals) > 0 {\n", mdVar, name)
	b.WriteString(indent(renderGRPCMetadataValue(md, namedResults), 3))
	switch {
	case md.Required:
		b.WriteString("\t\t} else {\n")
		fmt.Fprintf(&b, "\t\t\terr = loom.MergeErrors(err, loom.MissingFieldError(%s, \"metadata\"))\n", name)
	case md.DefaultLiteral != "":
		b.WriteString("\t\t} else {\n")
		if md.Pointer {
			fmt.Fprintf(&b, "\t\t\t%s = new(%s)\n", md.VarName, md.TypeName)
			fmt.Fprintf(&b, "\t\t\t*%s = %s\n", md.VarName, md.DefaultLiteral)
		} else {
			fmt.Fprintf(&b, "\t\t\t%s = %s\n", md.VarName, md.DefaultLiteral)
		}
	}
	b.WriteString("\t\t}\n")
	return b.String()
}

func renderGRPCMetadataValue(md *MetadataData, namedResults bool) string {
	switch {
	case md.Bearer.Framed:
		return renderGRPCBearerMetadataValue(md, namedResults)
	case md.TypeName == "string" || md.Type.Name() == "any":
		return fmt.Sprintf("%s = %svals[0]\n", md.VarName, addrPrefix(md.Pointer))
	case md.StringSlice:
		return fmt.Sprintf("%s = vals\n", md.VarName)
	case md.Slice:
		return renderGRPCSliceConversion(md, "vals")
	default:
		rawVar := md.VarName + "Raw"
		return fmt.Sprintf("%s := vals[0]\n\n", rawVar) + renderGRPCStringParse(md, rawVar)
	}
}

func renderGRPCBearerMetadataValue(md *MetadataData, namedResults bool) string {
	var b strings.Builder
	b.WriteString("value, ok := security.DecodeBearer(vals[0])\nif !ok")
	if md.Bearer.AllowOtherScheme {
		b.WriteString(" && !security.IsOtherAuthorizationScheme(vals[0], \"Bearer\")")
	}
	b.WriteString(" {\n")
	if namedResults {
		b.WriteString("\terr = loom.DecodePayloadError(\"invalid Authorization bearer credential\")\n\treturn\n")
	} else {
		b.WriteString("\treturn nil, loom.DecodePayloadError(\"invalid Authorization bearer credential\")\n")
	}
	b.WriteString("}\n")
	if md.Bearer.AllowOtherScheme {
		b.WriteString("if ok {\n")
	}
	fmt.Fprintf(&b, "%s = %svalue\n", md.VarName, addrPrefix(md.Pointer))
	if md.Bearer.AllowOtherScheme {
		b.WriteString("}\n")
	}
	return b.String()
}

func renderGRPCMetadataEncode(md *MetadataData, targetVar string) string {
	var b strings.Builder
	value := fieldSelector("result", md.FieldName)
	switch {
	case md.StringSlice:
		fmt.Fprintf(&b, "\t%s.Append(%q, %s...)\n", targetVar, md.Name, value)
	case md.Slice:
		fmt.Fprintf(&b, "\tfor _, value := range %s {\n", value)
		b.WriteString(indent(renderGRPCMetadataElement(md, "valueStr", "value"), 2))
		fmt.Fprintf(&b, "\t\t%s.Append(%q, valueStr)\n", targetVar, md.Name)
		b.WriteString("\t}\n")
	default:
		if md.Pointer {
			fmt.Fprintf(&b, "\tif %s != nil {\n", value)
		}
		fmt.Fprintf(&b, "\t\t%s.Append(%q, %s)\n", targetVar, md.Name, renderMetadataSingleValue(md, value))
		if md.Pointer {
			b.WriteString("\t}\n")
		}
	}
	return b.String()
}

func renderMetadataSingleValue(md *MetadataData, value string) string {
	switch md.Type.Name() {
	case "bytes":
		return "string(" + pointerPrefix(md.Pointer) + value + ")"
	case "string":
		if expr.IsAlias(md.FieldType) {
			return "string(" + pointerPrefix(md.Pointer) + value + ")"
		}
		return pointerPrefix(md.Pointer) + value
	default:
		return renderJen(jen.Qual("fmt", "Sprintf").Call(jen.Lit("%v"), exprCode(pointerPrefix(md.Pointer)+value)))
	}
}

func renderGRPCMetadataElement(md *MetadataData, varName, value string) string {
	if md.elementCast != "" {
		value = md.elementCast + "(" + value + ")"
	}
	return renderGRPCStringConversion(expr.AsArray(md.Type).ElemType.Type, varName, value)
}
