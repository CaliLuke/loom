package ir

import (
	"fmt"
	"maps"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

type (
	schemaOccurrenceAnalysis struct {
		root     *Schema
		analyzer *Analyzer
	}
	representationComponent struct {
		scope       *Analyzer
		local       string
		naming      representationComponentName
		fingerprint string
		assigned    string
	}
)

func (a *Analyzer) analyzeOccurrence(attribute *expr.AttributeExpr, context string, plan expr.ValuePlanNode) *Schema {
	// Validate from the occurrence root so diagnostics retain structural paths.
	// These paths never enter component fingerprints or example seed contexts.
	byteProjectionIdentity(attribute, plan)
	root := a.analyzeSchemaPositions(attribute, context, plan, expr.ValuePlanNode{})
	a.occurrences = append(a.occurrences, schemaOccurrenceAnalysis{root, a})
	return root
}

func (a *Analyzer) analyzePreparedOccurrence(
	attribute *expr.AttributeExpr,
	context string,
	target *transportir.ValueTarget,
) *Schema {
	plan := representationRoot(attribute, target)
	byteProjectionIdentity(attribute, plan)
	root := a.analyzePreparedSchemaPositions(attribute, context, target, plan)
	a.occurrences = append(a.occurrences, schemaOccurrenceAnalysis{root, a})
	return root
}

func (a *Analyzer) analyzePreparedSchemaPositions(
	attribute *expr.AttributeExpr,
	context string,
	target *transportir.ValueTarget,
	plan expr.ValuePlanNode,
	noRef ...bool,
) *Schema {
	if target == nil {
		return a.analyzeSchemaPositions(attribute, context, expr.ValuePlanNode{}, expr.ValuePlanNode{}, noRef...)
	}
	if target.Error != nil {
		panic(fmt.Errorf("OpenAPI representation: %w", target.Error))
	}
	previous := a.preparedExamples
	a.preparedExamples = target
	defer func() {
		a.preparedExamples = previous
	}()
	examplePlan := representationRoot(attribute, target)
	result := a.analyzeSchemaPositions(attribute, context, plan, examplePlan, noRef...)
	if !a.customExampleValue && result != nil && result.Example == nil {
		restore := a.schemaPlanScope(plan, examplePlan)
		a.applySchemaExample(result, attribute, context)
		restore()
	}
	return result
}

// finalizeRepresentations reserves every authored/public name before assigning
// variants. The compared identities are complete effective schemas, including
// recursive child representations; codec labels are not schema equivalence.
func (a *Analyzer) finalizeRepresentations() {
	a.discardRegistrationOnlyVariants()
	components, reserved := a.representationComponents()
	allocated, names := allocateRepresentations(components, reserved)
	a.installRepresentations(allocated, names)
}

func (a *Analyzer) representationComponents() ([]*representationComponent, map[string]bool) {
	components := make([]*representationComponent, 0, len(a.schemas))
	reserved := make(map[string]bool)
	fingerprints := completeSchemaFingerprints(a.schemas)
	for name := range a.schemas {
		naming, found := a.schemaNames[name]
		if !found {
			naming = representationComponentName{desired: name, logical: name}
		}
		reserved[naming.desired] = true
		components = append(components, &representationComponent{scope: a, local: name, naming: naming, fingerprint: fingerprints[name]})
	}
	return components, reserved
}

func allocateRepresentations(components []*representationComponent, reserved map[string]bool) (map[string]*representationComponent, map[*Analyzer]map[string]string) {
	sort.Slice(components, func(i, j int) bool {
		left, right := components[i], components[j]
		if left.naming.desired != right.naming.desired {
			return left.naming.desired < right.naming.desired
		}
		if left.naming.json != right.naming.json {
			return left.naming.json
		}
		if left.fingerprint != right.fingerprint {
			return left.fingerprint < right.fingerprint
		}
		return left.naming.logical < right.naming.logical
	})
	allocated := make(map[string]*representationComponent)
	names := make(map[*Analyzer]map[string]string)
	for _, component := range components {
		desired := component.naming.desired
		name := desired
		if prior, exists := allocated[name]; exists && prior.fingerprint != component.fingerprint {
			if component.naming.explicit && prior.naming.explicit && (prior.naming.logical != component.naming.logical || prior.naming.baseline != component.naming.baseline) {
				panic(fmt.Sprintf("openapi: explicit component name %q is claimed by multiple different schemas; use distinct Meta(%q, ...) values", desired, "openapi:typename"))
			}
			name = desired + "_" + component.fingerprint[:16]
			for suffix := 2; reserved[name] || (allocated[name] != nil && allocated[name].fingerprint != component.fingerprint); suffix++ {
				name = desired + "_" + component.fingerprint[:16] + "_" + strconv.Itoa(suffix)
			}
		}
		component.assigned = name
		if allocated[name] == nil {
			allocated[name] = component
		}
		if names[component.scope] == nil {
			names[component.scope] = make(map[string]string)
		}
		names[component.scope][component.local] = name
	}
	return allocated, names
}

func (a *Analyzer) installRepresentations(allocated map[string]*representationComponent, names map[*Analyzer]map[string]string) {
	originals := a.schemas
	a.schemas = make(map[string]*Schema, len(allocated))
	for name, component := range allocated {
		a.schemas[name] = mapSchemaReferences(originals[component.local], func(ref, _ string) string {
			local, ok := schemaComponentName(ref)
			if !ok {
				return ref
			}
			if assigned, found := names[component.scope][local]; found {
				return toRef(assigned)
			}
			return ref
		}, a.copySchemaConstruction)
	}
	for _, occurrence := range a.occurrences {
		if occurrence.root == nil {
			continue
		}
		mapped := mapSchemaReferences(occurrence.root, func(ref, _ string) string {
			local, ok := schemaComponentName(ref)
			if !ok {
				return ref
			}
			if assigned, found := names[occurrence.analyzer][local]; found {
				return toRef(assigned)
			}
			return ref
		}, a.copySchemaConstruction)
		a.copySchemaConstruction(mapped, occurrence.root)
		*occurrence.root = *mapped
	}
}

// mapSchemaReferences reports the path in stableSchemaEncoding with each
// reference. Consumers can retain exact serialized reference slots without
// treating lookalike strings in examples or extension values as graph edges.
func mapSchemaReferences(source *Schema, reference func(string, string) string, copied ...func(*Schema, *Schema)) *Schema {
	var visit func(*Schema, string) *Schema
	visit = func(source *Schema, path string) *Schema {
		if source == nil {
			return nil
		}
		result := *source
		result.Ref = reference(source.Ref, path+"/Ref")
		result.Items = visit(source.Items, path+"/Items")
		result.ContentSchema = visit(source.ContentSchema, path+"/ContentSchema")
		result.Not = visit(source.Not, path+"/Not")
		mapSchemaReferenceMaps(source, &result, path, visit)
		mapSchemaReferenceLists(source, &result, path, visit)
		mapSchemaReferenceAdditional(source, &result, path, visit)
		if source.Discriminator != nil {
			value := *source.Discriminator
			if value.Mapping != nil {
				value.Mapping = make(map[string]string, len(source.Discriminator.Mapping))
				for _, name := range slices.Sorted(maps.Keys(source.Discriminator.Mapping)) {
					referencePath := path + "/Discriminator/Mapping/" + strings.ReplaceAll(strings.ReplaceAll(name, "~", "~0"), "/", "~1")
					value.Mapping[name] = reference(source.Discriminator.Mapping[name], referencePath)
				}
			}
			result.Discriminator = &value
		}
		for _, record := range copied {
			record(source, &result)
		}
		return &result
	}
	return visit(source, "")
}

func mapSchemaReferenceMaps(source, result *Schema, path string, visit func(*Schema, string) *Schema) {
	for _, pair := range []struct {
		key    string
		input  map[string]*Schema
		output *map[string]*Schema
	}{{"Properties", source.Properties, &result.Properties}, {"Defs", source.Defs, &result.Defs}} {
		if pair.input == nil {
			continue
		}
		*pair.output = make(map[string]*Schema, len(pair.input))
		for _, name := range slices.Sorted(maps.Keys(pair.input)) {
			childPath := path + "/" + pair.key + "/" + strings.ReplaceAll(strings.ReplaceAll(name, "~", "~0"), "/", "~1")
			(*pair.output)[name] = visit(pair.input[name], childPath)
		}
	}
}

func mapSchemaReferenceLists(source, result *Schema, path string, visit func(*Schema, string) *Schema) {
	for _, pair := range []struct {
		key    string
		input  []*Schema
		output *[]*Schema
	}{{"AllOf", source.AllOf, &result.AllOf}, {"AnyOf", source.AnyOf, &result.AnyOf}, {"OneOf", source.OneOf, &result.OneOf}} {
		if pair.input == nil {
			continue
		}
		*pair.output = make([]*Schema, len(pair.input))
		for index, child := range pair.input {
			(*pair.output)[index] = visit(child, path+"/"+pair.key+"/"+strconv.Itoa(index))
		}
	}
}

func mapSchemaReferenceAdditional(source, result *Schema, path string, visit func(*Schema, string) *Schema) {
	for _, pair := range []struct {
		key    string
		input  *BoolOrSchema
		output **BoolOrSchema
	}{{"AdditionalProperties", source.AdditionalProperties, &result.AdditionalProperties}, {"UnevaluatedProperties", source.UnevaluatedProperties, &result.UnevaluatedProperties}} {
		if pair.input == nil {
			continue
		}
		value := *pair.input
		value.Schema = visit(value.Schema, path+"/"+pair.key+"/Schema")
		*pair.output = &value
	}
}

// discardRegistrationOnlyVariants distinguishes forced inclusion from an actual
// codec occurrence. Legacy schemas used to obtain annotations need no variant
// unless an actual occurrence (or another retained forced graph) references them.
func (a *Analyzer) discardRegistrationOnlyVariants() {
	reached := make(map[string]bool)
	var visit func(*Schema)
	visit = func(schema *Schema) {
		mapSchemaReferences(schema, func(ref, _ string) string {
			if name, local := schemaComponentName(ref); local && !reached[name] {
				reached[name] = true
				visit(a.schemas[name])
			}
			return ref
		})
	}
	for _, occurrence := range a.occurrences {
		visit(occurrence.root)
	}
	jsonLogical := make(map[string]bool)
	for name := range reached {
		naming := a.schemaNames[name]
		if naming.json {
			jsonLogical[naming.logical] = true
		}
	}
	for _, registration := range a.registrations {
		name, _ := schemaComponentName(registration.Ref)
		if !jsonLogical[a.schemaNames[name].logical] {
			visit(registration)
		}
	}
	for name, naming := range a.schemaNames {
		if !reached[name] && !naming.json && jsonLogical[naming.logical] {
			delete(a.schemas, name)
		}
	}
}
