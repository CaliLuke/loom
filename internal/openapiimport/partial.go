package openapiimport

import (
	"fmt"
	"strings"
)

type (
	// PartialAnalysis contains the renderable subset of an OpenAPI document.
	PartialAnalysis struct {
		// Document contains all operations that pass strict analysis.
		Document *Document
		// Blocked contains the grouped refusal source before operations are skipped.
		Blocked Diagnostics
		// Warnings contains allowed lossy diagnostics for retained operations.
		Warnings Diagnostics
		// Skipped contains operations that cannot be rendered and their diagnostics.
		Skipped []SkippedOperation
		// TotalOperations is the number of operations before partial analysis.
		TotalOperations int
		// TotalSchemas is the number of reachable schemas before partial analysis.
		TotalSchemas int
	}

	// SkippedOperation describes one operation omitted from a partial import.
	SkippedOperation struct {
		// Method is the uppercase HTTP method.
		Method string
		// Path is the authored OpenAPI path.
		Path string
		// Diagnostics explains why the operation cannot be rendered.
		Diagnostics Diagnostics
	}
)

// AnalyzePartial returns the largest operation subset that can be rendered.
// Lossy diagnostics are warnings only when allowLossy is true.
func AnalyzePartial(source []byte, selection Selection, allowLossy bool) (*PartialAnalysis, SelectionReport, error) {
	document, rawDiagnostics, report, err := analyzeSelectedDocument(source, selection)
	if err != nil {
		return nil, report, err
	}
	if document == nil {
		return &PartialAnalysis{Blocked: rawDiagnostics.sorted()}, report, nil
	}
	analysis := &PartialAnalysis{
		TotalOperations: len(document.Operations),
		TotalSchemas:    len(document.Components.Schemas),
	}
	retained := make([]Operation, 0, len(document.Operations))
	for _, operation := range document.Operations {
		candidate := documentForOperations(document, []Operation{operation})
		closure := pruneComponents(candidate)
		diagnostics := filterOperationDiagnostics(rawDiagnostics, operation, closure)
		diagnostics = append(diagnostics, planDocument(candidate).diagnostics...)
		fatal, warnings := diagnostics.sorted().Classify(allowLossy)
		if len(fatal) > 0 {
			analysis.Blocked = append(analysis.Blocked, fatal...)
			analysis.Skipped = append(analysis.Skipped, SkippedOperation{
				Method:      operation.Method,
				Path:        operation.Path,
				Diagnostics: fatal,
			})
			continue
		}
		retained = append(retained, operation)
		analysis.Warnings = append(analysis.Warnings, warnings...)
	}
	analysis.Blocked = uniqueDiagnostics(analysis.Blocked)
	analysis.Warnings = uniqueDiagnostics(analysis.Warnings)
	analysis.Document = documentForOperations(document, retained)
	pruneComponents(analysis.Document)
	assignOperationNames(analysis.Document.Operations)
	if diagnostics := planDocument(analysis.Document).diagnostics; len(diagnostics) > 0 {
		return nil, report, fmt.Errorf("plan partial OpenAPI import:\n%s", diagnostics.Error())
	}
	return analysis, report, nil
}

func documentForOperations(document *Document, operations []Operation) *Document {
	result := *document
	result.Components = document.Components
	result.Operations = append([]Operation(nil), operations...)
	result.Tags = append([]string(nil), document.Tags...)
	return &result
}

func filterOperationDiagnostics(diagnostics Diagnostics, operation Operation, closure componentClosure) Diagnostics {
	diagnostics = filterSelectionDiagnostics(diagnostics, closure)
	filtered := make(Diagnostics, 0, len(diagnostics))
	operationBase := "#/paths/" + escapeJSONPointer(operation.Path)
	methodBase := operationBase + "/" + strings.ToLower(operation.Method)
	for _, diagnostic := range diagnostics {
		if !strings.HasPrefix(diagnostic.Path, "#/paths/") ||
			diagnostic.Path == operationBase ||
			strings.HasPrefix(diagnostic.Path, operationBase+"/parameters") ||
			strings.HasPrefix(diagnostic.Path, methodBase) {
			filtered = append(filtered, diagnostic)
		}
	}
	return filtered
}

func uniqueDiagnostics(diagnostics Diagnostics) Diagnostics {
	seen := make(map[string]struct{}, len(diagnostics))
	unique := make(Diagnostics, 0, len(diagnostics))
	for _, diagnostic := range diagnostics.sorted() {
		key := diagnostic.Code + "\x00" + diagnostic.Path + "\x00" + diagnostic.Message
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, diagnostic)
	}
	return unique
}
