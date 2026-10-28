package openapiimport

import (
	"github.com/pb33f/go-yaml"
	"strings"
)

// referenceDescriptionOverride preserves absent versus explicitly empty
// description fields on reference occurrences.
func referenceDescriptionOverride(node *yaml.Node) *string {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for index := 0; index+1 < len(node.Content); index += 2 {
		if node.Content[index].Value == "description" {
			value := node.Content[index+1].Value
			return &value
		}
	}
	return nil
}

func (a *analyzer) referenceAnnotations(node *yaml.Node) (*string, *string) {
	if strings.HasPrefix(a.version, "3.0.") {
		return nil, nil
	}
	var description, summary *string
	if node != nil && node.Kind == yaml.MappingNode {
		for index := 0; index+1 < len(node.Content); index += 2 {
			value := node.Content[index+1].Value
			switch node.Content[index].Value {
			case "description":
				description = &value
			case "summary":
				summary = &value
			}
		}
	}
	return description, summary
}
