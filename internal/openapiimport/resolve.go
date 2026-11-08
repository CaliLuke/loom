package openapiimport

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pb33f/go-yaml"
)

type (
	sourceID string
	docURI   string
	document struct {
		key                               sourceID
		base, canonical, retrieval        docURI
		realPath, self, version, portable string
		dialect                           string
		hasSelf                           bool
		root                              *yaml.Node
		numbers                           map[string]numericSource
		schemaPositions                   map[string]struct{}
	}
	targetKey struct {
		doc     sourceID
		pointer string
	}
	edge struct {
		class, kind     string
		at              targetKey
		authored        string
		target          *targetKey
		entry, external string
		failure         *Diagnostic
		invalid         bool
		bundled         string
	}
	registry struct {
		alias     map[docURI]sourceID
		realPaths map[string]sourceID
		docs      map[sourceID]*document
	}
	resolution struct {
		registry *registry
		edges    []*edge
	}
	descriptionResolver struct {
		loader    *descriptionLoader
		result    *resolution
		seenEdges map[string]bool
		scanned   map[string]bool
		loaded    map[docURI]error
		contexts  map[targetKey]map[string]bool
		bytes     int
		root      string
	}
)

var (
	errDocumentIdentity = errors.New("document identity conflict")
	descriptionName     = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
)

func resolveDescription(source Source) (result *resolution, diagnostics Diagnostics, err error) {
	entry, err := newDescriptionDocument(source.Bytes, "", "", true)
	if err != nil {
		return descriptionFatal(err)
	}
	if !renderableVersion(entry.version) {
		return nil, nil, fmt.Errorf("%w %q; expected 3.0.x, 3.1.x, or 3.2.x", ErrUnsupportedVersion, entry.version)
	}
	loader, err := openDescriptionLoader(source)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		err = errors.Join(err, loader.close())
	}()
	return resolveLoadedDescription(entry, loader)
}

func resolveLoadedDescription(entry *document, loader *descriptionLoader) (*resolution, Diagnostics, error) {
	mappings := loader.mappings
	r := &descriptionResolver{
		loader:    loader,
		result:    &resolution{registry: &registry{alias: make(map[docURI]sourceID), realPaths: make(map[string]sourceID), docs: make(map[sourceID]*document)}},
		seenEdges: make(map[string]bool), scanned: make(map[string]bool),
		loaded: make(map[docURI]error), contexts: make(map[targetKey]map[string]bool),
	}
	r.root = loader.referencePath
	real := loader.entryReal
	var retrieval docURI
	if real != "" {
		retrieval = docURI((&url.URL{Scheme: "file", Path: filepath.ToSlash(real)}).String())
	}
	if err := entry.setRetrieval(retrieval, real, true); err != nil {
		return descriptionFatal(err)
	}
	if err := r.registerDocument(entry, 0, true); err != nil {
		return descriptionFatal(err)
	}
	for _, mapping := range mappings {
		if !mapping.prefix {
			if err := r.load(mapping.uri); errors.Is(err, errDocumentIdentity) || errors.Is(err, errResolutionBudget) {
				return descriptionFatal(err)
			}
		}
	}
	for {
		beforeDocs, beforeEdges, beforeAliases := len(r.result.registry.docs), len(r.result.edges), len(r.result.registry.alias)
		for i := 0; i < len(r.result.edges); i++ {
			e := r.result.edges[i]
			if e.target != nil || e.entry != "" {
				continue
			}
			if err := r.resolveEdge(e); err != nil {
				return descriptionFatal(err)
			}
		}
		if beforeDocs == len(r.result.registry.docs) && beforeEdges == len(r.result.edges) && beforeAliases == len(r.result.registry.alias) {
			break
		}
	}
	if err := r.checkReferenceDepth(); err != nil {
		return descriptionFatal(err)
	}
	var diagnostics Diagnostics
	for _, e := range r.result.edges {
		if e.target != nil && len(r.contexts[*e.target]) > 1 {
			e.failure = r.failure(e, "reference-context", "target is referenced as different OpenAPI object kinds")
			e.target = nil
		}
		if e.failure != nil && e.class != "discriminatorMapping" {
			diagnostics = append(diagnostics, *e.failure)
		}
	}
	return r.result, diagnostics.sorted(), nil
}

func descriptionFatal(err error) (*resolution, Diagnostics, error) {
	code := ""
	if errors.Is(err, errDocumentIdentity) {
		code = "document-identity"
	} else if errors.Is(err, errResolutionBudget) {
		code = "external-reference"
	}
	if code == "" {
		return nil, nil, err
	}
	return nil, Diagnostics{{Code: code, Path: "#", Message: err.Error()}}, nil
}

func newDescriptionDocument(data []byte, retrieval docURI, real string, entry bool) (*document, error) {
	root, numbers, err := parseDescriptionTree(data)
	if err != nil {
		return nil, err
	}
	doc := &document{key: sourceID(real), retrieval: retrieval, realPath: real, root: root, numbers: numbers}
	if entry {
		doc.key = "entry"
	}
	if root.Kind == yaml.MappingNode {
		fields, err := effectiveYAMLMapping(root, make(map[*yaml.Node]bool))
		if err != nil {
			return nil, err
		}
		if version := fields["openapi"]; version != nil {
			doc.version = version.Value
		} else if version := fields["swagger"]; entry && version != nil {
			doc.version = version.Value
		}
		if dialect := fields["jsonSchemaDialect"]; dialect != nil && doc.version != "" {
			doc.dialect = dialect.Value
		}
		if self := fields["$self"]; self != nil && strings.HasPrefix(doc.version, "3.2.") {
			if self.Kind != yaml.ScalarNode || self.Tag != "!!str" {
				return nil, fmt.Errorf("%w: $self must be a URI reference string", errDocumentIdentity)
			}
			doc.self, doc.hasSelf = self.Value, true
		}
	}
	if err := doc.setRetrieval(retrieval, real, entry); err != nil {
		return nil, err
	}
	return doc, nil
}

func (doc *document) setRetrieval(retrieval docURI, real string, entry bool) error {
	doc.key, doc.retrieval, doc.realPath = sourceID(real), retrieval, real
	if entry {
		doc.key = "entry"
	}
	doc.base = retrieval
	if doc.hasSelf {
		u, err := parseDescriptionURI(doc.self)
		if err != nil {
			return fmt.Errorf("%w: invalid $self URI", errDocumentIdentity)
		}
		if !u.IsAbs() && retrieval != "" {
			base, err := parseDescriptionURI(string(retrieval))
			if err != nil {
				return err
			}
			u = base.ResolveReference(u)
		}
		if u.IsAbs() {
			doc.base, err = normalizeURI(u.String())
			if err != nil {
				return fmt.Errorf("%w: invalid $self URI", errDocumentIdentity)
			}
		}
	}
	doc.canonical = doc.base
	return nil
}

func (r *descriptionResolver) registerDocument(doc *document, size int, entry bool) error {
	real, retrieval := doc.realPath, doc.retrieval
	if priorKey, exists := r.result.registry.realPaths[real]; real != "" && exists {
		prior := r.result.registry.docs[priorKey]
		if prior.retrieval != retrieval && (!prior.hasSelf || !doc.hasSelf || prior.canonical != doc.canonical) {
			return fmt.Errorf("%w: one source has incompatible retrieval bases", errDocumentIdentity)
		}
		doc.key = priorKey
	}
	for _, identity := range []docURI{retrieval, doc.base} {
		if identity == "" {
			continue
		}
		if prior, exists := r.result.registry.alias[identity]; exists && prior != doc.key {
			return fmt.Errorf("%w: distinct sources claim one URI", errDocumentIdentity)
		}
	}
	for _, identity := range []docURI{retrieval, doc.base} {
		if identity != "" {
			r.result.registry.alias[identity] = doc.key
		}
	}
	if prior := r.result.registry.docs[doc.key]; prior != nil {
		return nil
	}
	if !entry {
		if len(r.result.registry.docs)-1 >= maxResolutionDocuments || r.bytes+size > maxResolutionBytes {
			return errResolutionBudget
		}
		r.bytes += size
	}
	doc.portable = r.portable(doc)
	r.result.registry.docs[doc.key] = doc
	if real != "" {
		r.result.registry.realPaths[real] = doc.key
	}
	if entry || doc.version != "" {
		if err := r.scan(doc, "", "openapi"); err != nil {
			return err
		}
	}
	return nil
}

func (r *descriptionResolver) load(uri docURI) error {
	if prior, seen := r.loaded[uri]; seen {
		return prior
	}
	data, real, err := r.loader.load(uri)
	if err == nil {
		var doc *document
		doc, err = newDescriptionDocument(data, uri, real, false)
		if err == nil {
			err = r.registerDocument(doc, len(data), false)
		}
	}
	r.loaded[uri] = err
	return err
}

func (r *descriptionResolver) scan(doc *document, pointer, kind string) error {
	key := string(doc.key) + "\x00" + pointer + "\x00" + kind
	if r.scanned[key] {
		return nil
	}
	r.scanned[key] = true
	edges, err := descriptionEdges(doc, pointer, kind)
	if err != nil {
		return err
	}
	for _, e := range edges {
		key := string(e.at.doc) + "\x00" + e.at.pointer + "\x00" + e.class
		if !r.seenEdges[key] {
			r.seenEdges[key] = true
			r.result.edges = append(r.result.edges, e)
		}
	}
	return nil
}

func (r *descriptionResolver) resolveEdge(e *edge) error {
	if e.invalid {
		e.failure = r.failure(e, "external-reference", "reference must be a string")
		return nil
	}
	doc := r.result.registry.docs[e.at.doc]
	if r.resolveEntryComponent(e) {
		return nil
	}
	uri, pointer, err := resolveReference(e.authored, doc.base)
	if err != nil {
		e.failure = r.failure(e, "external-reference", err.Error())
		return nil
	}
	targetDoc, err := r.resolveEdgeDocument(e, doc, uri, pointer)
	if err != nil || targetDoc == nil {
		return err
	}
	target := targetKey{targetDoc.key, pointer}
	if err := r.admitTarget(targetDoc, target, e.kind); err != nil {
		code := "external-reference"
		if errors.Is(err, errSchemaResource) {
			code = "schema-resource"
		} else if errors.Is(err, errSchemaDialect) {
			code = "json-schema-dialect"
		}
		e.failure = r.failure(e, code, err.Error())
		return nil
	}
	if r.contexts[target] == nil {
		r.contexts[target] = make(map[string]bool)
	}
	r.contexts[target][e.kind] = true
	e.target, e.failure, e.external = &target, nil, ""
	return r.scan(targetDoc, pointer, e.kind)
}

// resolveEntryComponent applies entry-name precedence before URI resolution.
// Its result reports whether the occurrence is complete, including a bad name.
func (r *descriptionResolver) resolveEntryComponent(e *edge) bool {
	if e.class != "securityKey" && e.class != "discriminatorMapping" {
		return false
	}
	pointer := "/components/" + e.kind + "/" + escapeJSONPointer(e.authored)
	if _, err := descriptionNode(r.result.registry.docs["entry"].root, pointer); err == nil {
		e.entry, e.failure = e.authored, nil
		return true
	}
	if descriptionName.MatchString(e.authored) {
		e.failure = r.failure(e, "external-reference", "name does not identify an entry component; use a URI reference")
		return true
	}
	return false
}

// resolveEdgeDocument loads through explicit policy before consulting aliases.
// A nil document with no error is a completed blocked or external occurrence;
// identity and budget failures instead abort resolution.
func (r *descriptionResolver) resolveEdgeDocument(e *edge, doc *document, uri docURI, pointer string) (*document, error) {
	if uri == "" {
		return doc, nil
	}
	loadErr := r.load(uri)
	if errors.Is(loadErr, errDocumentIdentity) || errors.Is(loadErr, errResolutionBudget) {
		return nil, loadErr
	}
	if key, exists := r.result.registry.alias[uri]; exists {
		return r.result.registry.docs[key], nil
	}
	if e.class == "discriminatorMapping" && !strings.HasPrefix(string(uri), "file:") && errors.Is(loadErr, errDocumentNotSupplied) {
		e.external, e.failure = string(uri)+"#"+emitPointer(pointer), nil
		return nil, nil
	}
	if loadErr == nil {
		loadErr = errDocumentNotSupplied
	}
	e.failure = r.failure(e, "external-reference", loadErr.Error())
	return nil, nil
}

func (r *descriptionResolver) failure(e *edge, code, message string) *Diagnostic {
	doc := r.result.registry.docs[e.at.doc]
	d := &Diagnostic{Code: code, Path: "#" + e.at.pointer, Message: message}
	if doc.key != "entry" {
		d.Origin = &SourceLocation{Document: doc.portable, Pointer: "#" + e.at.pointer, Authored: e.authored}
	}
	return d
}

func (r *descriptionResolver) portable(doc *document) string {
	if doc.key == "entry" {
		return "entry"
	}
	uri, err := parseDescriptionURI(string(doc.canonical))
	if err == nil && uri.Scheme == "file" && r.root != "" {
		if relative, err := filepath.Rel(r.root, uri.Path); err == nil && filepath.IsLocal(relative) {
			return "root:" + filepath.ToSlash(relative)
		}
	}
	return string(doc.canonical)
}
