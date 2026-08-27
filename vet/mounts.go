package vet

import (
	"fmt"
	"go/ast"
	"go/types"
	"os"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

type designedHTTPService struct {
	name       string
	path       string
	suppressed bool
}

const httpEntrypointMeta = "loom:vet:http-entrypoint"

func analyzeConfiguredHTTPMounts(moduleRoot string, design *expr.RootExpr) ([]Diagnostic, error) {
	entrypoints := configuredHTTPEntrypoints(design)
	if len(entrypoints) == 0 {
		return nil, nil
	}
	services := designedHTTPServices(design)
	if len(services) == 0 {
		return nil, nil
	}
	loaded, err := loadHTTPEntrypoints(moduleRoot, entrypoints)
	if err != nil {
		return nil, err
	}
	mounted, err := mountedHTTPServicePaths(loaded)
	if err != nil {
		return nil, err
	}
	var diagnostics []Diagnostic
	for _, service := range services {
		if service.suppressed {
			continue
		}
		if _, exists := mounted[service.path]; exists {
			continue
		}
		diagnostics = append(diagnostics, Diagnostic{
			Rule:     RuleServiceNotMounted,
			Severity: SeverityError,
			Message: fmt.Sprintf(
				"HTTP service %q is not mounted in any configured entrypoint package; call its generated server Mount function or suppress service-not-mounted on the service",
				service.name,
			),
			Location: Location{Path: "service." + service.name + ".http"},
		})
	}
	return diagnostics, nil
}

func configuredHTTPEntrypoints(design *expr.RootExpr) []string {
	if design == nil || design.API == nil {
		return nil
	}
	seen := make(map[string]struct{})
	entrypoints := make([]string, 0, len(design.API.Meta[httpEntrypointMeta]))
	for _, configured := range design.API.Meta[httpEntrypointMeta] {
		configured = strings.TrimSpace(configured)
		if configured == "" {
			continue
		}
		if _, exists := seen[configured]; exists {
			continue
		}
		seen[configured] = struct{}{}
		entrypoints = append(entrypoints, configured)
	}
	sort.Strings(entrypoints)
	return entrypoints
}

func designedHTTPServices(design *expr.RootExpr) []designedHTTPService {
	if design == nil || design.API == nil || design.API.HTTP == nil {
		return nil
	}
	services := make([]designedHTTPService, 0, len(design.API.HTTP.Services))
	for _, service := range design.API.HTTP.Services {
		if service == nil || service.ServiceExpr == nil {
			continue
		}
		if len(service.HTTPEndpoints) == 0 && len(service.FileServers) == 0 {
			continue
		}
		services = append(services, designedHTTPService{
			name: service.Name(),
			path: codegen.SnakeCase(codegen.Goify(service.Name(), false)),
			suppressed: suppressed(service.Meta, RuleServiceNotMounted) ||
				suppressed(service.ServiceExpr.Meta, RuleServiceNotMounted),
		})
	}
	return services
}

func loadHTTPEntrypoints(moduleRoot string, entrypoints []string) ([]*packages.Package, error) {
	loaded, err := packages.Load(&packages.Config{
		Dir: moduleRoot,
		Mode: packages.NeedName |
			packages.NeedCompiledGoFiles |
			packages.NeedModule |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo,
	}, entrypoints...)
	if err != nil {
		return nil, fmt.Errorf("load configured HTTP mount entrypoints: %w", err)
	}
	if len(loaded) == 0 {
		return nil, fmt.Errorf("configured HTTP mount entrypoints matched no packages: %s", strings.Join(entrypoints, ", "))
	}
	for _, pkg := range loaded {
		if !packageSyntaxUsable(pkg) {
			if len(pkg.Errors) > 0 {
				return nil, fmt.Errorf("load configured HTTP mount entrypoint %q: %s", pkg.PkgPath, pkg.Errors[0].Msg)
			}
			return nil, fmt.Errorf("configured HTTP mount entrypoint %q has no analyzable Go source", pkg.PkgPath)
		}
		if !packageBelongsToModule(moduleRoot, pkg) {
			return nil, fmt.Errorf("configured HTTP mount entrypoint %q is outside the target module", pkg.PkgPath)
		}
	}
	return loaded, nil
}

func mountedHTTPServicePaths(loaded []*packages.Package) (map[string]struct{}, error) {
	mounted := make(map[string]struct{})
	for _, pkg := range loaded {
		for index, file := range pkg.Syntax {
			if index >= len(pkg.CompiledGoFiles) {
				continue
			}
			source, err := os.ReadFile(pkg.CompiledGoFiles[index])
			if err != nil {
				return nil, fmt.Errorf("read configured HTTP mount package %s: %w", pkg.CompiledGoFiles[index], err)
			}
			if generatedByLoom(source) {
				continue
			}
			ast.Inspect(file, func(node ast.Node) bool {
				if servicePath, ok := generatedHTTPMountServicePath(pkg, node); ok {
					mounted[servicePath] = struct{}{}
				}
				return true
			})
		}
	}
	return mounted, nil
}

func generatedHTTPMountServicePath(pkg *packages.Package, node ast.Node) (string, bool) {
	call, ok := node.(*ast.CallExpr)
	if !ok {
		return "", false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Mount" {
		return "", false
	}
	function, _ := pkg.TypesInfo.Uses[selector.Sel].(*types.Func)
	if selection := pkg.TypesInfo.Selections[selector]; selection != nil {
		function, _ = selection.Obj().(*types.Func)
	}
	if function == nil || function.Pkg() == nil {
		return "", false
	}
	return servicePathFromHTTPServerPackage(function.Pkg().Path())
}

func servicePathFromHTTPServerPackage(packagePath string) (string, bool) {
	const marker = "/gen/http/"
	index := strings.LastIndex(packagePath, marker)
	if index < 0 {
		return "", false
	}
	parts := strings.Split(packagePath[index+len(marker):], "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "server" {
		return "", false
	}
	return parts[0], true
}
