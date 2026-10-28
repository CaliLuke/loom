package openapiimport

import (
	"strconv"

	"github.com/pb33f/go-yaml"
)

// numericPosition identifies an authored scalar in the single admitted source
// document. External references are rejected before normalization.
type numericPosition struct {
	line, column int
}

// numericSourcePositions binds pristine tokens before the parser constructs
// semantic reference views. Copied or inherited nodes retain their source
// coordinates even when their occurrence's JSON Pointer changes.
func numericSourcePositions(root *yaml.Node, sources map[string]numericSource) map[numericPosition]numericSource {
	positions := make(map[numericPosition]numericSource)
	var walk func(*yaml.Node, string)
	walk = func(node *yaml.Node, path string) {
		if node == nil {
			return
		}
		switch node.Kind {
		case yaml.DocumentNode:
			if len(node.Content) == 1 {
				walk(node.Content[0], path)
			}
		case yaml.MappingNode:
			for index := 0; index+1 < len(node.Content); index += 2 {
				walk(node.Content[index+1], path+"/"+escapeJSONPointer(node.Content[index].Value))
			}
		case yaml.SequenceNode:
			for index, child := range node.Content {
				walk(child, path+"/"+strconv.Itoa(index))
			}
		case yaml.ScalarNode:
			if node.Line > 0 && node.Column > 0 {
				source, found := sources[path]
				if !found {
					// Inline YAML merge sources have no effective JSON Pointer
					// at their authored location. Capture their pristine scalar.
					source = numericSource{text: node.Value, tag: node.Tag, style: node.Style}
				}
				positions[numericPosition{node.Line, node.Column}] = source
			}
		}
	}
	walk(root, "#")
	return positions
}

func (a *analyzer) nodeNumericSource(node *yaml.Node) (numericSource, bool) {
	if node == nil {
		return numericSource{}, false
	}
	if source, found := a.numericPositions[numericPosition{node.Line, node.Column}]; found {
		return source, true
	}
	// Programmatic schemas have no original source coordinates. Their own
	// node is authoritative; an occurrence path must never supply another value.
	return numericSource{text: node.Value, tag: node.Tag, style: node.Style}, true
}
