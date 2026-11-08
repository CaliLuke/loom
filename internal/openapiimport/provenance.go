package openapiimport

import (
	"encoding/json/jsontext"
	"slices"
	"sort"
	"strings"
)

type (
	// ProvenanceReference identifies one bundled reference occurrence and its source.
	ProvenanceReference struct {
		// Path is the occurrence's JSON Pointer in the bundled description.
		Path string
		// Origin identifies the authored occurrence before bundling.
		Origin SourceLocation
	}
	// ProvenanceEntry describes a transplanted, inlined, or rewritten contract node.
	ProvenanceEntry struct {
		// Path is the node's JSON Pointer in the bundled description.
		Path string
		// Origin identifies the node before bundling.
		Origin SourceLocation
		// References lists every bundled occurrence referencing this node.
		References []ProvenanceReference
	}
)

// Provenance returns deterministic source evidence for the original analysis.
// Selection and partial import preserve this evidence; it is never emitted as DSL metadata.
func (d *Document) Provenance() []ProvenanceEntry {
	if d == nil || d.bundle == nil {
		return nil
	}
	var entries []ProvenanceEntry
	for pointer, origin := range d.bundle.origins {
		tokens := slices.Collect(jsontext.Pointer(strings.TrimPrefix(pointer, "#")).Tokens())
		component := len(tokens) == 3 && tokens[0] == "components"
		pathItem := len(tokens) == 2 && tokens[0] == "paths"
		moved := origin.Document != "entry" || origin.Pointer != pointer
		if !((component || pathItem) && moved) && origin.Authored == "" {
			continue
		}
		entry := ProvenanceEntry{Path: pointer, Origin: origin}
		for occurrence, target := range d.bundle.references {
			if target == pointer {
				entry.References = append(entry.References, ProvenanceReference{Path: occurrence, Origin: d.bundle.origins[occurrence]})
			}
		}
		sort.Slice(entry.References, func(i, j int) bool {
			return entry.References[i].Path < entry.References[j].Path
		})
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})
	for i := range entries {
		entries[i].Path = sourceDisplayPointer(entries[i].Path, entries[i].Origin.Authored)
		entries[i].Origin = publicSourceLocation(entries[i].Origin)
		for j := range entries[i].References {
			reference := &entries[i].References[j]
			reference.Path = sourceDisplayPointer(reference.Path, reference.Origin.Authored)
			reference.Origin = publicSourceLocation(reference.Origin)
		}
	}
	return entries
}
