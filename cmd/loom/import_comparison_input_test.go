package main

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"reflect"
	"strings"
)

// prepareGeneratedOpenAPIContract adapts known generated representation choices
// for semantic comparison. Authored documents still use strict Analyze directly.
// Raw values preserve numbers and literal data through this spelling-only pass.
func prepareGeneratedOpenAPIContract(source []byte, title string) ([]byte, error) {
	var document map[string]jsontext.Value
	if err := json.Unmarshal(source, &document); err != nil {
		return nil, err
	}
	var dialect, version string
	if err := json.Unmarshal(document["jsonSchemaDialect"], &dialect); err != nil {
		return nil, err
	}
	if dialect != "https://spec.openapis.org/oas/3.1/dialect/base" {
		return nil, fmt.Errorf("unexpected generated dialect %q", dialect)
	}
	if err := json.Unmarshal(document["openapi"], &version); err != nil {
		return nil, err
	}
	var servers []map[string]string
	if err := json.Unmarshal(document["servers"], &servers); err != nil {
		return nil, err
	}
	want := map[string]string{"url": "http://localhost:80", "description": "Default server for " + title}
	if strings.HasPrefix(version, "3.2.") {
		want["name"] = title
	}
	if !reflect.DeepEqual(servers, []map[string]string{want}) {
		return nil, fmt.Errorf("unexpected generated servers: %v", servers)
	}
	delete(document, "servers")
	delete(document, "jsonSchemaDialect")
	raw, err := json.Marshal(document, json.Deterministic(true))
	if err != nil {
		return nil, err
	}
	return rewriteComparisonObject(raw, "document")
}

// rewriteComparisonObject follows OpenAPI/schema structure, never arbitrary JSON
// objects in defaults, examples, extensions, or discriminator mappings.
func rewriteComparisonObject(raw jsontext.Value, kind string) (jsontext.Value, error) {
	if kind == "schema" && (raw.Kind() == 't' || raw.Kind() == 'f') {
		return raw, nil
	}
	var object map[string]jsontext.Value
	if err := json.Unmarshal(raw, &object); err != nil {
		return nil, fmt.Errorf("comparison %s: %w", kind, err)
	}
	if kind == "schema" && object["$ref"] != nil && len(object) > 1 {
		if object["allOf"] != nil {
			return nil, fmt.Errorf("comparison cannot normalize $ref together with allOf")
		}
		reference, err := json.Marshal([]map[string]jsontext.Value{{"$ref": object["$ref"]}})
		if err != nil {
			return nil, err
		}
		object["allOf"] = reference
		delete(object, "$ref")
	}
	for key, value := range object {
		child, mode := comparisonChild(kind, key)
		if child == "" {
			continue
		}
		var err error
		switch mode {
		case "map":
			var entries map[string]jsontext.Value
			if err = json.Unmarshal(value, &entries); err == nil {
				for name, entry := range entries {
					entries[name], err = rewriteComparisonObject(entry, child)
					if err != nil {
						break
					}
				}
				if err == nil {
					value, err = json.Marshal(entries, json.Deterministic(true))
				}
			}
		case "array":
			var entries []jsontext.Value
			if err = json.Unmarshal(value, &entries); err == nil {
				for index, entry := range entries {
					entries[index], err = rewriteComparisonObject(entry, child)
					if err != nil {
						break
					}
				}
				if err == nil {
					value, err = json.Marshal(entries, json.Deterministic(true))
				}
			}
		default:
			value, err = rewriteComparisonObject(value, child)
		}
		if err != nil {
			return nil, fmt.Errorf("comparison %s/%s: %w", kind, key, err)
		}
		object[key] = value
	}
	return json.Marshal(object, json.Deterministic(true))
}

func comparisonChild(kind, key string) (string, string) {
	switch kind {
	case "document":
		switch key {
		case "components":
			return "components", ""
		case "paths", "webhooks":
			return "path", "map"
		}
	case "components":
		switch key {
		case "schemas":
			return "schema", "map"
		case "parameters", "headers", "requestBodies", "responses":
			return "carrier", "map"
		case "pathItems":
			return "path", "map"
		}
	case "path":
		switch key {
		case "get", "put", "post", "delete", "options", "head", "patch", "trace", "query":
			return "operation", ""
		case "parameters":
			return "carrier", "array"
		}
	case "operation":
		switch key {
		case "parameters":
			return "carrier", "array"
		case "requestBody":
			return "carrier", ""
		case "responses":
			return "carrier", "map"
		}
	case "carrier":
		switch key {
		case "schema":
			return "schema", ""
		case "content":
			return "media", "map"
		case "headers":
			return "carrier", "map"
		}
	case "media":
		if key == "schema" {
			return "schema", ""
		}
	case "schema":
		switch key {
		case "properties", "patternProperties", "$defs", "dependentSchemas":
			return "schema", "map"
		case "allOf", "anyOf", "oneOf", "prefixItems":
			return "schema", "array"
		case "items", "additionalProperties", "unevaluatedProperties", "propertyNames", "contains", "not", "if", "then", "else":
			return "schema", ""
		}
	}
	return "", ""
}
