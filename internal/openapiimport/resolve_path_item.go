package openapiimport

import (
	"fmt"

	"github.com/pb33f/go-yaml"
)

type pathItemField struct {
	name   string
	node   *yaml.Node
	origin targetKey
}

func effectivePathItem(r *resolution, target targetKey) ([]pathItemField, *Diagnostic, error) {
	var fields []pathItemField
	seenTargets := make(map[targetKey]bool)
	seenFields := make(map[string]bool)
	fail := func(at targetKey, message string) ([]pathItemField, *Diagnostic, error) {
		origin := SourceLocation{Document: r.registry.docs[at.doc].portable, Pointer: "#" + at.pointer}
		return nil, &Diagnostic{Code: "path-reference", Path: "#" + target.pointer, Origin: &origin, Message: message}, nil
	}
	current := target
	for {
		if seenTargets[current] {
			return fail(current, "Path Item reference cycle")
		}
		seenTargets[current] = true
		if len(seenTargets) > maxPathItemObjects {
			return nil, nil, errResolutionBudget
		}
		node, err := descriptionNode(r.registry.docs[current.doc].root, current.pointer)
		if err != nil {
			return fail(current, err.Error())
		}
		if node.Kind != yaml.MappingNode {
			return fail(current, "Path Item must be an object")
		}
		var reference *edge
		for i := 0; i+1 < len(node.Content); i += 2 {
			name := node.Content[i].Value
			if name == "$ref" {
				for _, e := range r.edges {
					if e.class == "pathItem" && e.at == (targetKey{current.doc, current.pointer + "/$ref"}) {
						reference = e
						break
					}
				}
				if reference == nil {
					return fail(current, "Path Item reference has no resolution outcome")
				}
				continue
			}
			if seenFields[name] {
				return fail(current, fmt.Sprintf("field %q appears in both the referencing and referenced Path Item", name))
			}
			seenFields[name] = true
			fields = append(fields, pathItemField{name: name, node: node.Content[i+1], origin: targetKey{current.doc, current.pointer + "/" + escapeJSONPointer(name)}})
		}
		if reference == nil {
			return fields, nil, nil
		}
		if reference.failure != nil {
			return fail(reference.at, reference.failure.Message)
		}
		if reference.target == nil {
			return fail(reference.at, "Path Item reference has no target")
		}
		current = *reference.target
	}
}
