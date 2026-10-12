package exactnum

import (
	"strings"

	"go.yaml.in/yaml/v3"
)

// MarshalYAML emits an exact numeric scalar. Explicit node tags retain numeric
// meaning even when a YAML implementation cannot decode the token into float64.
func (n Number) MarshalYAML() (any, error) {
	token := n.String()
	tag := "!!int"
	if strings.ContainsAny(token, ".eE") {
		tag = "!!float"
	}
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: token}, nil
}
