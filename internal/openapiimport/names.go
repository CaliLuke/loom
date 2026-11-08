package openapiimport

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/CaliLuke/loom/codegen"
)

// generatedErrorMethodNames mirrors the receiver methods emitted by
// codegen/service.renderErrorMethods.
var generatedErrorMethodNames = map[string]struct{}{
	"Error":           {},
	"LoomErrorName":   {},
	"LoomErrorRemedy": {},
}

func assignOperationNames(operations []Operation) {
	sort.Slice(operations, func(i, j int) bool {
		if operations[i].Path != operations[j].Path {
			return operations[i].Path < operations[j].Path
		}
		return operations[i].Method < operations[j].Method
	})
	used := make(map[string]int, len(operations))
	for i := range operations {
		base := operations[i].OperationID
		if base == "" {
			base = operations[i].Method + " " + operations[i].Path
		} else {
			base = operationIDWithPathWords(base, operations[i].Path)
		}
		name := codegen.Goify(base, true)
		name = strings.ReplaceAll(name, "B2b", "B2B")
		operations[i].GoName = uniqueName(name, used)
	}
}

func operationIDWithPathWords(operationID, path string) string {
	target := alphanumericLower(operationID)
	words := strings.FieldsFunc(path, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	for start := range words {
		var candidate strings.Builder
		for end := start; end < len(words); end++ {
			candidate.WriteString(strings.ToLower(words[end]))
			if candidate.Len() > len(target) {
				break
			}
			if end > start && candidate.String() == target {
				return strings.Join(words[start:end+1], "_")
			}
		}
	}
	return operationID
}

func alphanumericLower(value string) string {
	var normalized strings.Builder
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			normalized.WriteRune(unicode.ToLower(r))
		}
	}
	return normalized.String()
}

func uniqueName(base string, used map[string]int) string {
	if base == "" {
		base = "Value"
	}
	key := strings.ToLower(base)
	if used[key] == 0 {
		used[key] = 1
		return base
	}
	for suffix := used[key] + 1; ; suffix++ {
		candidate := fmt.Sprintf("%s%d", base, suffix)
		candidateKey := strings.ToLower(candidate)
		if used[candidateKey] == 0 {
			used[key] = suffix
			used[candidateKey] = 1
			return candidate
		}
	}
}
