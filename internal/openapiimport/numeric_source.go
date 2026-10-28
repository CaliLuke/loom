package openapiimport

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"math/big"
	"slices"
	"strconv"
	"strings"

	"github.com/CaliLuke/libopenapi/datamodel/high/base"
	"github.com/CaliLuke/libopenapi/utils"
	"github.com/pb33f/go-yaml"

	"github.com/CaliLuke/loom/internal/exactnum"
)

type numericSource struct {
	text  string
	kind  byte
	tag   string
	style yaml.Style
}

// numericSources retains source token distinctions before libopenapi converts
// numeric values into finite carriers or changes inferred YAML scalar tags.
func numericSources(source []byte) (map[string]numericSource, error) {
	values := make(map[string]numericSource)
	if bytes.HasPrefix(bytes.TrimSpace(source), []byte("{")) && jsontext.Value(source).IsValid() {
		if err := collectJSONNumericSources(jsontext.Value(source), "#", values); err != nil {
			return nil, err
		}
		return values, nil
	}
	var document yaml.Node
	if err := yaml.Unmarshal(source, &document); err != nil {
		return nil, err
	}
	if len(document.Content) != 1 {
		return nil, fmt.Errorf("missing source document")
	}
	if err := collectYAMLNumericSources(document.Content[0], "#", values, make(map[*yaml.Node]bool)); err != nil {
		return nil, err
	}
	return values, nil
}

func collectJSONNumericSources(wire jsontext.Value, path string, values map[string]numericSource) error {
	switch wire.Kind() {
	case '{':
		var members map[string]jsontext.Value
		if err := json.Unmarshal(wire, &members); err != nil {
			return err
		}
		for name, child := range members {
			if err := collectJSONNumericSources(child, path+"/"+escapeJSONPointer(name), values); err != nil {
				return err
			}
		}
	case '[':
		var members []jsontext.Value
		if err := json.Unmarshal(wire, &members); err != nil {
			return err
		}
		for index, child := range members {
			if err := collectJSONNumericSources(child, path+"/"+strconv.Itoa(index), values); err != nil {
				return err
			}
		}
	default:
		values[path] = numericSource{text: string(wire), kind: byte(wire.Kind())}
	}
	return nil
}

func collectYAMLNumericSources(node *yaml.Node, path string, values map[string]numericSource, active map[*yaml.Node]bool) error {
	if node == nil || active[node] {
		return fmt.Errorf("invalid or cyclic YAML source at %s", path)
	}
	active[node] = true
	defer delete(active, node)
	switch node.Kind {
	case yaml.AliasNode:
		return collectYAMLNumericSources(node.Alias, path, values, active)
	case yaml.MappingNode:
		members, err := effectiveYAMLMapping(node, make(map[*yaml.Node]bool))
		if err != nil {
			return fmt.Errorf("YAML mapping at %s: %w", path, err)
		}
		keys := make([]string, 0, len(members))
		for key := range members {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		for _, key := range keys {
			if err := collectYAMLNumericSources(members[key], path+"/"+escapeJSONPointer(key), values, active); err != nil {
				return err
			}
		}
	case yaml.SequenceNode:
		for index, child := range node.Content {
			if err := collectYAMLNumericSources(child, path+"/"+strconv.Itoa(index), values, active); err != nil {
				return err
			}
		}
	case yaml.ScalarNode:
		values[path] = numericSource{text: node.Value, tag: node.Tag, style: node.Style}
	}
	return nil
}

// effectiveYAMLMapping selects whole values before traversing their children.
// YAML merges are shallow: explicit keys replace inherited subtrees, and the
// first mapping in a merge sequence supplies each otherwise absent key.
func effectiveYAMLMapping(node *yaml.Node, active map[*yaml.Node]bool) (map[string]*yaml.Node, error) {
	if node == nil || active[node] {
		return nil, fmt.Errorf("invalid or cyclic merge")
	}
	active[node] = true
	defer delete(active, node)
	if node.Kind == yaml.AliasNode {
		return effectiveYAMLMapping(node.Alias, active)
	}
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("merge requires a mapping")
	}
	members := make(map[string]*yaml.Node)
	for index := 0; index+1 < len(node.Content); index += 2 {
		if node.Content[index].Tag != "!!merge" {
			members[node.Content[index].Value] = node.Content[index+1]
		}
	}
	for index := 0; index+1 < len(node.Content); index += 2 {
		if node.Content[index].Tag != "!!merge" {
			continue
		}
		merged, err := yamlMergeSources(node.Content[index+1])
		if err != nil {
			return nil, err
		}
		for _, source := range merged {
			inherited, err := effectiveYAMLMapping(source, active)
			if err != nil {
				return nil, err
			}
			for key, value := range inherited {
				if _, present := members[key]; !present {
					members[key] = value
				}
			}
		}
	}
	return members, nil
}

// yamlMergeSources resolves the merge value before deciding whether it names
// one mapping or a sequence of mappings. Sequence members themselves must still
// resolve to mappings; nested sequences do not acquire merge semantics.
func yamlMergeSources(node *yaml.Node) ([]*yaml.Node, error) {
	seen := make(map[*yaml.Node]bool)
	for node != nil && node.Kind == yaml.AliasNode {
		if seen[node] {
			return nil, fmt.Errorf("cyclic merge alias")
		}
		seen[node] = true
		node = node.Alias
	}
	if node == nil {
		return nil, fmt.Errorf("invalid merge alias")
	}
	if node.Kind == yaml.SequenceNode {
		return node.Content, nil
	}
	return []*yaml.Node{node}, nil
}

func (a *analyzer) schemaNumericBounds(schema *Schema, authored *base.Schema, path string) {
	var nodes map[string]*yaml.Node
	if low := authored.GoLow(); low != nil {
		nodes = map[string]*yaml.Node{
			"minimum":          low.Minimum.ValueNode,
			"maximum":          low.Maximum.ValueNode,
			"exclusiveMinimum": low.ExclusiveMinimum.ValueNode,
			"exclusiveMaximum": low.ExclusiveMaximum.ValueNode,
		}
		// A non-finite carrier conversion can leave a low-level numeric field
		// empty. Its authored keyword still owns a token and must be diagnosed.
		root := utils.ResolveMergeNodes(low.RootNode)
		if root != nil && root.Kind == yaml.MappingNode {
			for index := 0; index+1 < len(root.Content); index += 2 {
				name := root.Content[index].Value
				if node, selected := nodes[name]; selected && node == nil {
					nodes[name] = root.Content[index+1]
				}
			}
		}
	}
	for _, endpoint := range []struct {
		name  string
		value **exactnum.Number
	}{{"minimum", &schema.Minimum}, {"maximum", &schema.Maximum}, {"exclusiveMinimum", &schema.ExclusiveMinimum}, {"exclusiveMaximum", &schema.ExclusiveMaximum}} {
		source, present := a.numbers[path+"/"+endpoint.name]
		if nodes != nil {
			source, present = a.nodeNumericSource(nodes[endpoint.name])
		}
		if !present {
			continue
		}
		if strings.HasPrefix(endpoint.name, "exclusive") && strings.HasPrefix(a.version, "3.0.") && source.boolean() {
			if strings.EqualFold(source.text, "true") {
				inclusive := &schema.Minimum
				if endpoint.name == "exclusiveMaximum" {
					inclusive = &schema.Maximum
				}
				*endpoint.value, *inclusive = *inclusive, nil
			}
			continue
		}
		number, err := source.number()
		if err != nil {
			a.unsupported("numeric-bound", path+"/"+endpoint.name, err.Error())
			continue
		}
		// The exact source may be valid JSON/YAML while its magnitude is outside
		// the retained public DSL admission contract. Never emit a known-invalid
		// design or silently omit the offending numeric constraint.
		if _, err := exactnum.ParseDSL(number.String()); err != nil {
			a.unsupported("numeric-bound", path+"/"+endpoint.name, fmt.Sprintf("numeric bound %s cannot be represented by the retained finite DSL admission contract", source.text))
			continue
		}
		*endpoint.value = &number
	}
}

func (s numericSource) boolean() bool {
	return s.kind == 't' || s.kind == 'f' || s.kind == 0 && s.tag == "!!bool"
}

func (s numericSource) number() (exactnum.Number, error) {
	if s.kind != 0 {
		if s.kind != '0' {
			return exactnum.Number{}, fmt.Errorf("numeric bound must be a JSON number, got %s", s.text)
		}
		return exactnum.ParseJSON(s.text)
	}
	if s.tag == "!!int" {
		text := strings.ReplaceAll(s.text, "_", "")
		sign := ""
		if strings.HasPrefix(text, "+") || strings.HasPrefix(text, "-") {
			sign, text = text[:1], text[1:]
		}
		base := 10
		for prefix, radix := range map[string]int{"0x": 16, "0o": 8, "0b": 2} {
			if strings.HasPrefix(strings.ToLower(text), prefix) {
				base, text = radix, text[2:]
				break
			}
		}
		integer, ok := new(big.Int).SetString(sign+text, base)
		if !ok {
			return exactnum.Number{}, fmt.Errorf("invalid YAML integer bound %s", s.text)
		}
		return exactnum.FromInteger(integer), nil
	}
	if s.tag == "!!float" || s.style == 0 {
		// Plain JSON decimal grammar is a YAML numeric subset even when the
		// finite YAML decoder inferred !!str after a range failure. Only a real
		// float tag admits YAML's additional separators/dot/leading-plus forms.
		text := s.text
		if s.tag == "!!float" {
			var err error
			text, err = yamlFloatJSON(text)
			if err != nil {
				return exactnum.Number{}, err
			}
		}
		if number, err := exactnum.ParseJSON(text); err == nil {
			return number, nil
		}
	}
	return exactnum.Number{}, fmt.Errorf("numeric bound must be a finite YAML number, got %q", s.text)
}

// yamlFloatJSON normalizes only the additional decimal spelling admitted by a
// YAML float tag. ParseJSON still owns final exact-number grammar validation.
func yamlFloatJSON(text string) (string, error) {
	mantissa := strings.SplitN(strings.ToLower(text), "e", 2)[0]
	if !strings.ContainsAny(mantissa, "0123456789") {
		return "", fmt.Errorf("invalid YAML float bound %q", text)
	}
	text = strings.ReplaceAll(text, "_", "")
	text = strings.TrimPrefix(text, "+")
	if strings.HasPrefix(text, ".") {
		text = "0" + text
	} else if strings.HasPrefix(text, "-.") {
		text = "-0" + text[1:]
	}
	// YAML decimal mantissas may have leading zeroes; JSON may not.
	sign := ""
	if strings.HasPrefix(text, "-") {
		sign, text = "-", text[1:]
	}
	end := strings.IndexAny(text, ".eE")
	if end < 0 {
		end = len(text)
	}
	whole := strings.TrimLeft(text[:end], "0")
	if whole == "" {
		whole = "0"
	}
	text = sign + whole + text[end:]
	if at := strings.IndexAny(text, "eE"); at >= 0 && at > 0 && text[at-1] == '.' {
		text = text[:at] + "0" + text[at:]
	} else if strings.HasSuffix(text, ".") {
		text += "0"
	}
	return text, nil
}
