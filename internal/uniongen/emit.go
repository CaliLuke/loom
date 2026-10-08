//nolint:errcheck // Generator helpers write only to in-memory builders.
package uniongen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/dave/jennifer/jen"
)

// Emit appends the union declaration, constructors, accessors and codecs to stmt.
func Emit(stmt *jen.Statement, data *Type) {
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

func addUnionAliasTypes(stmt *jen.Statement, data *Type) {
	for _, field := range data.Fields {
		if !field.EmitPrimitiveAlias {
			continue
		}
		decl := stmt.Type().Id(field.FieldType)
		if field.PrimitiveAliasType == "loom.JSONValue" {
			decl.Op("=")
		}
		decl.Add(codegen.Expr(field.PrimitiveAliasType))
		stmt.Line()
	}
}

func addUnionStructType(stmt *jen.Statement, data *Type) {
	codegen.Doc(stmt, fmt.Sprintf("%s is a sum-type union.", data.Name))
	stmt.Type().Id(data.Name).StructFunc(func(group *jen.Group) {
		group.Id("kind").Id(data.KindName)
		for _, field := range data.Fields {
			group.Id(field.FieldName).Id(field.FieldType)
		}
	})
	stmt.Line()
}

func addUnionKindType(stmt *jen.Statement, data *Type) {
	codegen.Doc(stmt, fmt.Sprintf("%s enumerates the union variants for %s.", data.KindName, data.Name))
	stmt.Type().Id(data.KindName).String()
	stmt.Line()
}

func addUnionKindConsts(stmt *jen.Statement, data *Type) {
	stmt.Const().DefsFunc(func(group *jen.Group) {
		for _, field := range data.Fields {
			group.Comment(codegen.LineComment(fmt.Sprintf("%s identifies the %s branch of the union.", field.KindConst, field.Name)))
			group.Id(field.KindConst).Id(data.KindName).Op("=").Lit(field.TypeTag)
		}
	})
	stmt.Line()
}

func addUnionKindMethod(stmt *jen.Statement, data *Type) {
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

func addUnionVariantMethods(stmt *jen.Statement, data *Type) {
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

func addUnionValidateMethod(stmt *jen.Statement, data *Type) {
	codegen.Doc(stmt, "Validate ensures the union discriminant is valid.")
	stmt.Func().
		Params(jen.Id("u").Id(data.Name)).
		Id("Validate").
		Params().
		Error().
		BlockFunc(func(group *jen.Group) {
			addRawUnionBlock(group, ValidateBody(data))
		})
	stmt.Line()
}

func addUnionMarshalJSONMethod(stmt *jen.Statement, data *Type) {
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
			addRawUnionBlock(group, MarshalJSONBody(data))
		})
	stmt.Line()
}

func addUnionMarshalFormMethod(stmt *jen.Statement, data *Type) {
	addUnionFormMethodComment(stmt, "MarshalFormValues", "marshals")
	stmt.Func().
		Params(jen.Id("u").Id(data.Name)).
		Id("MarshalFormValues").
		Params(jen.Id("values").Qual("net/url", "Values"), jen.Id("prefix").String()).
		Error().
		BlockFunc(func(group *jen.Group) {
			addRawUnionBlock(group, MarshalFormBody(data))
		})
	stmt.Line()
}

func addUnionUnmarshalFormMethod(stmt *jen.Statement, data *Type) {
	addUnionFormMethodComment(stmt, "UnmarshalFormValues", "unmarshals")
	stmt.Func().
		Params(jen.Id("u").Op("*").Id(data.Name)).
		Id("UnmarshalFormValues").
		Params(jen.Id("values").Qual("net/url", "Values"), jen.Id("prefix").String()).
		Error().
		BlockFunc(func(group *jen.Group) {
			addRawUnionBlock(group, UnmarshalFormBody(data))
		})
	stmt.Line()
}

func addUnionUnmarshalJSONMethod(stmt *jen.Statement, data *Type) {
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
			addRawUnionBlock(group, UnmarshalJSONBody(data))
		})
	stmt.Line()
	if data.Untagged {
		stmt.Add(codegen.Expr(data.JSON.Matcher())).Line()
	}
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
