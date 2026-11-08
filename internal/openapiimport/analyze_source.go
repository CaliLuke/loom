package openapiimport

import (
	"errors"
	"fmt"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
)

// ErrUnsupportedVersion indicates that the input is not an OpenAPI 3.0, 3.1, or 3.2
// document.
var ErrUnsupportedVersion = errors.New("unsupported OpenAPI version")

// Analyze parses source without writing to the filesystem and returns a
// normalized model plus every unsupported-feature diagnostic it can discover.
// Syntax, model-building, and version failures are returned as errors.
func Analyze(source []byte) (*Document, Diagnostics, error) {
	document, diagnostics, _, err := AnalyzeSelected(Source{Bytes: source}, Selection{})
	return document, diagnostics, err
}

// AnalyzeSelected parses source and retains operations that match selection.
// It returns tag counts and paths that do not match the requested tags.
func AnalyzeSelected(source Source, selection Selection) (document *Document, diagnostics Diagnostics, report SelectionReport, err error) {
	defer func() {
		var bundle *bundledDescription
		if document != nil {
			bundle = document.bundle
		}
		diagnostics = publicDiagnostics(diagnostics, bundle)
		err = RedactSourceError(err)
	}()
	document, diagnostics, report, err = analyzeSelectedDocument(source, selection)
	if err != nil || document == nil {
		return document, diagnostics, report, err
	}
	diagnostics = append(diagnostics, planDocument(document).diagnostics...)
	return document, uniqueDiagnostics(diagnostics), report, nil
}

func analyzeSelectedDocument(source Source, selection Selection) (*Document, Diagnostics, SelectionReport, error) {
	if err := selection.Validate(); err != nil {
		return nil, nil, SelectionReport{}, err
	}
	parsed, bundle, diagnostics, err := parseResolvedDescription(source)
	if err != nil || parsed == nil {
		return nil, diagnostics, SelectionReport{}, err
	}
	version := parsed.GetVersion()
	numbers := bundle.numbers
	positions := numericSourcePositions(parsed.GetSpecInfo().RootNode, numbers)
	model, err := parsed.BuildV3Model()
	if err != nil {
		return nil, nil, SelectionReport{}, fmt.Errorf("build OpenAPI %s model: %w", version, err)
	}
	analyzer := analyzer{
		bundle:           bundle,
		diagnostics:      bundle.diagnostics,
		selection:        selection,
		examples:         make(map[string]Example),
		schemas:          make(map[string]*Schema),
		version:          version,
		numbers:          numbers,
		numericPositions: positions,
	}
	document := analyzer.document(&model.Model)
	document.bundle = bundle
	for _, path := range bundle.unresolvedPaths {
		if len(selection.Tags) > 0 || selection.matches(path, nil) {
			analyzer.report.UnresolvedPaths = append(analyzer.report.UnresolvedPaths, path)
		}
	}
	document.unresolvedPaths = append([]string(nil), analyzer.report.UnresolvedPaths...)
	analyzer.diagnostics = filterUnresolvedPathDiagnostics(analyzer.diagnostics, document.unresolvedPaths)
	for i := range document.Components.Schemas {
		named := &document.Components.Schemas[i]
		if _, blocked := bundle.blocked["#/components/schemas/"+escapeJSONPointer(named.Name)]; blocked {
			named.GoName = ""
		}
	}
	if selection.Active() {
		closure := pruneComponents(document)
		analyzer.diagnostics = filterSelectionDiagnostics(analyzer.diagnostics, closure)
	}
	analyzer.diagnostics = filterSecurityFailures(document, analyzer.diagnostics)
	return document, analyzer.diagnostics.sorted(), analyzer.report, nil
}

func parseResolvedDescription(source Source) (libopenapi.Document, *bundledDescription, Diagnostics, error) {
	resolved, diagnostics, err := resolveDescription(source)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("inspect OpenAPI references: %w", err)
	}
	if resolved == nil {
		return nil, nil, diagnostics.sorted(), nil
	}
	parsed, err := libopenapi.NewDocument(source.Bytes)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("parse OpenAPI document: %w", err)
	}
	bundle, err := bundleResolution(resolved)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("bundle OpenAPI references: %w", err)
	}
	if bundle.changed() {
		bundledBytes, err := yaml.Marshal(bundle.root)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("encode bundled OpenAPI: %w", err)
		}
		configuration := datamodel.NewDocumentConfiguration()
		configuration.AllowFileReferences = false
		configuration.AllowRemoteReferences = false
		parsed, err = libopenapi.NewDocumentWithConfiguration(bundledBytes, configuration)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("parse bundled OpenAPI: %w", err)
		}
	}
	return parsed, bundle, nil, nil
}
