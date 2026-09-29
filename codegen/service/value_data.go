package service

import (
	"fmt"

	"github.com/CaliLuke/loom/expr"
)

type (
	// ValueData carries one effective service occurrence and its selected example
	// through transport analysis. Example preserves partial values, failures and
	// provenance; existing raw example fields use this same result's LegacyValue.
	ValueData struct {
		// Context owns the occurrence, sources and target projections for this generation.
		Context *expr.ValueContext
		// Occurrence identifies finalized service semantics independently of Go names.
		Occurrence expr.ValueOccurrence
		// Example retains the selected or synthesized result without target reselection.
		Example expr.ValueResult
	}

	serviceValueStore struct {
		context *expr.ValueContext
		values  map[serviceValueKey]*ValueData
	}

	serviceValueKey struct {
		method *expr.MethodExpr
		role   string
	}
)

func newValueData(context *expr.ValueContext, attribute *expr.AttributeExpr, generator *expr.ExampleGenerator) *ValueData {
	if attribute == nil || attribute.Type == expr.Empty {
		return nil
	}
	occurrence, err := context.NewOccurrence(attribute)
	if err != nil {
		// The enclosing service analyzer adds DSL attribution and converts
		// this malformed finalized design into a generation error.
		panic(fmt.Errorf("capture service value occurrence: %w", err))
	}
	suppress := false
	for _, key := range []string{"openapi:generate", "openapi:example"} {
		if value, found := attribute.Meta.Last(key); found && value == "false" {
			suppress = true
		}
	}
	selection := context.SelectExample(occurrence, expr.ExamplePolicy{Reachable: true, SuppressGenerated: suppress})
	var result expr.ValueResult
	if source, supplied := selection.Source(); supplied {
		result = context.Resolve(occurrence, source, expr.ValueRoleExample)
	} else {
		result = context.Synthesize(selection, generator)
	}
	return &ValueData{Context: context, Occurrence: occurrence, Example: result}
}

func valueDataExample(data *ValueData) any {
	if data == nil {
		return nil
	}
	value, _ := data.Example.LegacyValue()
	return value
}

func (d *ServicesData) valueStore() *serviceValueStore {
	if d.base != nil {
		return d.base.valueStore()
	}
	if d.values == nil {
		d.values = &serviceValueStore{context: expr.NewValueContext(), values: make(map[serviceValueKey]*ValueData)}
	}
	return d.values
}

func (d *ServicesData) methodValue(method *expr.MethodExpr, role string, attribute *expr.AttributeExpr, generator *expr.ExampleGenerator) *ValueData {
	store := d.valueStore()
	key := serviceValueKey{method: method, role: role}
	if value, found := store.values[key]; found {
		return value
	}
	value := newValueData(store.context, attribute, generator)
	store.values[key] = value
	return value
}
