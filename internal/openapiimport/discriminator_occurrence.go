package openapiimport

import "github.com/CaliLuke/loom/internal/httpencoding"

type (
	discriminatorOccurrence struct {
		schema       *Schema
		json, nested bool
	}
	discriminatorOccurrenceWalker struct {
		planner documentPlanner
		seen    map[discriminatorOccurrence]bool
		changed bool
	}
)

// discriminatorOccurrences computes a decreasing fixed point over captured
// occurrences. A later non-JSON use can revoke an earlier envelope candidate.
func (a *analyzer) discriminatorOccurrences(document *Document) {
	planner := documentPlanner{schemas: make(map[string]NamedSchema)}
	for _, named := range document.Components.Schemas {
		planner.schemas[named.Name] = named
	}
	for {
		walker := discriminatorOccurrenceWalker{planner: planner, seen: make(map[discriminatorOccurrence]bool)}
		walker.scan(document)
		if !walker.changed {
			return
		}
	}
}

// visit propagates codec and enclosing-union constraints through schema edges.
func (w *discriminatorOccurrenceWalker) visit(schema *Schema, json, nested bool) {
	if schema == nil {
		return
	}
	schema = w.planner.urlSchema(schema)
	if schema == nil {
		return
	}
	key := discriminatorOccurrence{schema, json, nested}
	if w.seen[key] {
		return
	}
	w.seen[key] = true
	if d := schema.Discriminator; d != nil && d.Tagged != nil && (!json || nested || schema.Nullable) {
		d.Tagged = nil
		w.changed = true
	}
	if len(schema.OneOf) > 0 {
		tagged := schema.Discriminator != nil && schema.Discriminator.Tagged != nil
		if tagged {
			for _, branch := range schema.Discriminator.Tagged.Branches {
				w.visit(branch.Payload, json, nested)
			}
		} else {
			for _, branch := range schema.OneOf {
				w.visit(branch, json, nested || len(schema.MediaTypes) == 0)
			}
		}
	}
	for _, property := range schema.Properties {
		w.visit(property.Schema, json, nested)
	}
	for _, base := range schema.Bases {
		w.visit(base, json, nested)
	}
	w.visit(schema.Items, json, nested)
	if schema.AdditionalProperties != nil {
		w.visit(schema.AdditionalProperties.Schema, json, nested)
	}
}

// media consumes the shared field codec resolver. It never grants a new style,
// infers JSON from branch objects, or changes authored/default serialization.
func (w *discriminatorOccurrenceWalker) media(schema *Schema, content string, nested bool, outer map[string]httpencoding.Serialization, active map[*Schema]bool) {
	if schema == nil {
		return
	}
	resolved := w.planner.urlSchema(schema)
	if resolved == nil || active[resolved] {
		return
	}
	active[resolved] = true
	defer delete(active, resolved)
	if len(resolved.MediaTypes) > 0 {
		for i, branch := range resolved.OneOf {
			w.media(branch, resolved.MediaTypes[i], nested, outer, active)
		}
		return
	}
	if !isFormMediaType(content) && !isMultipartMediaType(content) {
		w.visit(schema, isJSONMediaType(content), nested)
		return
	}
	if d := resolved.Discriminator; d != nil && d.Tagged != nil {
		d.Tagged = nil
		w.changed = true
	}
	encoding := mergeMediaEncoding(outer, w.planner.mediaEncodingFor(schema))
	if len(resolved.OneOf) > 0 {
		for _, branch := range resolved.OneOf {
			w.media(branch, content, true, encoding, active)
		}
		return
	}
	properties := make(map[string]*Schema)
	w.planner.formProperties(schema, properties, make(map[*Schema]bool))
	for name, property := range properties {
		json := false
		if isFormMediaType(content) {
			codec, _, err := httpencoding.Resolve(httpencoding.FormField, name, w.planner.urlShape(property), encoding[name])
			json = err == nil && codec.Style == httpencoding.JSON
		} else {
			codec, err := httpencoding.ResolveMultipart(name, w.planner.urlShape(property), encoding[name])
			json = err == nil && codec.Kind == httpencoding.JSONPart
		}
		w.visit(property, json, nested)
	}
	if resolved.AdditionalProperties != nil {
		w.visit(resolved.AdditionalProperties.Schema, false, nested)
	}
}

func (w *discriminatorOccurrenceWalker) scan(document *Document) {
	// Structural references count even in unconsumed declarations. They may be
	// used later without silently changing the shared component representation.
	for _, named := range document.Components.Schemas {
		w.visit(named.Schema, true, false)
	}
	body := func(value RequestBody) {
		for _, content := range value.ContentTypes {
			w.media(value.Schema, content, false, nil, make(map[*Schema]bool))
		}
	}
	response := func(value Response) {
		w.media(value.Schema, value.ContentType, false, nil, make(map[*Schema]bool))
		for _, header := range value.Headers {
			w.visit(header.Header.Schema, false, false)
		}
	}
	for _, named := range document.Components.RequestBodies {
		body(named.RequestBody)
	}
	for _, named := range document.Components.Responses {
		response(named.Response)
	}
	// Native tagged unions are not parameter codecs, including JSON query
	// content. Such occurrences retain FullObject and their captured JSON intent.
	for _, named := range document.Components.Parameters {
		w.visit(named.Parameter.Schema, false, false)
	}
	for _, named := range document.Components.Headers {
		w.visit(named.Header.Schema, false, false)
	}
	for _, operation := range document.Operations {
		for _, parameter := range operation.Parameters {
			w.visit(parameter.Schema, false, false)
		}
		if operation.RequestBody != nil {
			body(*operation.RequestBody)
		}
		for _, value := range operation.Responses {
			response(value.Response)
		}
	}
}
