package openapiimport

import (
	"encoding/json/jsontext"
	"slices"
	"sort"
	"strings"
)

func (b *bundledDescription) indexDeclarationOrigins() {
	b.declarations = make(map[string]declarationOrigin)
	var paths []string
	for path, origin := range b.origins {
		if origin.Document != "entry" || strings.HasPrefix(path, "#/components/") && origin.Pointer != path {
			paths = append(paths, path)
		}
	}
	sort.Slice(paths, func(i, j int) bool {
		x, y := b.origins[paths[i]], b.origins[paths[j]]
		xEntry, yEntry := x.Document == "entry", y.Document == "entry"
		if xEntry != yEntry {
			return xEntry
		}
		xRoot, yRoot := strings.HasPrefix(x.Document, "root:"), strings.HasPrefix(y.Document, "root:")
		if xRoot != yRoot {
			return xRoot
		}
		if x.Document != y.Document {
			return x.Document < y.Document
		}
		xTokens := slices.Collect(jsontext.Pointer(strings.TrimPrefix(x.Pointer, "#")).Tokens())
		yTokens := slices.Collect(jsontext.Pointer(strings.TrimPrefix(y.Pointer, "#")).Tokens())
		if comparison := slices.Compare(xTokens, yTokens); comparison != 0 {
			return comparison < 0
		}
		return paths[i] < paths[j]
	})
	for i, path := range paths {
		b.declarations[path] = declarationOrigin{introduced: true, order: i + 1}
	}
}

func (b *bundledDescription) declarationOrigin(pointer string) declarationOrigin {
	if b == nil {
		return declarationOrigin{}
	}
	for pointer != "" {
		if origin, ok := b.declarations[pointer]; ok {
			return origin
		}
		if _, ok := b.origins[pointer]; ok {
			return declarationOrigin{}
		}
		index := strings.LastIndex(pointer, "/")
		if index < 0 {
			break
		}
		pointer = pointer[:index]
	}
	return declarationOrigin{}
}

func (b *bundledDescription) changed() bool {
	if len(b.graph.registry.docs) > 1 || len(b.blocked) > 0 {
		return true
	}
	for pointer, origin := range b.origins {
		if pointer != origin.Pointer {
			return true
		}
	}
	for _, edge := range b.graph.edges {
		if edge.class == "pathItem" || edge.failure != nil || edge.external != "" {
			return true
		}
		if edge.target != nil && edge.authored != "#"+emitPointer(edge.target.pointer) {
			return true
		}
	}
	return false
}

func (b *bundledDescription) attachOrigins(diagnostics Diagnostics) {
	for i := range diagnostics {
		diagnostic := &diagnostics[i]
		if diagnostic.Origin != nil {
			continue
		}
		pointer := diagnostic.Path
		for pointer != "" {
			if origin, ok := b.origins[pointer]; ok {
				if origin.Document != "entry" || origin.Pointer != pointer {
					origin.Pointer += strings.TrimPrefix(diagnostic.Path, pointer)
					diagnostic.Origin = &origin
				}
				break
			}
			index := strings.LastIndex(pointer, "/")
			if index < 0 {
				break
			}
			pointer = pointer[:index]
		}
	}
}

// blockedDocumentDiagnostics uses the retained component inventory, not a name
// prefix. An authored component may legitimately be named x-loom-unresolved-1.
func blockedDocumentDiagnostics(document *Document) Diagnostics {
	if document.bundle == nil {
		return nil
	}
	var diagnostics Diagnostics
	for _, diagnostic := range document.bundle.blocked {
		diagnostics = append(diagnostics, diagnostic)
	}
	if document.retainedClosure != nil {
		diagnostics = filterSelectionDiagnostics(diagnostics, *document.retainedClosure)
	}
	for _, path := range document.unresolvedPaths {
		for _, diagnostic := range document.bundle.diagnostics {
			if diagnostic.Code == "path-reference" && diagnostic.Path == "#/paths/"+escapeJSONPointer(path) {
				diagnostics = append(diagnostics, diagnostic)
			}
		}
	}
	for _, diagnostic := range document.bundle.keyFailures {
		if securityFailureSelected(document, diagnostic.Path) {
			diagnostics = append(diagnostics, diagnostic)
		}
	}
	return diagnostics.sorted()
}

func securityFailureSelected(document *Document, pointer string) bool {
	if strings.HasPrefix(pointer, "#/security/") {
		return document.SecurityDefined
	}
	for _, operation := range document.Operations {
		prefix := operation.sourcePath + "/security/"
		if strings.HasPrefix(pointer, prefix) {
			return true
		}
	}
	return false
}

func filterSecurityFailures(document *Document, diagnostics Diagnostics) Diagnostics {
	if document.bundle == nil {
		return diagnostics
	}
	filtered := make(Diagnostics, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		if _, failure := document.bundle.keyFailures[diagnostic.Path]; !failure || securityFailureSelected(document, diagnostic.Path) {
			filtered = append(filtered, diagnostic)
		}
	}
	return filtered
}
