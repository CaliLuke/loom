package openapiimport

import "github.com/pb33f/go-yaml"

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
