package openapiv3

import "github.com/CaliLuke/loom/http/codegen/openapi"

func cleanupComponentSchemas(paths map[string]*PathItem, schemas map[string]*openapi.Schema, reusable reusableComponents) {
	collapseSchemaAliases(paths, schemas, reusable)
}

func collapseSchemaAliases(paths map[string]*PathItem, schemas map[string]*openapi.Schema, reusable reusableComponents) {
	if len(schemas) == 0 {
		return
	}
	aliases := schemaAliases(schemas)
	if len(aliases) == 0 {
		return
	}
	resolveRef := aliasResolver(aliases)
	for _, pathItem := range paths {
		rewritePathItemSchemaRefs(pathItem, resolveRef)
	}
	rewriteReusableSchemaRefs(reusable, resolveRef)
	for _, schema := range schemas {
		rewriteSchemaRefs(schema, resolveRef)
	}
	for name := range aliases {
		delete(schemas, name)
	}
}

func schemaAliases(schemas map[string]*openapi.Schema) map[string]string {
	aliases := make(map[string]string)
	for name, schema := range schemas {
		if !isPureRefSchema(schema) {
			continue
		}
		target, ok := schemaNameFromRef(schema.Ref)
		if !ok || target == name {
			continue
		}
		aliases[name] = target
	}
	return aliases
}

func aliasResolver(aliases map[string]string) func(string) string {
	return func(ref string) string {
		name, ok := schemaNameFromRef(ref)
		if !ok {
			return ref
		}
		seen := map[string]struct{}{name: {}}
		for {
			target, ok := aliases[name]
			if !ok {
				return toSchemaComponentRef(name)
			}
			if _, loop := seen[target]; loop {
				return toSchemaComponentRef(target)
			}
			seen[target] = struct{}{}
			name = target
		}
	}
}

func pruneUnusedComponentSchemas(paths map[string]*PathItem, schemas map[string]*openapi.Schema, reusable reusableComponents) map[string]*openapi.Schema {
	if len(schemas) == 0 {
		return schemas
	}
	reachable := collectReachableComponentSchemas(paths, schemas, reusable)
	pruned := make(map[string]*openapi.Schema, len(reachable))
	for name := range reachable {
		if schema := schemas[name]; schema != nil {
			pruned[name] = schema
		}
	}
	return pruned
}

func collectReachableComponentSchemas(paths map[string]*PathItem, schemas map[string]*openapi.Schema, reusable reusableComponents) map[string]struct{} {
	reachable := make(map[string]struct{}, len(schemas))
	queue := make([]string, 0, len(schemas))
	enqueue := func(ref string) {
		name, ok := schemaNameFromRef(ref)
		if !ok {
			return
		}
		if _, seen := reachable[name]; seen {
			return
		}
		reachable[name] = struct{}{}
		queue = append(queue, name)
	}

	for _, pathItem := range paths {
		collectPathItemSchemaRefs(pathItem, enqueue)
	}
	collectReusableComponentSchemaRefs(reusable, enqueue)

	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if schema := schemas[name]; schema != nil {
			collectSchemaRefs(schema, enqueue)
		}
	}
	return reachable
}
