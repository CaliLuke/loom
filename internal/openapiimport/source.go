package openapiimport

import (
	"errors"
	"strings"
)

type (
	// Source supplies the entry bytes and the explicit local document load policy.
	// A byte-only source has no implicit retrieval URI or filesystem access.
	Source struct {
		// Bytes contains the complete entry OpenAPI document.
		Bytes []byte
		// RetrievalPath is the entry file's absolute path, when known.
		RetrievalPath string
		// ReferenceRoot confines file references; empty uses the entry's real directory.
		ReferenceRoot string
		// Documents maps supplied retrieval URIs to local files or directories.
		Documents []DocumentMapping
	}

	// DocumentMapping supplies one exact URI or a URI prefix ending in a slash.
	DocumentMapping struct {
		// URI is an absolute document URI, or an absolute prefix ending in a slash.
		URI string
		// Path names the local file for an exact URI or directory for a prefix.
		Path string
	}

	// SourceLocation identifies an authored occurrence independently of bundle names.
	// Public diagnostics and provenance redact rejected URI userinfo from its fields.
	SourceLocation struct {
		// Document is a portable source key, without an implicit checkout path.
		Document string
		// Pointer is the source document's JSON Pointer, with a leading hash.
		Pointer string
		// Authored retains the occurrence's original reference spelling.
		Authored string
	}
)

// ParseDocumentMapping parses URI=PATH using the resolver's mapping policy.
// It normalizes URI identity without opening the supplied file or directory.
// Errors report a safe validation reason without echoing the composite input.
func ParseDocumentMapping(value string) (DocumentMapping, error) {
	uri, path, found := strings.Cut(value, "=")
	if !found {
		return DocumentMapping{}, errors.New("document mapping must have the form URI=PATH")
	}
	mappings, err := descriptionMappings([]DocumentMapping{{URI: uri, Path: path}})
	if err != nil {
		return DocumentMapping{}, err
	}
	return DocumentMapping{URI: string(mappings[0].uri), Path: path}, nil
}
