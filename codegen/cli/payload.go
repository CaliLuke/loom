package cli

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// PayloadBuilderSection builds the section that can be used to
// generate the payload builder code.
func PayloadBuilderSection(buildFunction *BuildFunctionData) codegen.Section {
	return codegen.NewJenniferSection("cli-build-payload", func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("%s builds the payload for the %s %s endpoint from CLI flags.", buildFunction.Name, buildFunction.ServiceName, buildFunction.MethodName))
		fn := stmt.Func().Id(buildFunction.Name).ParamsFunc(func(group *jen.Group) {
			for _, formal := range buildFunction.FormalParams {
				group.Id(formal).String()
			}
		}).Params(codegen.TypeRef(buildFunction.ResultType), jen.Error())
		fn.BlockFunc(func(group *jen.Group) {
			if buildFunction.CheckErr {
				group.Var().Err().Error()
			}
			for _, field := range buildFunction.Fields {
				if field.VarName == "" {
					continue
				}
				group.Var().Id(field.VarName).Add(codegen.TypeRef(field.TypeRef))
				group.Block(field.Init)
			}
			if buildFunction.PayloadInit != nil {
				if buildFunction.PayloadInit.Code != nil {
					group.Add(buildFunction.PayloadInit.Code)
					if buildFunction.PayloadInit.ReturnTypeAttribute != "" {
						group.Id("res").Op(":=").Op("&").Add(codegen.TypeRef(buildFunction.PayloadInit.ReturnTypeName)).ValuesFunc(func(values *jen.Group) {
							value := values.Id(buildFunction.PayloadInit.ReturnTypeAttribute).Op(":")
							if buildFunction.PayloadInit.ReturnTypeAttributePointer {
								value.Op("&")
							} else if buildFunction.PayloadInit.ReturnTypeAttributeUnionValue {
								value.Op("*")
							}
							value.Id("v")
						})
					}
				}
				if buildFunction.PayloadInit.ReturnIsStruct {
					if buildFunction.PayloadInit.Code == nil {
						target := "v"
						if buildFunction.PayloadInit.ReturnTypeAttribute != "" {
							target = "res"
						}
						group.Id(target).Op(":=").Op("&").Add(codegen.TypeRef(buildFunction.PayloadInit.ReturnTypeName)).Values()
					}
					group.Add(fieldCode(buildFunction.PayloadInit))
				}
				resultVar := "v"
				if buildFunction.PayloadInit.ReturnTypeAttribute != "" {
					resultVar = "res"
				}
				group.Return(codegen.Expr(resultVar), jen.Nil())
			}
		})
	})
}

// NewFlagData creates a new FlagData from the given argument attributes.
//
// svcn is the service name
// en is the endpoint name
// name is the flag name
// typeName is the flag type
// description is the flag description
// required determines if the flag is required
// example is an example value for the flag
func NewFlagData(svcn, en, name, typeName, description string, required bool, example, def any) *FlagData {
	ex := jsonExample(example)
	fn := goifyTerms(svcn, en, name)
	return &FlagData{
		Name:        codegen.KebabCase(name),
		VarName:     codegen.Goify(name, false),
		Type:        flagType(typeName),
		FullName:    fn,
		Description: description,
		Required:    required,
		Example:     ex,
		Default:     def,
	}
}

// FieldLoadCode returns the code used in the build payload function that
// initializes one of the payload object fields. It returns the initialization
// code and a boolean indicating whether the code requires an "err" variable.
func FieldLoadCode(f *FlagData, argName, argTypeName, validate string, defaultValue any, payload expr.DataType, payloadRef string) (*jen.Statement, bool) {
	var (
		code    *jen.Statement
		declErr bool
	)
	if argTypeName == codegen.GoNativeTypeName(expr.String) {
		code = codegen.Expr(argName + " = " + fieldLoadStringPrefix(f, defaultValue) + f.FullName)
		declErr = validate != ""
	} else {
		var checkErr bool
		code, declErr, checkErr = conversionCode(f.FullName, argName, argTypeName, !f.Required && defaultValue == nil)
		if checkErr {
			code.Line().If(jen.Err().Op("!=").Nil()).Block(buildFieldLoadConversionError(f, argName, argTypeName, payload, payloadRef))
		}
	}
	if validate != "" {
		validate = stripNilGuardValidation(validate, argName)
		code.Line().Add(codegen.Expr(validate)).Line()
		nilVal, declareZero := fieldLoadReturnZero(payload, payloadRef)
		if declareZero != "" {
			code.Add(codegen.Expr(declareZero)).Line()
		}
		code.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(codegen.Expr(nilVal), jen.Err()),
		)
	}
	if !f.Required {
		return jen.If(codegen.Expr(f.FullName).Op("!=").Lit("")).Block(code), declErr
	}
	return code, declErr
}

func fieldLoadStringPrefix(f *FlagData, defaultValue any) string {
	if f.Required || defaultValue != nil {
		return ""
	}
	return "&"
}

func buildFieldLoadConversionError(f *FlagData, argName, argTypeName string, payload expr.DataType, payloadRef string) *jen.Statement {
	nilVal, declareZero := fieldLoadReturnZero(payload, payloadRef)
	stmt := new(jen.Statement)
	if declareZero != "" {
		stmt.Add(codegen.Expr(declareZero)).Line()
	}
	stmt.Add(buildFieldLoadErrorReturn(f, argName, argTypeName, nilVal))
	return stmt
}

func buildFieldLoadErrorReturn(f *FlagData, argName, argTypeName, nilVal string) *jen.Statement {
	if flagType(argTypeName) == "JSON" {
		return jen.Return(
			codegen.Expr(nilVal),
			jen.Qual("fmt", "Errorf").Call(
				jen.Lit("invalid JSON for "+argName+", \nerror: %s, \nexample of valid JSON:\n%s"),
				jen.Err(),
				jen.Lit(f.Example),
			),
		)
	}
	return jen.Return(
		codegen.Expr(nilVal),
		jen.Qual("fmt", "Errorf").Call(
			jen.Lit("invalid value for "+argName+", must be "+f.Type),
		),
	)
}

func stripNilGuardValidation(validate, argName string) string {
	nilCheck := "if " + argName + " != nil {"
	if !strings.HasPrefix(validate, nilCheck) {
		return validate
	}
	lines := make([]string, 0, strings.Count(validate, "\n"))
	ls := strings.Split(validate, "\n")
	for i := 1; i < len(ls)-1; i++ {
		if ls[i+1] == nilCheck {
			i++
			continue
		}
		lines = append(lines, ls[i])
	}
	return strings.Join(lines, "\n")
}

func fieldLoadReturnZero(payload expr.DataType, payloadRef string) (nilVal, declareZero string) {
	if !expr.IsPrimitive(payload) {
		return "nil", ""
	}
	return "zero", "var zero " + payloadRef
}

func generateExample(sub *SubcommandData, svc string) {
	ex := codegen.KebabCase(svc) + " " + codegen.KebabCase(sub.Name)
	for _, f := range sub.Flags {
		ex += " --" + f.Name + " " + f.Example
	}
	sub.Example = ex
}

// fieldCode generates code to initialize the data structures fields
// from the given args. It is used only in templates.
func fieldCode(init *PayloadInitData) *jen.Statement {
	varn := "res"
	if init.ReturnTypeAttribute == "" {
		varn = "v"
	}
	// We can ignore the transform helpers as there won't be any generated
	// because the args cannot be user types.
	c, _, err := codegen.InitStructFields(init.Args, varn, "", init.ReturnTypePkg)
	if err != nil {
		panic(fmt.Errorf("build CLI payload field init for %s: %w", init.ReturnTypeName, err))
	}
	return codegen.Expr(c)
}
