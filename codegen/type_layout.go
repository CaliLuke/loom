package codegen

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/v2"
	"fmt"
)

type (
	// TypeLayouts is an immutable snapshot of named Go declaration layouts.
	// Contexts may share it while retaining mutable occurrence overrides in
	// their layout maps. It contains no attributes or semantic occurrences.
	TypeLayouts struct {
		entries  map[string]TypeLayout
		identity string
	}

	// TypeLayout records physical storage facts independently of their values.
	// An explicitly recorded false policy overrides an inherited true policy.
	TypeLayout struct {
		// JSONPresence selects explicit JSON presence.
		JSONPresence bool
		// Pointer selects pointer storage.
		Pointer bool
		// UseDefault selects value storage for defaulted optional fields.
		UseDefault bool
		// HasJSONPresence reports whether JSONPresence was recorded.
		HasJSONPresence bool
		// HasPointer reports whether Pointer was recorded.
		HasPointer bool
		// HasUseDefault reports whether UseDefault was recorded.
		HasUseDefault bool
	}
)

// CaptureTypeLayouts snapshots the effective registry of context. Later map
// writes cannot affect it. Capture once at the service layout allocation boundary.
func CaptureTypeLayouts(context *AttributeContext) *TypeLayouts {
	presence, pointers, defaults := context.TypeLayouts.maps()
	for key, value := range context.JSONPresenceTypes {
		presence[key] = value
	}
	for key, value := range context.PresencePointerTypes {
		pointers[key] = value
	}
	for key, value := range context.PresenceUseDefaultTypes {
		defaults[key] = value
	}
	result := &TypeLayouts{entries: make(map[string]TypeLayout, len(presence))}
	for key, value := range presence {
		result.entries[key] = TypeLayout{JSONPresence: value, HasJSONPresence: true}
	}
	for key, value := range pointers {
		entry := result.entries[key]
		entry.Pointer, entry.HasPointer = value, true
		result.entries[key] = entry
	}
	for key, value := range defaults {
		entry := result.entries[key]
		entry.UseDefault, entry.HasUseDefault = value, true
		result.entries[key] = entry
	}
	encoded, err := json.Marshal(result.entries, json.Deterministic(true))
	if err != nil {
		panic(fmt.Errorf("identify named type layouts: %w", err))
	}
	digest := sha256.Sum256(encoded)
	result.identity = hex.EncodeToString(digest[:])
	return result
}

// Identity returns the deterministic SHA-256 identity of every recorded fact.
// It is computed once on capture, so contributor keys do not retain full maps.
func (l *TypeLayouts) Identity() string {
	return l.identity
}

// Lookup returns the captured facts for key. A nil registry has no facts.
func (l *TypeLayouts) Lookup(key string) TypeLayout {
	if l == nil {
		return TypeLayout{}
	}
	return l.entries[key]
}

// TypeLayout resolves named storage from occurrence maps before the captured
// service registry. Each independently recorded fact overrides the registry.
func (a *AttributeContext) TypeLayout(key string) TypeLayout {
	result := a.TypeLayouts.Lookup(key)
	if value, found := a.JSONPresenceTypes[key]; found {
		result.JSONPresence, result.HasJSONPresence = value, true
	}
	if value, found := a.PresencePointerTypes[key]; found {
		result.Pointer, result.HasPointer = value, true
	}
	if value, found := a.PresenceUseDefaultTypes[key]; found {
		result.UseDefault, result.HasUseDefault = value, true
	}
	return result
}

// maps returns detached writable maps for recapture, including recorded false.
func (l *TypeLayouts) maps() (presence, pointers, defaults map[string]bool) {
	presence, pointers, defaults = make(map[string]bool), make(map[string]bool), make(map[string]bool)
	if l != nil {
		for key, entry := range l.entries {
			if entry.HasJSONPresence {
				presence[key] = entry.JSONPresence
			}
			if entry.HasPointer {
				pointers[key] = entry.Pointer
			}
			if entry.HasUseDefault {
				defaults[key] = entry.UseDefault
			}
		}
	}
	return
}
