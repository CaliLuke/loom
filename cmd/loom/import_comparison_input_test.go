package main

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"reflect"
	"strings"
)

// prepareGeneratedOpenAPIContract removes only verified generated server and
// dialect annotations. Schema and literal values remain untouched; strict Analyze
// owns admission of reference siblings and every other schema assertion.
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
	return json.Marshal(document, json.Deterministic(true))
}
