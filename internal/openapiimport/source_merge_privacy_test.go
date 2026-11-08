package openapiimport

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/stretchr/testify/require"
)

func TestSourceURICollectionRoles(t *testing.T) {
	const uri = "https://review-user:review-secret with-space@auth.test/key"
	for _, boundary := range []string{"node", "paths", "components", "array"} {
		for _, mode := range []string{"direct", "alias", "single", "sequence", "nested"} {
			t.Run(boundary+"/"+mode, func(t *testing.T) {
				source := sourceURICollectionFixture(uri, boundary, mode)
				var parsed yaml.Node
				require.NoError(t, yaml.Unmarshal(source, &parsed))
				keys, err := descriptionURIKeys(parsed.Content[0])
				require.NoError(t, err)
				found := 0
				seen := make(map[*yaml.Node]bool)
				var visit func(*yaml.Node)
				visit = func(node *yaml.Node) {
					if node == nil || seen[node] {
						return
					}
					seen[node] = true
					if node.Value == uri {
						found++
						require.True(t, keys[node], "grammar role retains original URI key node")
					}
					visit(node.Alias)
					for _, child := range node.Content {
						visit(child)
					}
				}
				visit(parsed.Content[0])
				require.Equal(t, 2, found)
				_, _, err = Analyze(source)
				require.ErrorContains(t, err, "already defined")
				requirePrivateOutput(t, err.Error())
			})
		}
	}
}

func sourceURICollectionFixture(uri, boundary, mode string) []byte {
	indent := func(value string, n int) string {
		prefix := strings.Repeat(" ", n)
		return prefix + strings.ReplaceAll(strings.TrimSuffix(value, "\n"), "\n", "\n"+prefix) + "\n"
	}
	requirement := fmt.Sprintf("%q: []\n%q: []\n", uri, uri)
	operation := "get:\n  security:\n    -\n" + indent(requirement, 6) + "  responses: {'204': {description: OK}}\n"
	base := operation
	switch boundary {
	case "paths":
		base = "/value:\n" + indent(operation, 2)
	case "components":
		base = "pathItems:\n  Value:\n" + indent(operation, 4)
	case "array":
		base = requirement
	}
	definitions := ""
	value := base
	if mode != "direct" {
		definitions = "x-base: &base\n" + indent(base, 2)
		switch mode {
		case "alias":
			value = "*base\n"
		case "single":
			value = "<<: *base\n"
		case "sequence":
			value = "<<: [*base, {x-extra: true}]\n"
		case "nested":
			definitions += "x-nested: &nested\n  <<: *base\n"
			value = "<<: *nested\n"
		}
	}
	field := func(prefix string, n int) string {
		if mode == "alias" {
			return prefix + " *base\n"
		}
		return prefix + "\n" + indent(value, n)
	}
	body := ""
	switch boundary {
	case "node":
		body = "paths:\n" + field("  /value:", 4)
	case "paths":
		body = field("paths:", 2)
	case "components":
		body = "paths: {}\n" + field("components:", 2)
	case "array":
		if mode == "alias" {
			definitions += "x-items: &items [*base]\n"
			body = "paths:\n  /value:\n    get:\n      security: *items\n      responses: {'204': {description: OK}}\n"
		} else {
			body = "paths:\n  /value:\n    get:\n      security:\n        -\n" + indent(value, 10) + "      responses: {'204': {description: OK}}\n"
		}
	}
	return []byte("openapi: 3.2.0\ninfo: {title: Source roles, version: '1'}\n" + definitions + body)
}

func TestSourceURICollectionOverrideRoles(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		entry := `openapi: 3.2.0
x-first: &first
  /value:
    get:
      security: [{"https://first.test/key": []}]
x-second: &second
  /value:
    get:
      security: [{"https://second.test/key": []}]
paths:
  <<: [*first, *second]
`
		if explicit {
			entry += "  /value:\n    get:\n      security: []\n"
		}
		var parsed yaml.Node
		require.NoError(t, yaml.Unmarshal([]byte(entry), &parsed))
		keys, err := descriptionURIKeys(parsed.Content[0])
		require.NoError(t, err)
		names := make(map[string]bool)
		for node := range keys {
			names[node.Value] = true
		}
		require.Equal(t, !explicit, names["https://first.test/key"])
		require.False(t, names["https://second.test/key"])
	}
}
