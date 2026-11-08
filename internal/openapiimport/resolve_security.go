package openapiimport

import (
	"fmt"
	"slices"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/base"
)

type resolvedSecurityKey struct {
	name string
	edge *edge
}

func (a *analyzer) projectSecurityKeys(source []*base.SecurityRequirement, path string, components Components) SecurityRequirements {
	requirements := make(SecurityRequirements, 0, len(source))
	for index, sourceRequirement := range source {
		requirementPath := fmt.Sprintf("%s/%d", path, index)
		if sourceRequirement == nil {
			a.unsupported("security-requirement", requirementPath, "security requirement is nil")
			continue
		}
		requirement := SecurityRequirement{}
		positions := make(map[string]int)
		for authored, scopes := range sourceRequirement.Requirements.FromOldest() {
			schemePath := requirementPath + "/" + escapeJSONPointer(authored)
			name := authored
			if a.bundle != nil {
				if _, failed := a.bundle.keyFailures[schemePath]; failed {
					continue
				}
				if resolved, ok := a.bundle.securityKeys[schemePath]; ok {
					name = resolved.name
				}
			}
			if securitySchemeByName(components.SecuritySchemes, name) == nil {
				a.unsupported("security-requirement", schemePath, fmt.Sprintf("security scheme %q is not a supported component", name))
				continue
			}
			position, seen := positions[name]
			if !seen {
				position = len(requirement.Schemes)
				positions[name] = position
				requirement.Schemes = append(requirement.Schemes, SecurityRequirementScheme{Name: name})
			}
			target := &requirement.Schemes[position]
			target.Scopes = append(target.Scopes, scopes...)
			slices.Sort(target.Scopes)
			target.Scopes = slices.Compact(target.Scopes)
		}
		requirements = append(requirements, requirement)
	}
	return requirements
}

func (b *bundledDescription) projectSecurityMetadata() error {
	b.securityURIs = make(map[string]string)
	groups := make(map[targetKey][]resolvedSecurityKey)
	for _, key := range b.securityKeys {
		if key.edge.target != nil && key.edge.target.doc != "entry" {
			groups[*key.edge.target] = append(groups[*key.edge.target], key)
		}
	}
	for target, keys := range groups {
		document := b.graph.registry.docs[target.doc]
		uri, err := parseDescriptionURI(string(document.canonical))
		if err != nil {
			return err
		}
		if uri.Scheme == "file" {
			continue
		}
		canonical := string(document.canonical) + "#" + emitPointer(target.pointer)
		spellings := make(map[string]bool)
		for _, key := range keys {
			spellings[key.edge.authored] = true
		}
		if len(spellings) == 1 && b.securitySpellingStable(keys[0].edge.authored, target) {
			canonical = keys[0].edge.authored
		}
		b.securityURIs[keys[0].name] = canonical
	}
	return nil
}

func (b *bundledDescription) securitySpellingStable(authored string, target targetKey) bool {
	uri, err := parseDescriptionURI(authored)
	if err != nil {
		return false
	}
	entry := b.graph.registry.docs["entry"]
	var regeneratedBase docURI
	if entry.hasSelf {
		self, err := parseDescriptionURI(entry.self)
		if err == nil && self.IsAbs() {
			regeneratedBase = entry.base
		}
	}
	if !uri.IsAbs() && regeneratedBase == "" {
		return false
	}
	resolved, pointer, err := resolveReference(authored, regeneratedBase)
	if err != nil || pointer != target.pointer {
		return false
	}
	return b.graph.registry.alias[resolved] == target.doc
}

func securityKeyFailure(entry *document, e *edge) *Diagnostic {
	if !strings.HasPrefix(entry.version, "3.2.") && e.entry == "" {
		message := "URI security requirement keys require OpenAPI 3.2"
		if descriptionName.MatchString(e.authored) && e.failure != nil {
			message = e.failure.Message
		}
		return &Diagnostic{Code: "security-requirement", Message: message}
	}
	return e.failure
}
