package openapiimport

import (
	"encoding/json/jsontext"
	"slices"
	"strings"
)

func (p *componentPruner) visitExamples(examples []Example) {
	for _, example := range examples {
		if example.ComponentName != "" {
			p.closure.examples[example.ComponentName] = struct{}{}
		}
	}
}

func (p *componentPruner) visitSecurity(requirements SecurityRequirements) {
	for _, requirement := range requirements {
		for _, scheme := range requirement.Schemes {
			p.closure.securitySchemes[scheme.Name] = struct{}{}
		}
	}
}

// visitSourceOperation keeps grammar dependencies even when a source-only
// admission diagnostic prevented a normalized value from being constructed.
// In particular, a discarded example must not hide an unresolved dependency.
// A component retained here still has its normalized body walked.
func (p *componentPruner) visitSourceOperation(bundle *bundledDescription, operation Operation) {
	base := "#/paths/" + escapeJSONPointer(operation.Path)
	visited := make(map[string]bool)
	p.visitSourcePrefix(bundle, operation.sourcePath, visited)
	p.visitSourcePrefix(bundle, base+"/parameters", visited)
}

func (p *componentPruner) visitSourcePrefix(bundle *bundledDescription, prefix string, visited map[string]bool) {
	if visited[prefix] {
		return
	}
	visited[prefix] = true
	for occurrence, target := range bundle.references {
		if occurrence != prefix && !strings.HasPrefix(occurrence, prefix+"/") {
			continue
		}
		tokens := slices.Collect(jsontext.Pointer(strings.TrimPrefix(target, "#")).Tokens())
		if len(tokens) != 3 || tokens[0] != "components" || !p.retainComponent(tokens[1], tokens[2]) {
			continue
		}
		p.visitSourcePrefix(bundle, target, visited)
	}
}

func filterUnresolvedPathDiagnostics(diagnostics Diagnostics, paths []string) Diagnostics {
	retained := make(map[string]bool, len(paths))
	for _, path := range paths {
		retained["#/paths/"+escapeJSONPointer(path)] = true
	}
	result := make(Diagnostics, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		if diagnostic.Code != "path-reference" || retained[diagnostic.Path] {
			result = append(result, diagnostic)
		}
	}
	return result
}
