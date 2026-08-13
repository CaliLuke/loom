package openapiimport

import (
	"fmt"
	"sort"
	"strings"

	"github.com/CaliLuke/loom/codegen"
)

func assignSchemaNames(schemas []NamedSchema) {
	sort.Slice(schemas, func(i, j int) bool {
		return schemas[i].Name < schemas[j].Name
	})
	used := make(map[string]int, len(schemas))
	for i := range schemas {
		schemas[i].GoName = uniqueName(codegen.Goify(schemas[i].Name, true), used)
	}
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
		}
		operations[i].GoName = uniqueName(codegen.Goify(base, true), used)
	}
}

func uniqueName(base string, used map[string]int) string {
	if base == "" {
		base = "Value"
	}
	key := strings.ToLower(base)
	used[key]++
	if used[key] == 1 {
		return base
	}
	return fmt.Sprintf("%s%d", base, used[key])
}
