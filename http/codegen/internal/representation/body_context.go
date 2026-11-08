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
		JSONLayout, PointerLayout, DefaultLayout                     map[string]bool
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
	result.JSONPresenceTypes = maps.Clone(context.JSONPresenceTypes)
	result.PresencePointerTypes = maps.Clone(context.PresencePointerTypes)
	result.PresenceUseDefaultTypes = maps.Clone(context.PresenceUseDefaultTypes)
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
	identity := wirePolicyIdentity{
		Codec: wire.codec, Pointer: context.Pointer, UseDefault: context.UseDefault,
		JSONPresence: context.JSONPresence, CollectionElementPresence: context.CollectionElementPresence,
		JSONLayout: context.JSONPresenceTypes, PointerLayout: context.PresencePointerTypes,
		DefaultLayout: context.PresenceUseDefaultTypes, Fields: wire.fields,
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
