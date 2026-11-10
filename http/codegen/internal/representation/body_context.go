package representation

import (
	"encoding/json/v2"
	"fmt"
	"maps"
	"slices"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

type (
	// contributorField contains only physical wire facts. Requiredness and
	// semantic defaults remain owned by each projected declaration.
	contributorField struct {
		WireName string
		Visible  bool
		Presence expr.ValueFieldPresence
	}

	contributorContract struct {
		parameter parameterContract
		fields    map[string]contributorField
	}

	// wirePolicyIdentity excludes mutable scopes and graph pointer identities.
	// The complete occurrence field map is retained because a later published
	// child may own fields absent from the current contributor.
	wirePolicyIdentity struct {
		Codec                                                        expr.ValueCodec
		Pointer, UseDefault, JSONPresence, CollectionElementPresence bool
		Layout                                                       string
		Fields                                                       map[string]contributorField
		Contract                                                     contributorKey
	}
)

// ProjectionIdentity identifies the complete occurrence context before its
// contributor graph is built. It is provisional, not schema equivalence.
func (w *WireContext) ProjectionIdentity(projection *expr.HTTPBodyProjection) string {
	key := struct {
		Wire     string
		Excluded []string
	}{w.identity, slices.Sorted(slices.Values(projection.Excluded))}
	encoded, err := json.Marshal(key, json.Deterministic(true))
	if err != nil {
		panic(fmt.Errorf("identify HTTP body projection: %w", err))
	}
	return string(encoded)
}

func captureProjectionContext(context *codegen.AttributeContext, target *expr.AttributeExpr) *codegen.AttributeContext {
	result := context.Dup()
	if context.TypeLayouts == nil || target == nil {
		// Standalone contexts capture their registry; already captured contexts
		// only copy their small local overrides when handed to a contributor.
		result.JSONPresenceTypes = maps.Clone(context.JSONPresenceTypes)
		result.PresencePointerTypes = maps.Clone(context.PresencePointerTypes)
		result.PresenceUseDefaultTypes = maps.Clone(context.PresenceUseDefaultTypes)
	} else {
		result.JSONPresenceTypes = captureLayoutOverrides(context.JSONPresenceTypes, func(key string) (bool, bool) {
			facts := context.TypeLayouts.Lookup(key)
			return facts.JSONPresence, facts.HasJSONPresence
		})
		result.PresencePointerTypes = captureLayoutOverrides(context.PresencePointerTypes, func(key string) (bool, bool) {
			facts := context.TypeLayouts.Lookup(key)
			return facts.Pointer, facts.HasPointer
		})
		result.PresenceUseDefaultTypes = captureLayoutOverrides(context.PresenceUseDefaultTypes, func(key string) (bool, bool) {
			facts := context.TypeLayouts.Lookup(key)
			return facts.UseDefault, facts.HasUseDefault
		})
	}
	seen := make(map[*expr.AttributeExpr]bool)
	for target != nil && !seen[target] {
		seen[target] = true
		ApplyLayout(result, target)
		named, ok := target.Type.(expr.UserType)
		if !ok {
			break
		}
		target = named.Attribute()
	}
	return result
}

// captureLayoutOverrides retains every changed fact, including facts for a
// sibling that a contributor may publish later. Registry-equal entries cost no
// target-owned storage. The mutable source remains borrowed only for this call.
func captureLayoutOverrides(source map[string]bool, inherited func(string) (bool, bool)) map[string]bool {
	var result map[string]bool
	for key, value := range source {
		if prior, found := inherited(key); found && prior == value {
			continue
		}
		if result == nil {
			result = make(map[string]bool)
		}
		result[key] = value
	}
	return result
}

func contributorObject(target *expr.AttributeExpr) *expr.AttributeExpr {
	seen := make(map[*expr.AttributeExpr]bool)
	for target != nil && !seen[target] {
		seen[target] = true
		named, ok := target.Type.(expr.UserType)
		if !ok {
			if expr.IsObject(target.Type) {
				return target
			}
			return nil
		}
		target = named.Attribute()
	}
	return nil
}

func wireContextIdentity(wire *WireContext) (string, error) {
	context := wire.context
	layouts := context.TypeLayouts
	if layouts == nil || len(context.JSONPresenceTypes)+len(context.PresencePointerTypes)+len(context.PresenceUseDefaultTypes) != 0 {
		layouts = codegen.CaptureTypeLayouts(context)
	}
	identity := wirePolicyIdentity{
		Codec: wire.codec, Pointer: context.Pointer, UseDefault: context.UseDefault,
		JSONPresence: context.JSONPresence, CollectionElementPresence: context.CollectionElementPresence,
		Layout: layouts.Identity(), Fields: wire.fields,
	}
	if wire.contract.form != nil {
		identity.Contract.Form = wire.contract.form.Fields
	}
	if wire.contract.multipart != nil {
		identity.Contract.Multipart = wire.contract.multipart.Fields
		identity.Contract.Custom = wire.contract.multipart.Custom
	}
	encoded, err := json.Marshal(identity, json.Deterministic(true))
	if err != nil {
		return "", fmt.Errorf("identify HTTP body context: %w", err)
	}
	return string(encoded), nil
}
