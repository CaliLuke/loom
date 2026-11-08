package openapiimport

import (
	"encoding/json/jsontext"
	"slices"
	"strings"
)

func (p *componentPruner) visitExamples(examples []Example) {
	for _, example := range examples {
		if example.ComponentName != "" {
			p.retainComponent("examples", example.ComponentName)
		}
	}
}

func (p *componentPruner) visitSecurity(requirements SecurityRequirements) {
	for _, requirement := range requirements {
		for _, scheme := range requirement.Schemes {
			p.retainComponent("securitySchemes", scheme.Name)
		}
	}
}

// visitSourceOperation keeps grammar dependencies even when a source-only
// admission diagnostic prevented a normalized value from being constructed.
// In particular, a discarded example must not hide an unresolved dependency.
// Components retained here have their normalized body and raw source walked.
func (p *componentPruner) visitSourceOperation(bundle *bundledDescription, operation Operation) {
	base := "#/paths/" + escapeJSONPointer(operation.Path)
	p.visitSourcePrefix(bundle, operation.sourcePath, p.sources)
	p.visitSourcePrefix(bundle, base+"/parameters", p.sources)
}

func (p *componentPruner) visitSourcePrefix(bundle *bundledDescription, prefix string, visited map[string]bool) {
	if visited[prefix] {
		return
	}
	visited[prefix] = true
	for _, target := range sourceTargets(bundle, prefix) {
		tokens := slices.Collect(jsontext.Pointer(strings.TrimPrefix(target, "#")).Tokens())
		if len(tokens) != 3 || tokens[0] != "components" || !p.retainComponent(tokens[1], tokens[2]) {
			continue
		}
		p.visitSourcePrefix(bundle, target, visited)
	}
}

// indexComponentReferences groups the bundle's finalized references by the
// component prefix that owns each occurrence. bundleResolution calls it once,
// after the last reference is recorded. The index is never written afterwards,
// so every pruner and projection copy of one analysis shares it without
// synchronization. A hand-built bundle must call it before its component
// prefixes are walked.
func (b *bundledDescription) indexComponentReferences() {
	b.componentReferences = make(map[string][]string)
	for occurrence, target := range b.references {
		tokens := slices.Collect(jsontext.Pointer(strings.TrimPrefix(occurrence, "#")).Tokens())
		if len(tokens) >= 3 && tokens[0] == "components" {
			key := componentSource(tokens[1], tokens[2])
			b.componentReferences[key] = append(b.componentReferences[key], target)
		}
	}
}

// sourceTargets returns the targets of bundled references at or beneath
// prefix. A component prefix is answered from the bundle's finalized index, so
// walking every retained component's raw source never rescans the references.
func sourceTargets(bundle *bundledDescription, prefix string) []string {
	tokens := slices.Collect(jsontext.Pointer(strings.TrimPrefix(prefix, "#")).Tokens())
	if len(tokens) == 3 && tokens[0] == "components" {
		return bundle.componentReferences[componentSource(tokens[1], tokens[2])]
	}
	var targets []string
	for occurrence, target := range bundle.references {
		if occurrence == prefix || strings.HasPrefix(occurrence, prefix+"/") {
			targets = append(targets, target)
		}
	}
	return targets
}

// componentSource is the bundled source prefix that owns a component.
func componentSource(kind, name string) string {
	return "#/components/" + kind + "/" + escapeJSONPointer(name)
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
