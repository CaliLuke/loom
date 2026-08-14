package openapiimport

import (
	"encoding/json"
	"fmt"

	"github.com/pb33f/libopenapi/orderedmap"
	yaml4 "go.yaml.in/yaml/v4"
)

func (a *analyzer) extensions(path string, extensions *orderedmap.Map[string, *yaml4.Node]) map[string]any {
	if orderedmap.Len(extensions) == 0 {
		return nil
	}
	result := make(map[string]any, orderedmap.Len(extensions))
	for name, node := range extensions.FromOldest() {
		var value any
		if node == nil {
			value = nil
		} else if err := node.Decode(&value); err != nil {
			a.unsupported("vendor-extension", path+"/"+escapeJSONPointer(name), fmt.Sprintf("vendor extension cannot be decoded: %v", err))
			continue
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			a.unsupported("vendor-extension", path+"/"+escapeJSONPointer(name), fmt.Sprintf("vendor extension is not JSON-compatible: %v", err))
			continue
		}
		if err := json.Unmarshal(encoded, &value); err != nil {
			a.unsupported("vendor-extension", path+"/"+escapeJSONPointer(name), fmt.Sprintf("vendor extension cannot be normalized: %v", err))
			continue
		}
		result[name] = value
	}
	return result
}

func (a *analyzer) unsupportedExtensions(path string, extensions *orderedmap.Map[string, *yaml4.Node]) {
	if orderedmap.Len(extensions) > 0 {
		a.unsupported("vendor-extension", path, "vendor extensions at this location are not in the strict import subset")
	}
}
