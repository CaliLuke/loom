package openapiimport

import (
	"slices"
	"strings"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/high/base"
)

// selectDiscriminatorEnvelopes checks representation independently of admission.
// Failing exactness preserves the full schema; it never rejects or repairs it.
func (a *analyzer) selectDiscriminatorEnvelopes(document *Document) {
	for _, named := range document.Components.Schemas {
		a.schemas[named.Name] = named.Schema
	}
	for _, check := range a.discriminators {
		if d := check.schema.Discriminator; d != nil && d.validated && !d.Parent {
			d.Tagged = a.discriminatorEnvelope(check.schema)
		}
	}
	a.discriminatorOccurrences(document)
	for _, check := range a.discriminators {
		if d := check.schema.Discriminator; d != nil && d.Tagged != nil {
			for _, branch := range check.schema.OneOf {
				a.discriminatorSchema(branch).synthesizedEnvelope = true
			}
		}
	}
}

func (a *analyzer) discriminatorEnvelope(schema *Schema) *DiscriminatorEnvelope {
	d := schema.Discriminator
	if schema.Nullable || len(schema.OneOf) != len(a.schemaSources[schema].OneOf) {
		return nil
	}
	envelope := &DiscriminatorEnvelope{}
	targets := make(map[targetKey]bool)
	tags := make(map[string]bool)
	for _, branch := range schema.OneOf {
		target := a.discriminatorSchema(branch)
		candidate, valueName := a.discriminatorEnvelopeBranch(branch, target, d.PropertyName)
		if candidate == nil {
			return nil
		}
		identity := discriminatorIdentity(a.bundle, target)
		if targets[identity] || tags[candidate.Tag] || !a.discriminatorKey(d, candidate.Tag, target) {
			return nil
		}
		targets[identity], tags[candidate.Tag] = true, true
		if envelope.ValueName != "" && envelope.ValueName != valueName || a.discriminatorEnvelopeShared(schema, target) {
			return nil
		}
		envelope.ValueName = valueName
		envelope.Branches = append(envelope.Branches, *candidate)
	}
	return envelope
}

// discriminatorEnvelopeBranch checks what the existing tagged constructor can
// reproduce, including annotation presence and authored payload representation.
func (a *analyzer) discriminatorEnvelopeBranch(branch, target *Schema, propertyName string) (*DiscriminatorBranch, string) {
	if target == nil || target.Type != "object" || target.Nullable || len(target.Properties) != 2 || len(target.Required) != 2 || len(target.Bases) > 0 {
		return nil, ""
	}
	if !discriminatorKeywords(a.schemaSources[target], "type", "properties", "required", "description") {
		return nil, ""
	}
	node := a.schemaSources[target].GoLow().RootNode
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == "description" && node.Content[i+1].Value == "" {
			return nil, ""
		}
	}
	// An occurrence overlay on the envelope itself would disappear when synthesized.
	if branch.Ref != "" && branch.referenceSiblings {
		return nil, ""
	}
	tag, payload, valueName := discriminatorEnvelopeMembers(target, propertyName)
	if tag == nil || tag.Type != "string" || tag.Ref != "" || len(tag.Enum) != 1 || !discriminatorKeywords(a.schemaSources[tag], "type", "enum", "const") {
		return nil, ""
	}
	wire, ok := tag.Enum[0].(string)
	if !ok || wire == "" || !a.discriminatorEnvelopePayload(target, payload, valueName) {
		return nil, ""
	}
	component, _ := componentReferenceName(branch.Ref, "#/components/schemas/")
	return &DiscriminatorBranch{Tag: wire, Payload: payload, Component: component, Description: target.Description}, valueName
}

func discriminatorEnvelopeMembers(target *Schema, propertyName string) (tag, payload *Schema, valueName string) {
	for _, property := range target.Properties {
		if !slices.Contains(target.Required, property.Name) {
			return nil, nil, ""
		}
		if property.Name == propertyName {
			tag = property.Schema
		} else {
			payload = property.Schema
			valueName = property.Name
		}
	}
	return tag, payload, valueName
}

func (a *analyzer) discriminatorEnvelopePayload(target, payload *Schema, valueName string) bool {
	authored, _ := a.schemaSources[target].Properties.Get(valueName)
	if authored == nil || payload == nil {
		return false
	}
	if !authored.IsReference() {
		raw := authored.Schema()
		if raw == nil || len(raw.Type) != 1 || !slices.Contains([]string{"string", "integer", "number", "boolean"}, raw.Type[0]) {
			return false
		}
	}
	return payload.Ref != "" || !payload.Nullable && slices.Contains([]string{"string", "integer", "number", "boolean"}, payload.Type)
}

func discriminatorKeywords(source *base.Schema, allowed ...string) bool {
	if source == nil || source.GoLow() == nil {
		return false
	}
	node := source.GoLow().RootNode
	if node == nil || node.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if !slices.Contains(allowed, node.Content[i].Value) {
			return false
		}
	}
	return true
}

func (a *analyzer) discriminatorKey(d *Discriminator, key string, target *Schema) bool {
	for _, entry := range d.Explicit {
		if entry.Key == key {
			return discriminatorIdentity(a.bundle, entry.Target) == discriminatorIdentity(a.bundle, target)
		}
	}
	// Only entry-document component names form implicit keys; transplanted names
	// do not change the source hint's meaning during representation selection.
	name := componentDeclaration(key)
	source := a.schemas[key]
	origin, exists := a.bundle.origins[name]
	return exists && origin.Document == "entry" && origin.Pointer == name && source != nil && discriminatorIdentity(a.bundle, source) == discriminatorIdentity(a.bundle, target)
}

func (a *analyzer) discriminatorEnvelopeShared(union, target *Schema) bool {
	wanted := discriminatorIdentity(a.bundle, target)
	for occurrence, reference := range a.bundle.references {
		name, ok := componentReferenceName(reference, "#/components/schemas/")
		if !ok || discriminatorIdentity(a.bundle, a.discriminatorSchema(a.schemas[name])) != wanted {
			continue
		}
		if strings.HasPrefix(occurrence, union.sourcePath+"/oneOf/") {
			continue
		}
		return true
	}
	for occurrence := range a.bundle.origins {
		if !strings.Contains(occurrence, "/discriminator/") || strings.HasPrefix(occurrence, union.sourcePath+"/discriminator/") {
			continue
		}
		_, resolved, message := a.discriminatorTarget(occurrence)
		if message == "" && resolved != nil && discriminatorIdentity(a.bundle, a.discriminatorSchema(resolved)) == wanted {
			return true
		}
	}
	return false
}
