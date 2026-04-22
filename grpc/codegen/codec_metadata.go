//nolint:errcheck // Generator helpers write only to in-memory builders.
package codegen

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
)

func renderGRPCMetadataAppend(md *MetadataData, root string, schemes []*service.SchemeData) string {
	var b sourceBuilder
	value := fieldSelector(root, md.FieldName)
	switch {
	case md.StringSlice:
		fmt.Fprintf(&b, "\tfor _, value := range %s {\n", value)
		fmt.Fprintf(&b, "\t\t(*md).Append(%q, value)\n", md.Name)
		b.Add("\t}\n")
	case md.Slice:
		fmt.Fprintf(&b, "\tfor _, value := range %s {\n", value)
		b.Add(indent(renderGRPCStringConversion(expr.AsArray(md.Type).ElemType.Type, "valueStr", "value"), 2))
		fmt.Fprintf(&b, "\t\t(*md).Append(%q, valueStr)\n", md.Name)
		b.Add("\t}\n")
	default:
		if md.Pointer {
			fmt.Fprintf(&b, "\tif %s != nil {\n", value)
		}
		if md.Name == "Authorization" && isBearer(schemes) {
			fmt.Fprintf(&b, "\t\tif !strings.Contains(%s%s, \" \") {\n", pointerPrefix(md.Pointer), value)
			fmt.Fprintf(&b, "\t\t\t(*md).Append(%q, \"Bearer \"+%s%s)\n", md.Name, pointerPrefix(md.Pointer), value)
			b.Add("\t\t} else {\n")
			fmt.Fprintf(&b, "\t\t\t(*md).Append(%q, %s)\n", md.Name, renderMetadataSingleValue(md, value))
			b.Add("\t\t}\n")
		} else {
			fmt.Fprintf(&b, "\t\t(*md).Append(%q, %s)\n", md.Name, renderMetadataSingleValue(md, value))
		}
		if md.Pointer {
			b.Add("\t}\n")
		}
	}
	return b.String()
}

func renderGRPCMetadataDecode(md *MetadataData, mdVar string) string {
	var b sourceBuilder
	name := renderJen(jen.Lit(md.Name))
	switch {
	case md.TypeName == "string" || md.Type.Name() == "any":
		if md.Required {
			fmt.Fprintf(&b, "\t\tif vals := %s.Get(%s); len(vals) == 0 {\n", mdVar, name)
			fmt.Fprintf(&b, "\t\t\terr = loom.MergeErrors(err, loom.MissingFieldError(%s, \"metadata\"))\n", name)
			b.Add("\t\t} else {\n")
			fmt.Fprintf(&b, "\t\t\t%s = vals[0]\n", md.VarName)
			b.Add("\t\t}\n")
		} else {
			fmt.Fprintf(&b, "\t\tif vals := %s.Get(%s); len(vals) > 0 {\n", mdVar, name)
			fmt.Fprintf(&b, "\t\t\t%s = vals[0]\n", md.VarName)
			b.Add("\t\t}\n")
		}
	case md.StringSlice:
		if md.Required {
			fmt.Fprintf(&b, "\t\tif vals := %s.Get(%s); len(vals) == 0 {\n", mdVar, name)
			fmt.Fprintf(&b, "\t\t\terr = loom.MergeErrors(err, loom.MissingFieldError(%s, \"metadata\"))\n", name)
			b.Add("\t\t} else {\n")
			fmt.Fprintf(&b, "\t\t\t%s = vals\n", md.VarName)
			b.Add("\t\t}\n")
		} else {
			fmt.Fprintf(&b, "\t\t%s = %s.Get(%s)\n", md.VarName, mdVar, name)
		}
	case md.Slice:
		rawVar := md.VarName + "Raw"
		if md.Required {
			fmt.Fprintf(&b, "\t\tif %s := %s.Get(%s); len(%s) == 0 {\n", rawVar, mdVar, name, rawVar)
			fmt.Fprintf(&b, "\t\t\terr = loom.MergeErrors(err, loom.MissingFieldError(%s, \"metadata\"))\n", name)
			b.Add("\t\t} else {\n")
			b.Add(indent(renderGRPCSliceConversion(md, rawVar), 3))
			b.Add("\t\t}\n")
		} else {
			fmt.Fprintf(&b, "\t\tif %s := %s.Get(%s); len(%s) > 0 {\n", rawVar, mdVar, name, rawVar)
			b.Add(indent(renderGRPCSliceConversion(md, rawVar), 3))
			b.Add("\t\t}\n")
		}
	default:
		rawVar := md.VarName + "Raw"
		if md.Required {
			fmt.Fprintf(&b, "\t\tif vals := %s.Get(%s); len(vals) == 0 {\n", mdVar, name)
			fmt.Fprintf(&b, "\t\t\terr = loom.MergeErrors(err, loom.MissingFieldError(%s, \"metadata\"))\n", name)
			b.Add("\t\t} else {\n")
			fmt.Fprintf(&b, "\t\t\t%s = vals[0]\n", rawVar)
			b.Add("\n")
			b.Add(indent(renderGRPCStringParse(md, rawVar), 3))
			b.Add("\t\t}\n")
		} else {
			fmt.Fprintf(&b, "\t\tif vals := %s.Get(%s); len(vals) > 0 {\n", mdVar, name)
			fmt.Fprintf(&b, "\t\t\t%s = vals[0]\n", rawVar)
			b.Add("\n")
			b.Add(indent(renderGRPCStringParse(md, rawVar), 3))
			b.Add("\t\t}\n")
		}
	}
	return b.String()
}

func renderGRPCRequestMetadataDecode(md *MetadataData) string {
	var b sourceBuilder
	name := renderJen(jen.Lit(md.Name))
	switch {
	case md.TypeName == "string" || md.Type.Name() == "any":
		if md.Required {
			fmt.Fprintf(&b, "\t\tif vals := md.Get(%s); len(vals) == 0 {\n", name)
			fmt.Fprintf(&b, "\t\t\terr = loom.MergeErrors(err, loom.MissingFieldError(%s, \"metadata\"))\n", name)
			b.Add("\t\t} else {\n")
			fmt.Fprintf(&b, "\t\t\t%s = vals[0]\n", md.VarName)
			b.Add("\t\t}\n")
		} else {
			fmt.Fprintf(&b, "\t\tif vals := md.Get(%s); len(vals) > 0 {\n", name)
			if md.Pointer {
				fmt.Fprintf(&b, "\t\t\t%s = &vals[0]\n", md.VarName)
			} else {
				fmt.Fprintf(&b, "\t\t\t%s = vals[0]\n", md.VarName)
			}
			b.Add("\t\t}\n")
		}
	case md.StringSlice:
		if md.Required {
			fmt.Fprintf(&b, "\t\tif vals := md.Get(%s); len(vals) == 0 {\n", name)
			fmt.Fprintf(&b, "\t\t\terr = loom.MergeErrors(err, loom.MissingFieldError(%s, \"metadata\"))\n", name)
			b.Add("\t\t} else {\n")
			fmt.Fprintf(&b, "\t\t\t%s = vals\n", md.VarName)
			b.Add("\t\t}\n")
		} else {
			fmt.Fprintf(&b, "\t\t%s = md.Get(%s)\n", md.VarName, name)
		}
	case md.Slice:
		rawVar := md.VarName + "Raw"
		if md.Required {
			fmt.Fprintf(&b, "\t\tif %s := md.Get(%s); len(%s) == 0 {\n", rawVar, name, rawVar)
			fmt.Fprintf(&b, "\t\t\terr = loom.MergeErrors(err, loom.MissingFieldError(%s, \"metadata\"))\n", name)
			b.Add("\t\t} else {\n")
			b.Add(indent(renderGRPCSliceConversion(md, rawVar), 3))
			b.Add("\t\t}\n")
		} else {
			fmt.Fprintf(&b, "\t\tif %s := md.Get(%s); len(%s) > 0 {\n", rawVar, name, rawVar)
			b.Add(indent(renderGRPCSliceConversion(md, rawVar), 3))
			b.Add("\t\t}\n")
		}
	default:
		rawVar := md.VarName + "Raw"
		if md.Required {
			fmt.Fprintf(&b, "\t\tif vals := md.Get(%s); len(vals) == 0 {\n", name)
			fmt.Fprintf(&b, "\t\t\terr = loom.MergeErrors(err, loom.MissingFieldError(%s, \"metadata\"))\n", name)
			b.Add("\t\t} else {\n")
			fmt.Fprintf(&b, "\t\t\t%s := vals[0]\n", rawVar)
			b.Add("\n")
			b.Add(indent(renderGRPCStringParse(md, rawVar), 3))
			b.Add("\t\t}\n")
		} else {
			fmt.Fprintf(&b, "\t\tif vals := md.Get(%s); len(vals) > 0 {\n", name)
			fmt.Fprintf(&b, "\t\t\t%s := vals[0]\n", rawVar)
			b.Add("\n")
			b.Add(indent(renderGRPCStringParse(md, rawVar), 3))
			b.Add("\t\t}\n")
		}
	}
	return b.String()
}

func renderGRPCMetadataEncode(md *MetadataData, targetVar string) string {
	var b sourceBuilder
	switch {
	case md.StringSlice:
		fmt.Fprintf(&b, "\t%s.Append(%q, res.%s...)\n", targetVar, md.Name, md.FieldName)
	case md.Slice:
		fmt.Fprintf(&b, "\tfor _, value := range res.%s {\n", md.FieldName)
		b.Add(indent(renderGRPCStringConversion(expr.AsArray(md.Type).ElemType.Type, "valueStr", "value"), 2))
		fmt.Fprintf(&b, "\t\t%s.Append(%q, valueStr)\n", targetVar, md.Name)
		b.Add("\t}\n")
	default:
		if md.Pointer {
			fmt.Fprintf(&b, "\tif res.%s != nil {\n", md.FieldName)
		}
		fmt.Fprintf(&b, "\t\t%s.Append(%q, %s)\n", targetVar, md.Name, renderTemplateMetadataValue(md))
		if md.Pointer {
			b.Add("\t}\n")
		}
	}
	return b.String()
}

func renderMetadataSingleValue(md *MetadataData, value string) string {
	switch md.Type.Name() {
	case "bytes":
		return "string(" + pointerPrefix(md.Pointer) + value + ")"
	case "string":
		return pointerPrefix(md.Pointer) + value
	default:
		return renderJen(jen.Qual("fmt", "Sprintf").Call(jen.Lit("%v"), exprCode(pointerPrefix(md.Pointer)+value)))
	}
}

func renderTemplateMetadataValue(md *MetadataData) string {
	switch md.Type.Name() {
	case "bytes":
		return "string(*p." + md.FieldName + ")"
	case "string":
		if md.Pointer {
			return "*p." + md.FieldName
		}
		return "p." + md.FieldName
	default:
		return renderJen(jen.Qual("fmt", "Sprintf").Call(jen.Lit("%v"), exprCode("*p."+md.FieldName)))
	}
}
