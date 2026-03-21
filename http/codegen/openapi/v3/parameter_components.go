package openapiv3

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"goa.design/goa/v3/codegen"
)

const parameterComponentRefPrefix = "#/components/parameters/"

func componentizeParameters(paths map[string]*PathItem) map[string]*ParameterRef {
	if len(paths) == 0 {
		return nil
	}

	counts := make(map[string]int)
	inlineRefs := collectInlineParameters(paths)
	for _, ref := range inlineRefs {
		hash, err := parameterHash(ref.Value)
		if err != nil {
			continue
		}
		counts[hash]++
	}

	if len(counts) == 0 {
		return nil
	}

	components := make(map[string]*ParameterRef)
	namesByHash := make(map[string]string)
	hashesByName := make(map[string]string)
	for _, ref := range inlineRefs {
		if ref == nil || ref.Value == nil || ref.Ref != "" {
			continue
		}
		hash, err := parameterHash(ref.Value)
		if err != nil || counts[hash] < 2 {
			continue
		}

		name, ok := namesByHash[hash]
		if !ok {
			name = uniqueParameterComponentName(ref.Value, hash, hashesByName)
			namesByHash[hash] = name
			hashesByName[name] = hash
			components[name] = &ParameterRef{Value: ref.Value}
		}

		ref.Ref = parameterComponentRefPrefix + name
		ref.Value = nil
	}

	if len(components) == 0 {
		return nil
	}
	return components
}

func collectInlineParameters(paths map[string]*PathItem) []*ParameterRef {
	refs := make([]*ParameterRef, 0)
	for _, path := range orderedPathKeys(paths) {
		pathItem := paths[path]
		if pathItem == nil {
			continue
		}
		refs = appendInlineParameterRefs(refs, pathItem.Parameters)
		for _, op := range orderedOperations(pathItem) {
			refs = appendInlineParameterRefs(refs, op.operation.Parameters)
		}
	}
	return refs
}

func appendInlineParameterRefs(dst []*ParameterRef, refs []*ParameterRef) []*ParameterRef {
	for _, ref := range refs {
		if ref == nil || ref.Value == nil || ref.Ref != "" {
			continue
		}
		dst = append(dst, ref)
	}
	return dst
}

func parameterHash(param *Parameter) (string, error) {
	if param == nil {
		return "", nil
	}
	data, err := json.Marshal(param)
	if err != nil {
		return "", err
	}
	sum := sha1.Sum(data)
	return hex.EncodeToString(sum[:]), nil
}

func uniqueParameterComponentName(param *Parameter, hash string, hashesByName map[string]string) string {
	base := parameterComponentName(param)
	if existingHash, ok := hashesByName[base]; !ok || existingHash == hash {
		return base
	}
	return fmt.Sprintf("%s_%s", base, hash[:8])
}

func parameterComponentName(param *Parameter) string {
	if param == nil {
		return "Parameter"
	}
	base := codegen.Goify(param.In, true) + codegen.Goify(parameterComponentSuffix(param.Name), true)
	if base == "" {
		return "Parameter"
	}
	return base
}

func parameterComponentSuffix(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "Parameter"
	}
	return trimmed
}
