// Package jsondecode classifies the standard JSON decoder's conversion failures.
package jsondecode

import (
	"encoding/base64"
	"encoding/json/v2"
	"errors"
	"strconv"
)

// IsMismatch reports the structured type and conversion errors that reject one
// candidate. Opaque codec failures retain their cause and are not mismatches.
// Classification follows the error contract, not an application's intent.
func IsMismatch(err error) bool {
	var semantic *json.SemanticError
	if !errors.As(err, &semantic) {
		return false
	}
	var corrupt base64.CorruptInputError
	return semantic.Err == nil || errors.Is(semantic.Err, strconv.ErrRange) || errors.Is(semantic.Err, strconv.ErrSyntax) || errors.As(semantic.Err, &corrupt)
}
