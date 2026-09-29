package ir

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strconv"
	"strings"
)

type representationGraphNode struct {
	label   string
	encoded []byte
	edges   []string
	paths   []string
}

// completeSchemaFingerprints compares finite, labeled reference graphs. It
// preserves every serialized non-reference keyword and annotation; it does not
// decide general JSON Schema equivalence. Refinement only splits classes, and
// quotient traversal makes an extra unfolding of a recursive node immaterial.
func completeSchemaFingerprints(schemas map[string]*Schema) map[string]string {
	graph := captureRepresentationGraph(schemas)
	signatures := make(map[string]string, len(graph))
	for name, node := range graph {
		signatures[name] = node.label
	}
	classes, count := representationClasses(signatures)
	for {
		for name, node := range graph {
			var signature strings.Builder
			signature.WriteString(node.label)
			signature.WriteByte(0)
			signature.WriteString(strconv.Itoa(classes[name]))
			for _, edge := range node.edges {
				signature.WriteByte(':')
				signature.WriteString(strconv.Itoa(classes[edge]))
			}
			signatures[name] = signature.String()
		}
		next, nextCount := representationClasses(signatures)
		classes = next
		if nextCount == count {
			break
		}
		count = nextCount
	}
	return fingerprintRepresentationQuotient(graph, classes)
}

// Capture serializes each schema once, including custom annotation values.
// Subsequent refinement never calls user codecs. The reference walker visits
// maps in key order and lists in index order, so edge slots identify schema paths.
func captureRepresentationGraph(schemas map[string]*Schema) map[string]representationGraphNode {
	graph := make(map[string]representationGraphNode, len(schemas))
	for _, name := range slices.Sorted(maps.Keys(schemas)) {
		node := representationGraphNode{}
		normalized := mapSchemaReferences(schemas[name], func(ref, path string) string {
			if target, local := schemaComponentName(ref); local && schemas[target] != nil {
				node.edges = append(node.edges, target)
				node.paths = append(node.paths, path)
			}
			return ref
		})
		encoded, err := json.Marshal(normalized, json.Deterministic(true))
		if err != nil {
			panic(fmt.Errorf("OpenAPI representation fingerprint: %w", err))
		}
		node.encoded = encoded
		references := make(map[string]string, len(node.paths))
		for index, path := range node.paths {
			references[path] = "local:" + strconv.Itoa(index)
		}
		// Paths distinguish local slots even if an external reference or authored
		// annotation happens to contain the same spelling as a placeholder.
		paths, err := json.Marshal(node.paths)
		if err != nil {
			panic(fmt.Errorf("OpenAPI representation reference paths: %w", err))
		}
		node.label = rewriteCapturedReferences(encoded, references) + "\x00" + string(paths)
		graph[name] = node
	}
	return graph
}

func representationClasses(signatures map[string]string) (map[string]int, int) {
	ordered := slices.Sorted(maps.Values(signatures))
	ordered = slices.Compact(ordered)
	identities := make(map[string]int, len(ordered))
	for index, signature := range ordered {
		identities[signature] = index
	}
	classes := make(map[string]int, len(signatures))
	for name, signature := range signatures {
		classes[name] = identities[signature]
	}
	return classes, len(ordered)
}

func fingerprintRepresentationQuotient(graph map[string]representationGraphNode, classes map[string]int) map[string]string {
	quotient := make(map[int]representationGraphNode)
	for _, name := range slices.Sorted(maps.Keys(graph)) {
		quotient[classes[name]] = graph[name]
	}
	var expand func(int, map[int]int) string
	expand = func(class int, active map[int]int) string {
		if depth, found := active[class]; found {
			return "recursive:" + strconv.Itoa(depth)
		}
		active[class] = len(active)
		node := quotient[class]
		references := make(map[string]string, len(node.edges))
		for index, edge := range node.edges {
			references[node.paths[index]] = "schema:" + expand(classes[edge], active)
		}
		delete(active, class)
		return fingerprintString(rewriteCapturedReferences(node.encoded, references))
	}
	fingerprints := make(map[int]string, len(quotient))
	for class := range quotient {
		fingerprints[class] = expand(class, make(map[int]int))
	}
	result := make(map[string]string, len(graph))
	for name, class := range classes {
		result[name] = fingerprints[class]
	}
	return result
}

// rewriteCapturedReferences preserves the prior fingerprint serialization. Only
// strings at structurally captured reference paths are replaced; arbitrary
// annotation strings, exact numeric tokens and all other fields are retained.
func rewriteCapturedReferences(encoded []byte, references map[string]string) string {
	decoder := jsontext.NewDecoder(bytes.NewReader(encoded))
	var output bytes.Buffer
	encoder := jsontext.NewEncoder(&output)
	for {
		token, err := decoder.ReadToken()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			panic(fmt.Errorf("read captured OpenAPI schema: %w", err))
		}
		kind, length := decoder.StackIndex(decoder.StackDepth())
		if token.Kind() == jsontext.KindString && kind == jsontext.KindBeginObject && length%2 == 0 {
			if reference, found := references[string(decoder.StackPointer())]; found {
				token = jsontext.String(reference)
			}
		}
		if err := encoder.WriteToken(token); err != nil {
			panic(fmt.Errorf("write captured OpenAPI schema: %w", err))
		}
	}
	return strings.TrimSuffix(output.String(), "\n")
}
