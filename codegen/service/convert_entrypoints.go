package service

import (
	"path/filepath"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// ConvertFiles returns multiple files containing conversion and creation
// functions, grouped by target package as specified by struct:pkg:path
// metadata.
func ConvertFiles(root *expr.RootExpr, service *expr.ServiceExpr, services *ServicesData) ([]*codegen.File, error) {
	svc := services.Get(service.Name)
	conversions, creations := relevantTypeMaps(root, service, svc)
	if len(conversions) == 0 && len(creations) == 0 {
		return nil, nil
	}

	conversionsByPath, creationsByPath, allPaths := groupTypeMapsByPath(conversions, creations, service)
	files := make([]*codegen.File, 0, len(allPaths))
	for convertPath := range allPaths {
		packageName := convertPackageName(conversionsByPath[convertPath], creationsByPath[convertPath], svc.PkgName)
		file, err := buildConvertFile(
			convertPath,
			service.Name,
			packageName,
			conversionsByPath[convertPath],
			creationsByPath[convertPath],
			func(c *expr.TypeMap, names map[string]struct{}) (convertData, []*codegen.TransformFunctionData, error) {
				return buildConvertSectionData(c, svc, packageName, names)
			},
			func(c *expr.TypeMap, names map[string]struct{}) (convertData, []*codegen.TransformFunctionData, error) {
				return buildCreateSectionData(c, svc, packageName, names)
			},
		)
		if err != nil {
			return nil, err
		}
		if file != nil {
			files = append(files, file)
		}
	}

	return files, nil
}

// ConvertFile returns the legacy single convert.go file for the default service
// path when any conversions or creations are defined.
func ConvertFile(root *expr.RootExpr, service *expr.ServiceExpr, services *ServicesData) (*codegen.File, error) {
	svc := services.Get(service.Name)
	conversions, creations := relevantTypeMaps(root, service, svc)
	if len(conversions) == 0 && len(creations) == 0 {
		return nil, nil
	}

	return buildConvertFile(
		filepath.Join(codegen.Gendir, codegen.SnakeCase(service.Name), "convert.go"),
		service.Name,
		svc.PkgName,
		conversions,
		creations,
		func(c *expr.TypeMap, names map[string]struct{}) (convertData, []*codegen.TransformFunctionData, error) {
			return buildDefaultConvertSectionData(c, svc, names)
		},
		func(c *expr.TypeMap, names map[string]struct{}) (convertData, []*codegen.TransformFunctionData, error) {
			return buildDefaultCreateSectionData(c, svc, names)
		},
	)
}
