package service

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

type (
	// convertData contains the info needed to render convert and create
	// functions.
	convertData struct {
		// Name is the name of the function.
		Name string
		// ReceiverTypeRef is a reference to the receiver type.
		ReceiverTypeRef string
		// TypeRef is a reference to the external type.
		TypeRef string
		// TypeName is the name of the external type.
		TypeName string
		// Code is the function body.
		Code string
		// ErrorAware marks conversion that can fail before publishing its result.
		ErrorAware bool
	}

	convertDataBuilder func(*expr.TypeMap, map[string]struct{}) (convertData, []*codegen.TransformFunctionData, error)
)

func buildConvertFile(
	convertPath string,
	serviceName string,
	packageName string,
	conversions []*expr.TypeMap,
	creations []*expr.TypeMap,
	buildConvert convertDataBuilder,
	buildCreate convertDataBuilder,
) (*codegen.File, error) {
	if len(conversions) == 0 && len(creations) == 0 {
		return nil, nil
	}

	pkgs, err := convertImports(conversions, creations)
	if err != nil {
		return nil, err
	}

	sections := []codegen.Section{
		codegen.Header(serviceName+" service type conversion functions", packageName, pkgs),
	}
	names := map[string]struct{}{}

	sections, transformHelpers, err := appendConvertSections(
		sections,
		nil,
		conversions,
		names,
		buildConvert,
		func(data convertData) codegen.Section {
			return convertSection("convert-to", data)
		},
	)
	if err != nil {
		return nil, err
	}
	sections, transformHelpers, err = appendConvertSections(
		sections,
		transformHelpers,
		creations,
		names,
		buildCreate,
		func(data convertData) codegen.Section {
			return createSection("create-from", data)
		},
	)
	if err != nil {
		return nil, err
	}
	sections = appendTransformHelperSections(sections, transformHelpers)

	return &codegen.File{Path: convertPath, Sections: sections}, nil
}

func appendConvertSections(
	sections []codegen.Section,
	transformHelpers []*codegen.TransformFunctionData,
	typeMaps []*expr.TypeMap,
	names map[string]struct{},
	buildData convertDataBuilder,
	renderSection func(convertData) codegen.Section,
) ([]codegen.Section, []*codegen.TransformFunctionData, error) {
	for _, tm := range typeMaps {
		data, helpers, err := buildData(tm, names)
		if err != nil {
			return nil, nil, err
		}
		transformHelpers = codegen.AppendHelpers(transformHelpers, helpers)
		sections = append(sections, renderSection(data))
	}

	return sections, transformHelpers, nil
}

func appendTransformHelperSections(sections []codegen.Section, transformHelpers []*codegen.TransformFunctionData) []codegen.Section {
	seen := make(map[string]struct{}, len(transformHelpers))
	for _, tf := range transformHelpers {
		if _, ok := seen[tf.Name]; ok {
			continue
		}
		seen[tf.Name] = struct{}{}
		sections = append(sections, transformHelperSection("convert-create-helper", tf))
	}

	return sections
}

func convertSection(name string, data convertData) codegen.Section {
	return codegen.NewJenniferSection(name, func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("%s creates an instance of %s initialized from t.", data.Name, data.TypeName))
		declaration := stmt.Func().Params(jen.Id("t").Add(codegen.TypeRef(data.ReceiverTypeRef))).Id(data.Name).Params()
		if data.ErrorAware {
			declaration.Params(codegen.TypeRef(data.TypeRef), jen.Error())
		} else {
			declaration.Add(codegen.TypeRef(data.TypeRef))
		}
		declaration.BlockFunc(func(group *jen.Group) {
			if data.ErrorAware {
				group.Id("transformErr").Op(":=").New(jen.Error())
			}
			appendConvertRawBlock(group, data.Code)
			group.Line()
			if data.ErrorAware {
				group.If(jen.Op("*").Id("transformErr").Op("!=").Nil()).Block(
					jen.Var().Id("zero").Add(codegen.TypeRef(data.TypeRef)),
					jen.Return(jen.Id("zero"), jen.Op("*").Id("transformErr")),
				)
				group.Return(jen.Id("v"), jen.Nil())
			} else {
				group.Return(jen.Id("v"))
			}
		})
	})
}

func createSection(name string, data convertData) codegen.Section {
	return codegen.NewJenniferSection(name, func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("%s initializes t from the fields of v", data.Name))
		declaration := stmt.Func().Params(jen.Id("t").Add(codegen.TypeRef(data.ReceiverTypeRef))).Id(data.Name).Params(jen.Id("v").Add(codegen.TypeRef(data.TypeRef)))
		if data.ErrorAware {
			declaration.Error()
		}
		declaration.BlockFunc(func(group *jen.Group) {
			if data.ErrorAware {
				group.Id("transformErr").Op(":=").New(jen.Error())
			}
			appendConvertRawBlock(group, data.Code)
			if data.ErrorAware {
				group.If(jen.Op("*").Id("transformErr").Op("!=").Nil()).Block(jen.Return(jen.Op("*").Id("transformErr")))
			}
			group.Op("*").Id("t").Op("=").Op("*").Id("temp")
			if data.ErrorAware {
				group.Return(jen.Nil())
			}
		})
	})
}

func transformHelperSection(name string, data *codegen.TransformFunctionData) codegen.Section {
	return codegen.NewJenniferSection(name, func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("%s builds a value of type %s from a value of type %s.", data.Name, data.ResultTypeRef, data.ParamTypeRef))
		stmt.Func().Id(data.Name).ParamsFunc(func(params *jen.Group) {
			params.Id("v").Add(codegen.TypeRef(data.ParamTypeRef))
			if data.ErrorAware {
				params.Id("transformErr").Op("*").Error()
			}
		}).Add(codegen.TypeRef(data.ResultTypeRef)).BlockFunc(func(group *jen.Group) {
			appendConvertRawBlock(group, data.Code)
			group.Line()
			group.Return(jen.Id("res"))
		})
		stmt.Line()
	})
}

func appendConvertRawBlock(group *jen.Group, code string) {
	if strings.TrimSpace(code) == "" {
		return
	}
	if strings.HasPrefix(code, "\n") {
		group.Line()
	}
	group.Add(codegen.Expr(strings.TrimSpace(code)))
}
