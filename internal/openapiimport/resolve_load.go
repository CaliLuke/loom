package openapiimport

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxResolutionDocuments     = 256
	maxResolutionDocumentBytes = 16 * 1024 * 1024
	maxResolutionBytes         = 64 * 1024 * 1024
	maxResolutionDepth         = 64
	maxPathItemObjects         = 64
)

var (
	errResolutionBudget    = errors.New("resolution budget exceeded")
	errDocumentNotSupplied = errors.New("target document is not supplied; pass --document")
)

type (
	documentLoadMapping struct {
		uri    docURI
		path   string
		prefix bool
		root   *os.Root
	}
	descriptionLoader struct {
		mappings      []documentLoadMapping
		referenceRoot *os.Root
		referencePath string
		entryReal     string
	}
)

// loadDocument is the one-document seam; graph resolution retains one loader
// and its opened roots for the entire graph.
func loadDocument(source Source, uri docURI) (data []byte, real string, err error) {
	loader, err := openDescriptionLoader(source)
	if err != nil {
		return nil, "", err
	}
	defer func() {
		err = errors.Join(err, loader.close())
	}()
	return loader.load(uri)
}

func openDescriptionLoader(source Source) (loader *descriptionLoader, err error) {
	mappings, err := descriptionMappings(source.Documents)
	if err != nil {
		return nil, err
	}
	loader = &descriptionLoader{mappings: mappings}
	defer func() {
		if err != nil {
			err = errors.Join(err, loader.close())
			loader = nil
		}
	}()
	for i := range loader.mappings {
		mapping := &loader.mappings[i]
		if !mapping.prefix {
			continue
		}
		mapping.path, err = filepath.EvalSymlinks(mapping.path)
		if err != nil {
			return loader, err
		}
		mapping.path, err = filepath.Abs(mapping.path)
		if err != nil {
			return loader, err
		}
		mapping.root, err = os.OpenRoot(mapping.path)
		if err != nil {
			return loader, err
		}
	}
	if source.RetrievalPath != "" {
		loader.referencePath, err = descriptionReferenceRoot(source)
		if err != nil {
			return loader, err
		}
		loader.entryReal, err = filepath.EvalSymlinks(source.RetrievalPath)
		if err != nil {
			return loader, err
		}
		loader.entryReal, err = filepath.Abs(loader.entryReal)
		if err != nil {
			return loader, err
		}
		loader.referenceRoot, err = os.OpenRoot(loader.referencePath)
		if err != nil {
			return loader, err
		}
	} else if source.ReferenceRoot != "" {
		return loader, errors.New("reference root requires a retrieved entry")
	}
	return loader, nil
}

func (l *descriptionLoader) close() error {
	var result error
	for _, mapping := range l.mappings {
		if mapping.root != nil {
			result = errors.Join(result, mapping.root.Close())
		}
	}
	if l.referenceRoot != nil {
		result = errors.Join(result, l.referenceRoot.Close())
	}
	return result
}

func (l *descriptionLoader) load(uri docURI) ([]byte, string, error) {
	identity, err := normalizeURI(string(uri))
	if err != nil {
		return nil, "", err
	}
	u, err := parseDescriptionURI(string(identity))
	if err != nil {
		return nil, "", err
	}
	if u.RawQuery != "" || u.ForceQuery {
		return nil, "", errors.New("document loading does not permit queries")
	}
	for _, mapping := range l.mappings {
		if !mapping.prefix && mapping.uri == identity {
			return readDescriptionFile(mapping.path)
		}
	}
	var selected *documentLoadMapping
	for i := range l.mappings {
		mapping := &l.mappings[i]
		if mapping.prefix && strings.HasPrefix(string(identity), string(mapping.uri)) && (selected == nil || len(mapping.uri) > len(selected.uri)) {
			selected = mapping
		}
	}
	if selected != nil {
		rel, err := uriRelativeFile(strings.TrimPrefix(string(identity), string(selected.uri)))
		if err != nil {
			return nil, "", err
		}
		return readConfinedDescription(selected.root, selected.path, rel)
	}
	if u.Scheme == "file" && l.referenceRoot != nil {
		if u.Host != "" && u.Host != "localhost" {
			return nil, "", errors.New("file URI host must be empty or localhost")
		}
		decoded, err := uriRelativeFile(strings.TrimPrefix(u.EscapedPath(), "/"))
		if err != nil {
			return nil, "", err
		}
		path, err := filepath.EvalSymlinks(filepath.FromSlash("/" + decoded))
		if err != nil {
			return nil, "", fmt.Errorf("resolve referenced file: %w", err)
		}
		rel, err := filepath.Rel(l.referencePath, path)
		if err != nil || !filepath.IsLocal(rel) {
			return nil, "", errors.New("file reference escapes the reference root")
		}
		return readConfinedDescription(l.referenceRoot, l.referencePath, rel)
	}
	return nil, "", errDocumentNotSupplied
}

func descriptionMappings(input []DocumentMapping) ([]documentLoadMapping, error) {
	seen := make(map[docURI]bool)
	mappings := make([]documentLoadMapping, 0, len(input))
	for _, mapping := range input {
		u, err := parseDescriptionURI(mapping.URI)
		if err != nil {
			return nil, err
		}
		if !u.IsAbs() || strings.Contains(mapping.URI, "#") || u.RawQuery != "" || u.ForceQuery || mapping.Path == "" {
			return nil, errors.New("document mapping requires an absolute URI without query or fragment and a local path")
		}
		identity, err := normalizeURI(mapping.URI)
		if err != nil {
			return nil, err
		}
		if seen[identity] {
			return nil, errors.New("duplicate document mapping URI")
		}
		seen[identity] = true
		mappings = append(mappings, documentLoadMapping{uri: identity, path: mapping.Path, prefix: strings.HasSuffix(mapping.URI, "/")})
	}
	return mappings, nil
}

func descriptionReferenceRoot(source Source) (string, error) {
	entry, err := filepath.EvalSymlinks(source.RetrievalPath)
	if err != nil {
		return "", fmt.Errorf("resolve entry path: %w", err)
	}
	entry, err = filepath.Abs(entry)
	if err != nil {
		return "", err
	}
	root := source.ReferenceRoot
	if root == "" {
		return filepath.Dir(entry), nil
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolve reference root: %w", err)
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, entry)
	if err != nil || !filepath.IsLocal(rel) {
		return "", errors.New("reference root must contain the entry's real path")
	}
	return root, nil
}

// uriRelativeFile decodes each segment exactly once, after identity normalization.
func uriRelativeFile(value string) (string, error) {
	parts := strings.Split(value, "/")
	for i, part := range parts {
		decoded, err := url.PathUnescape(part)
		if err != nil || decoded == "" || decoded == "." || decoded == ".." || strings.ContainsAny(decoded, "/\\\x00:") {
			return "", errors.New("invalid document path segment")
		}
		parts[i] = decoded
	}
	return filepath.Join(parts...), nil
}

func readDescriptionFile(path string) (data []byte, real string, err error) {
	real, err = filepath.EvalSymlinks(path)
	if err != nil {
		return nil, "", fmt.Errorf("resolve supplied document: %w", err)
	}
	real, err = filepath.Abs(real)
	if err != nil {
		return nil, "", err
	}
	info, err := os.Stat(real)
	if err != nil {
		return nil, "", err
	}
	if !info.Mode().IsRegular() {
		return nil, "", errors.New("supplied document must be a regular file")
	}
	file, err := os.OpenFile(real, descriptionOpenFlags, 0)
	if err != nil {
		return nil, "", fmt.Errorf("open supplied document: %w", err)
	}
	defer func() {
		err = errors.Join(err, file.Close())
	}()
	data, err = readDescriptionBytes(file)
	return data, real, err
}

func readConfinedDescription(root *os.Root, directory, relative string) (data []byte, real string, err error) {
	info, err := root.Stat(relative)
	if err != nil {
		return nil, "", err
	}
	if !info.Mode().IsRegular() {
		return nil, "", errors.New("supplied document must be a regular file")
	}
	file, err := root.OpenFile(relative, descriptionOpenFlags, 0)
	if err != nil {
		return nil, "", fmt.Errorf("open confined document: %w", err)
	}
	defer func() {
		err = errors.Join(err, file.Close())
	}()
	real, err = filepath.EvalSymlinks(filepath.Join(directory, relative))
	if err != nil {
		return nil, "", err
	}
	rel, relErr := filepath.Rel(directory, real)
	if relErr != nil || !filepath.IsLocal(rel) {
		return nil, "", errors.New("document symlink escapes its root")
	}
	opened, err := file.Stat()
	if err != nil {
		return nil, "", err
	}
	current, err := os.Stat(real)
	if err != nil {
		return nil, "", err
	}
	if !os.SameFile(opened, current) {
		return nil, "", errors.New("document identity changed during loading")
	}
	data, err = readDescriptionBytes(file)
	return data, real, err
}

func readDescriptionBytes(file *os.File) ([]byte, error) {
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("supplied document must be a regular file")
	}
	if info.Size() > maxResolutionDocumentBytes {
		return nil, errResolutionBudget
	}
	data, err := io.ReadAll(io.LimitReader(file, maxResolutionDocumentBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxResolutionDocumentBytes {
		return nil, errResolutionBudget
	}
	return data, nil
}
