package openapiimport

import (
	"encoding/json/jsontext"
	"fmt"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/pb33f/go-yaml"
)

type (
	provenance         map[string]SourceLocation
	bundledDescription struct {
		root            *yaml.Node
		numbers         map[string]numericSource
		origins         provenance
		blocked         map[string]Diagnostic
		diagnostics     Diagnostics
		unresolvedPaths []string
		graph           *resolution
		declarations    map[string]declarationOrigin
		references      map[string]string
		keyFailures     map[string]Diagnostic
		securityKeys    map[string]resolvedSecurityKey
		securityURIs    map[string]string
	}
	portableKey struct {
		class    int
		document string
		pointer  []string
		kind     string
	}
	rawKeyAllocator map[string]map[string]bool
)

type descriptionBundler struct {
	result  *bundledDescription
	edges   map[targetKey]*edge
	targets map[targetKey]string
	kinds   map[targetKey]string
	blocked map[*edge]string
	ordered []targetKey
}

func bundleResolution(r *resolution) (*bundledDescription, error) {
	b := &descriptionBundler{
		result: &bundledDescription{numbers: make(map[string]numericSource), origins: make(provenance), blocked: make(map[string]Diagnostic), references: make(map[string]string), keyFailures: make(map[string]Diagnostic), securityKeys: make(map[string]resolvedSecurityKey), graph: r},
		edges:  make(map[targetKey]*edge), targets: make(map[targetKey]string), kinds: make(map[targetKey]string), blocked: make(map[*edge]string),
	}
	if err := b.allocate(); err != nil {
		return nil, err
	}
	root, err := b.copyNode(targetKey{"entry", ""}, "")
	if err != nil {
		return nil, err
	}
	// Emit a block mapping even for JSON input. Flow YAML begins with an opening
	// brace but is not necessarily JSON, which format detection would assume.
	root.Style = 0
	b.result.root = root
	for _, target := range b.ordered {
		pointer := b.targets[target]
		node, err := b.copyNode(target, pointer)
		if err != nil {
			return nil, err
		}
		b.addComponent(pointer, node)
	}
	blocked := make([]*edge, 0, len(b.blocked))
	for e := range b.blocked {
		blocked = append(blocked, e)
	}
	sort.Slice(blocked, func(i, j int) bool { return b.blocked[blocked[i]] < b.blocked[blocked[j]] })
	for _, e := range blocked {
		node, err := blockedComponentBody(e.kind)
		if err != nil {
			return nil, err
		}
		pointer := b.blocked[e]
		b.addComponent(pointer, node)
		origin := b.origin(e.at)
		origin.Authored = safeAuthoredReference(e.authored)
		diagnostic := Diagnostic{Code: e.failure.Code, Path: "#" + pointer, Origin: &origin, Message: e.failure.Message}
		b.result.blocked["#"+pointer] = diagnostic
		b.result.diagnostics = append(b.result.diagnostics, diagnostic)
		b.result.origins["#"+pointer] = origin
	}
	b.result.diagnostics = b.result.diagnostics.sorted()
	slices.Sort(b.result.unresolvedPaths)
	b.result.indexDeclarationOrigins()
	if err := b.result.projectSecurityMetadata(); err != nil {
		return nil, err
	}
	return b.result, nil
}

func blockedComponentBody(kind string) (*yaml.Node, error) {
	bodies := map[string]string{
		"schemas": "{}", "parameters": "{name: x-loom-unresolved, in: query, schema: {}}",
		"responses": "{description: Unresolved}", "requestBodies": "{content: {}}",
		"headers": "{schema: {}}", "examples": "{}",
		"securitySchemes": "{type: apiKey, name: x-loom-unresolved, in: header}",
	}
	body, ok := bodies[kind]
	if !ok {
		return nil, fmt.Errorf("unsupported blocked component kind %q", kind)
	}
	node, _, err := parseDescriptionTree([]byte(body))
	return node, err
}

func (b *descriptionBundler) allocate() error {
	reachable := b.reachableEdges()
	var blocked []*edge
	for e := range reachable {
		b.edges[e.at] = e
		if e.failure != nil && e.class == "component" {
			blocked = append(blocked, e)
		}
		if e.target != nil && e.class != "pathItem" {
			tokens := slices.Collect(jsontext.Pointer(e.target.pointer).Tokens())
			entryComponent := e.target.doc == "entry" && len(tokens) == 3 && tokens[0] == "components" && tokens[1] == e.kind
			if entryComponent {
				b.targets[*e.target] = e.target.pointer
			} else {
				b.kinds[*e.target] = e.kind
			}
		}
	}
	for target := range b.kinds {
		b.ordered = append(b.ordered, target)
	}
	sort.Slice(b.ordered, func(i, j int) bool {
		return b.less(b.ordered[i], b.ordered[j], b.kinds[b.ordered[i]], b.kinds[b.ordered[j]])
	})
	claims := b.entryClaims()
	for _, target := range b.ordered {
		kind := b.kinds[target]
		base, err := b.targetBase(target)
		if err != nil {
			return err
		}
		name := claims.claim(kind, base)
		b.targets[target] = "/components/" + kind + "/" + escapeJSONPointer(name)
	}
	sort.Slice(blocked, func(i, j int) bool { return b.less(blocked[i].at, blocked[j].at, blocked[i].kind, blocked[j].kind) })
	next := make(map[string]int)
	for _, e := range blocked {
		for {
			next[e.kind]++
			name := fmt.Sprintf("x-loom-unresolved-%d", next[e.kind])
			if claims[e.kind][name] {
				continue
			}
			claims.claim(e.kind, name)
			b.blocked[e] = "/components/" + e.kind + "/" + name
			break
		}
	}
	for e := range reachable {
		if e.class == "discriminatorMapping" && e.target != nil {
			e.bundled = "#" + emitPointer(b.targets[*e.target])
		}
	}
	return nil
}

func (a rawKeyAllocator) claim(kind, base string) string {
	if a[kind] == nil {
		a[kind] = make(map[string]bool)
	}
	name := base
	for n := 2; a[kind][name]; n++ {
		name = fmt.Sprintf("%s_%d", base, n)
	}
	a[kind][name] = true
	return name
}

func (b *descriptionBundler) less(a, c targetKey, akind, ckind string) bool {
	key := func(t targetKey, kind string) portableKey {
		document := b.result.graph.registry.docs[t.doc].portable
		class := 1
		if t.doc == "entry" {
			class = -1
		} else if strings.HasPrefix(document, "root:") {
			class = 0
		}
		return portableKey{class, document, slices.Collect(jsontext.Pointer(t.pointer).Tokens()), kind}
	}
	x, y := key(a, akind), key(c, ckind)
	if x.class != y.class {
		return x.class < y.class
	}
	if x.document != y.document {
		return x.document < y.document
	}
	if comparison := slices.Compare(x.pointer, y.pointer); comparison != 0 {
		return comparison < 0
	}
	return x.kind < y.kind
}

func (b *descriptionBundler) origin(at targetKey) SourceLocation {
	return SourceLocation{Document: b.result.graph.registry.docs[at.doc].portable, Pointer: "#" + at.pointer}
}

func (b *descriptionBundler) copyNode(at targetKey, destination string) (*yaml.Node, error) {
	doc := b.result.graph.registry.docs[at.doc]
	node, err := descriptionNode(doc.root, at.pointer)
	if err != nil {
		return nil, err
	}
	origin := b.origin(at)
	b.result.origins["#"+destination] = origin
	if reference := b.promotedSchemaReference(at, destination, origin); reference != nil {
		return reference, nil
	}
	b.recordOccurrence(at, destination, origin)
	if number, ok := doc.numbers["#"+at.pointer]; ok {
		b.result.numbers["#"+destination] = number
	}
	if at.doc == "entry" && strings.HasPrefix(destination, "/paths/") && len(slices.Collect(jsontext.Pointer(destination).Tokens())) == 2 {
		return b.copyPathItem(at, destination)
	}
	copy := *node
	copy.Content = nil
	if e := b.edges[at]; e != nil && e.class != "securityKey" {
		replacement := b.referenceValue(e)
		if replacement != "" {
			copy = *descriptionString(replacement)
			origin.Authored = safeAuthoredReference(e.authored)
			b.result.origins["#"+destination] = origin
		}
	}
	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i].Value
			suffix := "/" + escapeJSONPointer(key)
			child, err := b.copyNode(targetKey{at.doc, at.pointer + suffix}, destination+suffix)
			if err != nil {
				return nil, err
			}
			copy.Content = append(copy.Content, descriptionString(key), child)
		}
	case yaml.SequenceNode:
		for i := range node.Content {
			suffix := fmt.Sprintf("/%d", i)
			child, err := b.copyNode(targetKey{at.doc, at.pointer + suffix}, destination+suffix)
			if err != nil {
				return nil, err
			}
			copy.Content = append(copy.Content, child)
		}
	}
	return &copy, nil
}

func (b *descriptionBundler) referenceValue(e *edge) string {
	if e.external != "" {
		return e.external
	}
	if e.target != nil && e.class != "pathItem" {
		return "#" + emitPointer(b.targets[*e.target])
	}
	if pointer := b.blocked[e]; pointer != "" {
		return "#" + emitPointer(pointer)
	}
	return ""
}

// promotedSchemaReference joins copied Schema occurrences to their allocated
// declaration. The canonical destination itself must materialize exactly once.
func (b *descriptionBundler) promotedSchemaReference(at targetKey, destination string, origin SourceLocation) *yaml.Node {
	pointer := b.targets[at]
	if b.kinds[at] != "schemas" || destination == pointer {
		return nil
	}
	// The introduced reference retains the source Schema identity; it does not
	// invent an authored $ref occurrence.
	occurrence := "#" + destination + "/$ref"
	b.result.origins[occurrence] = origin
	b.result.references[occurrence] = "#" + pointer
	return &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
		Content: []*yaml.Node{
			descriptionString("$ref"), descriptionString("#" + emitPointer(pointer)),
		},
	}
}

func (b *descriptionBundler) addComponent(pointer string, node *yaml.Node) {
	tokens := slices.Collect(jsontext.Pointer(pointer).Tokens())
	current := b.result.root
	for _, token := range tokens[:len(tokens)-1] {
		var next *yaml.Node
		for i := 0; i+1 < len(current.Content); i += 2 {
			if current.Content[i].Value == token {
				next = current.Content[i+1]
				break
			}
		}
		if next == nil {
			next = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			current.Content = append(current.Content, descriptionString(token), next)
		}
		current = next
	}
	current.Content = append(current.Content, descriptionString(tokens[len(tokens)-1]), node)
}

func descriptionString(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}

func safeAuthoredReference(value string) string {
	if _, err := parseDescriptionURI(value); err != nil {
		return ""
	}
	return value
}

func (b *descriptionBundler) reachableEdges() map[*edge]bool {
	reachable := make(map[*edge]bool)
	visited := make(map[targetKey]bool)
	var visit func(targetKey)
	visit = func(at targetKey) {
		if visited[at] {
			return
		}
		visited[at] = true
		for _, e := range b.result.graph.edges {
			if e.at.doc != at.doc || (at.pointer != "" && !strings.HasPrefix(e.at.pointer, at.pointer+"/")) {
				continue
			}
			reachable[e] = true
			if e.target != nil {
				visit(*e.target)
			}
		}
	}
	visit(targetKey{"entry", ""})
	return reachable
}

func (b *descriptionBundler) targetBase(target targetKey) (string, error) {
	tokens := slices.Collect(jsontext.Pointer(target.pointer).Tokens())
	base := ""
	for _, token := range tokens {
		if token != "" {
			base = token
		}
	}
	if base == "" {
		u, err := parseDescriptionURI(string(b.result.graph.registry.docs[target.doc].canonical))
		if err != nil {
			return "", err
		}
		base = strings.TrimSuffix(path.Base(u.Path), path.Ext(u.Path))
	}
	base = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, base)
	if base == "" {
		base = "Value"
	}
	return base, nil
}

func (b *descriptionBundler) recordOccurrence(at targetKey, destination string, origin SourceLocation) {
	if e := b.edges[at]; e != nil && e.class != "discriminatorMapping" && e.class != "pathItem" {
		if e.class == "securityKey" {
			if failure := securityKeyFailure(b.result.graph.registry.docs["entry"], e); failure != nil {
				failureOrigin := origin
				failureOrigin.Authored = e.authored
				diagnostic := Diagnostic{Code: failure.Code, Path: "#" + destination, Origin: &failureOrigin, Message: failure.Message}
				b.result.keyFailures[diagnostic.Path] = diagnostic
				b.result.diagnostics = append(b.result.diagnostics, diagnostic)
			} else {
				name := e.entry
				if e.target != nil {
					name = jsontext.Pointer(b.targets[*e.target]).LastToken()
				}
				b.result.securityKeys["#"+destination] = resolvedSecurityKey{name: name, edge: e}
			}
		}
		pointer := b.blocked[e]
		if e.target != nil {
			pointer = b.targets[*e.target]
		}
		if e.entry != "" {
			pointer = "/components/" + e.kind + "/" + escapeJSONPointer(e.entry)
		}
		if pointer != "" {
			b.result.references["#"+destination] = "#" + pointer
		}
	}
}

func (b *descriptionBundler) copyPathItem(at targetKey, destination string) (*yaml.Node, error) {
	if e := b.edges[targetKey{at.doc, at.pointer + "/$ref"}]; e != nil && e.target != nil {
		pathOrigin := b.origin(*e.target)
		pathOrigin.Authored = safeAuthoredReference(e.authored)
		b.result.origins["#"+destination] = pathOrigin
	}
	fields, diagnostic, err := effectivePathItem(b.result.graph, at)
	if err != nil {
		return nil, err
	}
	result := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if diagnostic != nil {
		diagnostic.Path = "#" + destination
		b.result.diagnostics = append(b.result.diagnostics, *diagnostic)
		tokens := slices.Collect(jsontext.Pointer(destination).Tokens())
		b.result.unresolvedPaths = append(b.result.unresolvedPaths, tokens[1])
		return result, nil
	}
	for _, field := range fields {
		value, err := b.copyNode(field.origin, destination+"/"+escapeJSONPointer(field.name))
		if err != nil {
			return nil, err
		}
		result.Content = append(result.Content, descriptionString(field.name), value)
	}
	return result, nil
}

func (b *descriptionBundler) entryClaims() rawKeyAllocator {
	claims := make(rawKeyAllocator)
	if components, err := descriptionNode(b.result.graph.registry.docs["entry"].root, "/components"); err == nil && components.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(components.Content); i += 2 {
			kind, values := components.Content[i].Value, components.Content[i+1]
			claims[kind] = make(map[string]bool)
			if values.Kind == yaml.MappingNode {
				for j := 0; j+1 < len(values.Content); j += 2 {
					claims[kind][values.Content[j].Value] = true
				}
			}
		}
	}
	return claims
}
