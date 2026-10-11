package ir

import (
	"fmt"
	"sort"
	"strings"
)

// OperationIdentity is the document-level identity that must remain unique
// after operations have been assembled from all generated sources.
type OperationIdentity struct {
	// Path is the normalized OpenAPI path key.
	Path string
	// Method is the HTTP method used by the operation.
	Method string
	// OperationID is the emitted operationId, if present.
	OperationID string
	// Source identifies the Loom operation that contributed this claim.
	Source string
}

// ComponentIdentity records an explicit reusable component name and the
// canonical value claimed for it.
type ComponentIdentity struct {
	// Kind names the OpenAPI component collection, such as "mediaTypes".
	Kind string
	// Name is the explicit component key.
	Name string
	// Value is a deterministic representation of the complete component value.
	Value string
	// Source identifies the Loom value that contributed this claim.
	Source string
}

// ValidateOperationIdentities rejects duplicate OpenAPI path templates,
// duplicate method/path slots, and duplicate operation IDs.
func ValidateOperationIdentities(claims []OperationIdentity) error {
	ordered := append([]OperationIdentity(nil), claims...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Path != ordered[j].Path {
			return ordered[i].Path < ordered[j].Path
		}
		if ordered[i].Method != ordered[j].Method {
			return ordered[i].Method < ordered[j].Method
		}
		if ordered[i].OperationID != ordered[j].OperationID {
			return ordered[i].OperationID < ordered[j].OperationID
		}
		return ordered[i].Source < ordered[j].Source
	})

	type routeSlot struct {
		path   string
		method string
	}
	byPathTemplate := make(map[string]OperationIdentity, len(ordered))
	byRouteSlot := make(map[routeSlot]OperationIdentity, len(ordered))
	byOperationID := make(map[string]OperationIdentity, len(ordered))
	for _, claim := range ordered {
		slot := routeSlot{path: claim.Path, method: claim.Method}
		if previous, exists := byRouteSlot[slot]; exists {
			return fmt.Errorf("OpenAPI operation %s %s is claimed by both %s and %s", claim.Method, claim.Path, identitySource(previous.Source), identitySource(claim.Source))
		}
		byRouteSlot[slot] = claim

		canonical := canonicalPathTemplate(claim.Path)
		if previous, exists := byPathTemplate[canonical]; exists && previous.Path != claim.Path {
			return fmt.Errorf("OpenAPI path templates %q and %q are equivalent", previous.Path, claim.Path)
		}
		byPathTemplate[canonical] = claim

		if claim.OperationID != "" {
			if previous, exists := byOperationID[claim.OperationID]; exists {
				return fmt.Errorf("OpenAPI operationId %q is claimed by both %s and %s", claim.OperationID, identitySource(previous.Source), identitySource(claim.Source))
			}
			byOperationID[claim.OperationID] = claim
		}
	}
	return nil
}

// ValidateComponentIdentities rejects explicit component names claimed for
// different values. Identical claims are safe to share.
func ValidateComponentIdentities(claims []ComponentIdentity) error {
	ordered := append([]ComponentIdentity(nil), claims...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Kind != ordered[j].Kind {
			return ordered[i].Kind < ordered[j].Kind
		}
		if ordered[i].Name != ordered[j].Name {
			return ordered[i].Name < ordered[j].Name
		}
		if ordered[i].Value != ordered[j].Value {
			return ordered[i].Value < ordered[j].Value
		}
		return ordered[i].Source < ordered[j].Source
	})
	type componentKey struct {
		kind string
		name string
	}
	byName := make(map[componentKey]ComponentIdentity, len(ordered))
	for _, claim := range ordered {
		if claim.Name == "" {
			continue
		}
		key := componentKey{kind: claim.Kind, name: claim.Name}
		if previous, exists := byName[key]; exists && claim.Value != previous.Value {
			return fmt.Errorf("OpenAPI %s component %q has conflicting claims from %s and %s", claim.Kind, claim.Name, identitySource(previous.Source), identitySource(claim.Source))
		}
		byName[key] = claim
	}
	return nil
}

func canonicalPathTemplate(path string) string {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		segments[i] = canonicalTemplateSegment(segment)
	}
	return strings.Join(segments, "/")
}

func canonicalTemplateSegment(segment string) string {
	var canonical strings.Builder
	for len(segment) > 0 {
		open := strings.IndexByte(segment, '{')
		if open < 0 {
			canonical.WriteString(segment)
			break
		}
		canonical.WriteString(segment[:open])
		segment = segment[open:]
		close := strings.IndexByte(segment, '}')
		if close < 0 {
			canonical.WriteString(segment)
			break
		}
		canonical.WriteString("{}")
		segment = segment[close+1:]
	}
	return canonical.String()
}

func identitySource(source string) string {
	if source == "" {
		return "another generated operation"
	}
	return source
}
