package openapiimport

import (
	"encoding/json/v2"
	"fmt"
	"go/token"
	"maps"
	"slices"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func (r *renderer) discriminatorMetadata(schema *Schema) error {
	d := schema.Discriminator
	if d == nil {
		return nil
	}
	if !d.validated {
		return fmt.Errorf("cannot render an unvalidated discriminator hint")
	}
	if d.Tagged == nil {
		r.line("Meta(%q, %q)", "openapi:discriminator:propertyName", d.PropertyName)
	} else {
		r.line("Meta(%q, %q)", "oneof:type:field", d.PropertyName)
		r.line("Meta(%q, %q)", "oneof:value:field", d.Tagged.ValueName)
	}
	if d.DefaultMapping != "" {
		r.line("Meta(%q, %q)", "openapi:discriminator:defaultMapping", d.DefaultMapping)
	}
	if d.Parent {
		var mappings []string
		for _, entry := range d.Explicit {
			found := false
			for _, named := range r.document.Components.Schemas {
				if named.Schema == entry.Target || entry.Target.sourcePath != "" && discriminatorIdentity(r.document.bundle, named.Schema) == discriminatorIdentity(r.document.bundle, entry.Target) {
					mappings = append(mappings, entry.Key+"="+named.Name)
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("discriminator mapping %q target was not retained", entry.Key)
			}
		}
		if len(mappings) > 0 {
			r.quotedCall("Meta", append([]string{"openapi:discriminator:mapping"}, mappings...))
		}
	}
	for _, name := range slices.Sorted(maps.Keys(d.Extensions)) {
		value, err := json.Marshal(d.Extensions[name], json.Deterministic(true))
		if err != nil {
			return fmt.Errorf("discriminator extension %q: %w", name, err)
		}
		r.line("Meta(%q, %q)", "openapi:discriminator:extension:"+name, string(value))
	}
	return nil
}

func (r *renderer) discriminatorBranchAliases(schema *Schema, index int) []string {
	if schema.Discriminator == nil {
		return nil
	}
	resolve := func(branch *Schema) *Schema {
		if branch == nil || branch.Ref == "" {
			return branch
		}
		target, err := r.resolveSchemaReference(branch, "")
		if err != nil {
			return nil
		}
		return target
	}
	target := resolve(schema.OneOf[index])
	for i := 0; i < index; i++ {
		if discriminatorIdentity(r.document.bundle, resolve(schema.OneOf[i])) == discriminatorIdentity(r.document.bundle, target) {
			return nil
		}
	}
	var keys []string
	for _, entry := range schema.Discriminator.Explicit {
		if discriminatorIdentity(r.document.bundle, entry.Target) == discriminatorIdentity(r.document.bundle, target) {
			keys = append(keys, entry.Key)
		}
	}
	return keys
}

func (r *renderer) discriminatorEnvelopeExpression(schema *Schema, path string) (string, bool, error) {
	d := schema.Discriminator
	child := r.childRenderer()
	child.open("OneOf(func()")
	bases, keys := make([]string, len(d.Tagged.Branches)), make([]string, len(d.Tagged.Branches))
	for i, branch := range d.Tagged.Branches {
		bases[i] = codegen.Goify(branch.Tag, true)
		if token.IsIdentifier(branch.Tag) {
			bases[i] = branch.Tag
		}
		if bases[i] == "" {
			bases[i] = fmt.Sprintf("Branch%d", i+1)
		}
		keys[i] = fmt.Sprintf("%08d", i)
	}
	names := expr.UniqueStableNames(bases, keys, func(base string, ordinal int) string {
		return fmt.Sprintf("%s%d", base, ordinal)
	})
	for index, branch := range d.Tagged.Branches {
		metadata := []renderedMetadata{{name: "oneof:type:tag", value: branch.Tag}}
		if branch.Component != "" {
			metadata = append(metadata, renderedMetadata{name: "openapi:component:unionEnvelope", value: branch.Component})
		}
		// Empty is meaningful: it suppresses the exporter's default description.
		metadata = append(metadata, renderedMetadata{name: "openapi:description:unionEnvelope", value: branch.Description})
		for _, key := range r.discriminatorBranchAliases(schema, index) {
			if key != branch.Tag {
				metadata = append(metadata, renderedMetadata{name: "openapi:discriminator:mapping", value: key})
			}
		}
		if err := child.attribute(names[index], branch.Payload, nil, fmt.Sprintf("%s/oneOf/%d/properties/%s", path, index, escapeJSONPointer(d.Tagged.ValueName)), metadata...); err != nil {
			return "", false, err
		}
	}
	child.close()
	return strings.TrimSpace(child.builder.String()), false, nil
}
