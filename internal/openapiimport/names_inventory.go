package openapiimport

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/internal/openapiref"
)

type (
	designNameAllocator struct {
		document         *Document
		types, variables map[string]int
		scope            *codegen.NameScope
	}
	declarationOrigin struct {
		// introduced distinguishes generated declarations from authored entry components.
		introduced bool
		order      int
	}
	cloneIdentity struct {
		method, path, status string
	}
	cloneNames struct {
		typeName, variable string
	}
	designNames struct {
		clones        map[cloneIdentity]cloneNames
		informational map[cloneIdentity]cloneNames
		handshakes    map[cloneIdentity]cloneNames
	}
	errorCloneCandidate struct {
		identity cloneIdentity
		name     string
		origin   declarationOrigin
	}
)

func errorCloneInventory(document *Document) ([]errorCloneCandidate, error) {
	resolved, diagnostics := resolveReusableDocument(document)
	if len(diagnostics) > 0 {
		return nil, diagnostics
	}
	r := renderer{document: resolved, schemas: make(map[string]NamedSchema), errorSchemas: make(map[string]struct{})}
	for _, named := range resolved.Components.Schemas {
		r.schemas[named.Name] = named
	}
	for i := range resolved.Operations {
		operation := &resolved.Operations[i]
		responses, err := r.operationResponses(operation, fmt.Sprintf("#/operations/%d", i))
		if err != nil {
			return nil, err
		}
		r.operations = append(r.operations, operationPlan{renderedResponses: responses})
	}
	disambiguateSharedErrorStatuses(resolved.Operations, r.operations)
	if err := r.classifyErrorClones(); err != nil {
		return nil, err
	}
	introduced := make(map[string]bool)
	for i, plan := range r.operations {
		for _, response := range plan.failures {
			if len(response.headers) > 0 || response.statusField != "" || response.response.Schema == nil || response.response.Schema.Ref == "" {
				continue
			}
			name, err := openapiref.Name(response.response.Schema.Ref, "#/components/schemas/")
			if err != nil {
				return nil, err
			}
			introduced[name] = introduced[name] || r.schemas[name].origin.introduced || errorResponseIntroduced(resolved, resolved.Operations[i], response)
		}
	}
	var inventory []errorCloneCandidate
	for i, plan := range r.operations {
		operation := resolved.Operations[i]
		for _, response := range plan.failures {
			if !response.cloneErrorType {
				continue
			}
			name, err := openapiref.Name(response.response.Schema.Ref, "#/components/schemas/")
			if err != nil {
				return nil, err
			}
			inventory = append(inventory, errorCloneCandidate{
				identity: cloneIdentity{operation.Method, operation.Path, response.status},
				name:     response.errorName, origin: declarationOrigin{introduced: introduced[name]},
			})
		}
	}
	return inventory, nil
}

func errorResponseIntroduced(document *Document, operation Operation, response renderedResponse) bool {
	if operation.origin.introduced {
		return true
	}
	base := operation.sourcePath + "/responses/" + escapeJSONPointer(response.status)
	if document.bundle.declarationOrigin(base).introduced {
		return true
	}
	for _, reference := range response.response.referenceChain {
		if document.bundle.declarationOrigin(reference.Ref).introduced {
			return true
		}
	}
	return false
}

func allocateDesignNames(document *Document) error {
	sort.Slice(document.Components.Schemas, func(i, j int) bool { return document.Components.Schemas[i].Name < document.Components.Schemas[j].Name })
	sort.Slice(document.Components.SecuritySchemes, func(i, j int) bool {
		return document.Components.SecuritySchemes[i].Name < document.Components.SecuritySchemes[j].Name
	})
	a := designNameAllocator{document: document, types: make(map[string]int), variables: make(map[string]int), scope: codegen.NewNameScope()}
	a.reserveEntry()
	candidates, err := errorCloneInventory(document)
	document.names = designNames{clones: make(map[cloneIdentity]cloneNames), informational: make(map[cloneIdentity]cloneNames), handshakes: make(map[cloneIdentity]cloneNames)}
	a.allocateClones(candidates, false)
	a.allocateIntroducedSchemas()
	a.allocateIntroducedSchemes()
	a.allocateClones(candidates, true)
	a.allocateInformational()
	resolved, diagnostics := resolveReusableDocument(document)
	if len(diagnostics) > 0 {
		return diagnostics
	}
	a.allocateHandshakes(resolved.Operations)
	return err
}

func designNameDiagnostics(err error) Diagnostics {
	var diagnostics Diagnostics
	if errors.As(err, &diagnostics) {
		return diagnostics
	}
	return Diagnostics{{Code: "render-plan", Path: "#", Message: err.Error()}}
}

func (a *designNameAllocator) blocked(name string) bool {
	if a.document.bundle == nil {
		return false
	}
	_, ok := a.document.bundle.blocked["#/components/schemas/"+escapeJSONPointer(name)]
	return ok
}

func (a *designNameAllocator) reserveEntry() {
	for i := range a.document.Components.Schemas {
		named := &a.document.Components.Schemas[i]
		if a.blocked(named.Name) {
			named.GoName = ""
			continue
		}
		if !named.origin.introduced {
			base := codegen.Goify(named.Name, true)
			if a.document.bundle == nil && named.GoName != "" {
				base = named.GoName
			}
			named.GoName = uniqueName(base, a.types)
			a.scope.Unique(named.GoName)
			uniqueName("Imported"+named.GoName, a.variables)
		}
	}
	for i := range a.document.Components.SecuritySchemes {
		scheme := &a.document.Components.SecuritySchemes[i]
		if !scheme.origin.introduced {
			base := "Imported" + codegen.Goify(scheme.Name, true) + "Security"
			if a.document.bundle == nil && scheme.GoName != "" {
				base = scheme.GoName
			}
			scheme.GoName = uniqueName(base, a.variables)
		}
	}
}

func (a *designNameAllocator) allocateClones(candidates []errorCloneCandidate, introduced bool) {
	for _, candidate := range candidates {
		if candidate.origin.introduced != introduced {
			continue
		}
		name := a.scope.Unique(codegen.Goify(candidate.name, true))
		for a.types[strings.ToLower(name)] > 0 {
			name = a.scope.Unique(codegen.Goify(candidate.name, true))
		}
		a.types[strings.ToLower(name)] = 1
		variable := uniqueName("Imported"+name+"Error", a.variables)
		a.document.names.clones[candidate.identity] = cloneNames{name, variable}
	}
}

func (a *designNameAllocator) allocateIntroducedSchemas() {
	var schemas []int
	for i, named := range a.document.Components.Schemas {
		if named.origin.introduced && !a.blocked(named.Name) {
			schemas = append(schemas, i)
		}
	}
	sort.SliceStable(schemas, func(i, j int) bool {
		return a.document.Components.Schemas[schemas[i]].origin.order < a.document.Components.Schemas[schemas[j]].origin.order
	})
	for _, i := range schemas {
		named := &a.document.Components.Schemas[i]
		base := codegen.Goify(named.Name, true)
		if base == "" {
			base = "Value"
		}
		name := base
		for suffix := 2; a.types[strings.ToLower(name)] > 0 || a.variables[strings.ToLower("Imported"+name)] > 0; suffix++ {
			name = fmt.Sprintf("%s%d", base, suffix)
		}
		named.GoName = name
		a.types[strings.ToLower(name)] = 1
		a.variables[strings.ToLower("Imported"+name)] = 1
		a.scope.Unique(name)
	}
}

func (a *designNameAllocator) allocateIntroducedSchemes() {
	var schemes []int
	for i, scheme := range a.document.Components.SecuritySchemes {
		if scheme.origin.introduced {
			schemes = append(schemes, i)
		}
	}
	sort.SliceStable(schemes, func(i, j int) bool {
		return a.document.Components.SecuritySchemes[schemes[i]].origin.order < a.document.Components.SecuritySchemes[schemes[j]].origin.order
	})
	for _, i := range schemes {
		scheme := &a.document.Components.SecuritySchemes[i]
		scheme.GoName = uniqueName("Imported"+codegen.Goify(scheme.Name, true)+"Security", a.variables)
	}
}

func (a *designNameAllocator) allocateInformational() {
	for _, operation := range a.document.Operations {
		plan, _ := classifyOperationResponses(&operation, operation.sourcePath, a.document.Components)
		for _, response := range plan.informational {
			base := operation.GoName + "Status" + selectorName(response.selector)
			name := a.scope.Unique(base)
			for a.types[strings.ToLower(name)] > 0 {
				name = a.scope.Unique(base)
			}
			a.types[strings.ToLower(name)] = 1
			variable := uniqueName("Imported"+name, a.variables)
			a.document.names.informational[cloneIdentity{operation.Method, operation.Path, response.source.Status}] = cloneNames{name, variable}
		}
	}
}
