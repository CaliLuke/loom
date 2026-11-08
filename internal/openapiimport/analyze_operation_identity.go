package openapiimport

import (
	"fmt"
)

// validateOperationIdentities checks the selected occurrence set. Reusing a
// Path Item creates independent operations, each of which must have a unique ID.
func (a *analyzer) validateOperationIdentities(operations []Operation) {
	counts := make(map[string]int)
	for _, operation := range operations {
		if operation.OperationID != "" {
			counts[operation.OperationID]++
		}
	}
	for _, operation := range operations {
		if operation.OperationID == "" || counts[operation.OperationID] < 2 {
			continue
		}
		pointer := operation.sourcePath + "/operationId"
		a.unsupported("operation-id", pointer, fmt.Sprintf("operationId %q is used by multiple selected operations", operation.OperationID))
	}
}
