package openapiimport

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
)

type (
	// Discriminator is a validated, non-validating OpenAPI selection hint.
	Discriminator struct {
		// PropertyName is the discriminating JSON member name.
		PropertyName string
		// Explicit retains authored mapping keys in source order, including aliases.
		Explicit []DiscriminatorEntry
		// DefaultMapping is a canonical component reference or external absolute URI.
		DefaultMapping string
		// Extensions contains Discriminator Object extension values.
		Extensions map[string]any
		// Parent reports the allOf parent hint form.
		Parent bool
		// Tagged records an exact JSON envelope representation, when eligible.
		Tagged    *DiscriminatorEnvelope
		validated bool
		children  []targetKey
	}
	// DiscriminatorEntry binds one explicit key to a normalized alternative.
	DiscriminatorEntry struct {
		// Key is the authored mapping key, never a generated branch name.
		Key string
		// Target is the normalized schema identity; duplicate alternatives share it.
		Target *Schema
	}
	// DiscriminatorEnvelope retains every fact needed to synthesize exact envelopes.
	DiscriminatorEnvelope struct {
		// ValueName is the common payload member.
		ValueName string
		// Branches are the payloads and envelope annotations in alternative order.
		Branches []DiscriminatorBranch
	}
	// DiscriminatorBranch describes one exact envelope's payload and wire identity.
	DiscriminatorBranch struct {
		// Tag is the literal required tag value.
		Tag string
		// Payload is the original value member, including its annotations.
		Payload *Schema
		// Component is the source envelope's component name, if any.
		Component string
		// Description is the source envelope's description.
		Description string
	}
	discriminatorCheck struct {
		schema *Schema
		source *base.Schema
		path   string
	}
)

// normalizeDiscriminators runs after every declaration and occurrence is known.
// Resolver outcomes identify authored occurrences, not their destination schemas.
func (a *analyzer) normalizeDiscriminators() {
	for _, check := range a.discriminators {
		a.normalizeDiscriminator(check)
	}
}

func (a *analyzer) normalizeDiscriminator(check discriminatorCheck) {
	schema, source, path := check.schema, check.source, check.path
	hint := source.Discriminator
	if hint == nil {
		return
	}
	fail := func(message string) {
		a.unsupported("schema-discriminator", path+"/discriminator", message)
	}
	if len(source.OneOf) > 0 && (len(source.AllOf) > 0 || len(source.AnyOf) > 0) {
		fail("adjacent oneOf and anyOf/allOf with a discriminator is undefined (§4.25.2)")
		return
	}
	if len(source.AnyOf) > 0 {
		fail("anyOf discriminator admission belongs to #1019")
		return
	}
	d := a.discriminatorFields(check)
	if d == nil {
		return
	}
	targets := make(map[targetKey]bool)
	for _, branch := range schema.OneOf {
		targets[discriminatorIdentity(a.bundle, a.discriminatorAlternative(branch))] = true
	}
	if d.Parent {
		var err error
		d.children, err = a.directAllOfChildren(schema)
		if err != nil {
			fail("cannot inspect direct allOf child roles: " + err.Error())
			return
		}
		for _, identity := range d.children {
			targets[identity] = true
		}
		if len(d.children) == 0 {
			fail("discriminator without oneOf or direct allOf children is undefined (§4.25.2)")
			return
		}
	}
	if message := a.bindDiscriminatorMappings(d, targets, check); message != "" {
		fail(message)
		return
	}
	optional := !a.discriminatorPropertiesDefined(check, d.PropertyName)
	if optional && (a.openAPI32() && d.DefaultMapping == "" || strings.HasPrefix(a.version, "3.0.")) {
		if a.openAPI32() {
			fail("optional discriminating property requires defaultMapping (§4.25.4)")
		} else {
			fail("OpenAPI 3.0 requires a required discriminating property")
		}
		return
	}
	d.validated = true
	*schema.Discriminator = *d
}

// discriminatorPropertiesDefined keeps hint ownership on the authored
// alternatives. An outer nullable wrapper does not change an inner union hint.
func (a *analyzer) discriminatorPropertiesDefined(check discriminatorCheck, property string) bool {
	if len(check.source.OneOf) == 0 {
		return a.schemaDefinesProperties(check.schema, []string{property})
	}
	if len(check.source.OneOf) > len(check.schema.OneOf) {
		return false
	}
	for _, alternative := range check.schema.OneOf {
		if !a.schemaDefinesProperties(alternative, []string{property}) {
			return false
		}
	}
	return true
}

// bindDiscriminatorMappings joins resolver outcomes to the complete alternative inventory.
func (a *analyzer) bindDiscriminatorMappings(d *Discriminator, targets map[targetKey]bool, check discriminatorCheck) string {
	for key := range check.source.Discriminator.Mapping.FromOldest() {
		_, target, message := a.discriminatorTarget(check.path + "/discriminator/mapping/" + escapeJSONPointer(key))
		if message != "" {
			return fmt.Sprintf("mapping %q unresolved: %s", key, message)
		}
		if target == nil || !targets[discriminatorIdentity(a.bundle, target)] {
			return fmt.Sprintf("mapping %q does not target a listed alternate", key)
		}
		d.Explicit = append(d.Explicit, DiscriminatorEntry{Key: key, Target: target})
	}
	return ""
}

// discriminatorFields validates the versioned Discriminator Object grammar.
func (a *analyzer) discriminatorFields(check discriminatorCheck) *Discriminator {
	source, path := check.source, check.path
	hint := source.Discriminator
	fail := func(message string) {
		a.unsupported("schema-discriminator", path+"/discriminator", message)
	}
	node, err := descriptionNode(a.bundle.root, strings.TrimPrefix(path, "#")+"/discriminator")
	if err != nil || node.Kind != yaml.MappingNode {
		fail("discriminator must be an object")
		return nil
	}
	d := &Discriminator{PropertyName: hint.PropertyName, Parent: len(source.OneOf) == 0}
	extensions := orderedmap.New[string, *yaml.Node]()
	if d.PropertyName == "" {
		fail("propertyName must be a nonempty string")
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key, value := node.Content[i].Value, node.Content[i+1]
		switch key {
		case "propertyName":
			if value.Tag != "!!str" {
				fail("propertyName must be a string")
				return nil
			}
		case "mapping":
			if value.Kind != yaml.MappingNode {
				fail("mapping must be an object")
				return nil
			}
		case "defaultMapping":
			if !a.openAPI32() {
				a.unsupported("versioned-field", path+"/discriminator/defaultMapping", "defaultMapping requires OpenAPI 3.2")
				return nil
			}
			if value.Tag != "!!str" || value.Value == "" {
				fail("defaultMapping must be a nonempty string")
				return nil
			}
			normalized, _, message := a.discriminatorTarget(path + "/discriminator/defaultMapping")
			if message != "" {
				fail("defaultMapping unresolved: " + message)
				return nil
			}
			d.DefaultMapping = normalized
		default:
			if !strings.HasPrefix(key, "x-") {
				fail("unknown discriminator field " + key)
				return nil
			}
			if strings.HasPrefix(a.version, "3.0.") {
				a.unsupported("versioned-field", path+"/discriminator/"+escapeJSONPointer(key), "Discriminator extensions require OpenAPI 3.1 or 3.2")
				return nil
			}
			extensions.Set(key, value)
		}
	}
	d.Extensions = a.extensions(path+"/discriminator", extensions)
	return d
}

func (a *analyzer) discriminatorSchema(schema *Schema) *Schema {
	seen := make(map[*Schema]bool)
	for schema != nil && schema.Ref != "" && !seen[schema] {
		seen[schema] = true
		name, ok := componentReferenceName(schema.Ref, "#/components/schemas/")
		if !ok {
			return nil
		}
		schema = a.schemas[name]
	}
	return schema
}

// directAllOfChildren reads admitted direct allOf edges from the complete
// source graph, before operation selection can hide inline child occurrences.
// Schema grammar roles exclude lookalike allOf members in properties or data.
func (a *analyzer) directAllOfChildren(parent *Schema) ([]targetKey, error) {
	wanted := discriminatorIdentity(a.bundle, parent)
	children := make(map[targetKey]bool)
	for _, reference := range a.bundle.graph.edges {
		if reference.kind != "schemas" || reference.class != "component" || reference.target == nil || *reference.target != wanted {
			continue
		}
		occurrence := strings.TrimSuffix(reference.at.pointer, "/$ref")
		owner, index, found := strings.Cut(occurrence, "/allOf/")
		// The last allOf is the direct owner when compositions are nested.
		if at := strings.LastIndex(occurrence, "/allOf/"); at >= 0 {
			owner, index = occurrence[:at], occurrence[at+len("/allOf/"):]
		}
		if !found {
			continue
		}
		if _, err := strconv.ParseUint(index, 10, 64); err != nil {
			continue
		}
		positions, err := descriptionSchemaPositions(a.bundle.graph.registry.docs[reference.at.doc])
		if err != nil {
			return nil, err
		}
		if _, schemaRole := positions[owner]; schemaRole {
			children[targetKey{doc: reference.at.doc, pointer: owner}] = true
		}
	}
	result := make([]targetKey, 0, len(children))
	for child := range children {
		result = append(result, child)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].doc != result[j].doc {
			return result[i].doc < result[j].doc
		}
		return result[i].pointer < result[j].pointer
	})
	return result, nil
}

func (a *analyzer) discriminatorTarget(path string) (string, *Schema, string) {
	origin, ok := a.bundle.origins[path]
	if !ok {
		return "", nil, "mapping occurrence has no resolver provenance"
	}
	for _, outcome := range a.bundle.graph.discriminatorEdges() {
		doc := a.bundle.graph.registry.docs[outcome.source.doc]
		if doc.portable != origin.Document || "#"+outcome.source.pointer != origin.Pointer {
			continue
		}
		if outcome.failure != nil {
			return "", nil, outcome.failure.Message
		}
		if outcome.external != "" {
			return outcome.external, nil, ""
		}
		ref := outcome.bundled
		if outcome.entry != "" {
			ref = componentDeclaration(outcome.entry)
		}
		name, ok := componentReferenceName(ref, "#/components/schemas/")
		if !ok {
			return "", nil, "mapping outcome has no loaded schema"
		}
		target := a.schemas[name]
		if target == nil {
			return "", nil, "mapping target has no normalized schema"
		}
		return ref, target, ""
	}
	return "", nil, "mapping occurrence was not resolved"
}

func discriminatorIdentity(bundle *bundledDescription, schema *Schema) targetKey {
	if schema == nil {
		return targetKey{}
	}
	if bundle != nil {
		if origin, ok := bundle.origins[schema.sourcePath]; ok {
			for id, document := range bundle.graph.registry.docs {
				if document.portable == origin.Document {
					return targetKey{doc: id, pointer: strings.TrimPrefix(origin.Pointer, "#")}
				}
			}
		}
	}
	return targetKey{pointer: strings.TrimPrefix(schema.sourcePath, "#")}
}

// discriminatorAlternative resolves the listed reference once. A component alias
// is itself a distinct Schema target, even when it has the same value domain.
func (a *analyzer) discriminatorAlternative(schema *Schema) *Schema {
	if schema == nil || schema.Ref == "" {
		return schema
	}
	name, ok := componentReferenceName(schema.Ref, "#/components/schemas/")
	if !ok {
		return nil
	}
	return a.schemas[name]
}
