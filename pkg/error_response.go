package loom

type (
	// ErrorResponse records the complete failure's public response ownership.
	// Its fields describe the permitted single-cause spine, never an arbitrary
	// branch of an independent join. The original error is not modified.
	ErrorResponse struct {
		// Owner is the first explicitly named error on the single-cause spine.
		// It is nil when no complete named contract exists. Encoders must extract
		// its concrete type directly, without searching its causes again.
		Owner error
		// Name is Owner's Loom error name, or empty when Owner is nil.
		Name string
		// Remedy is the first explicitly supplied remedy on the permitted spine.
		// A nil remedy on an outer remedier shadows remedies of its causes.
		Remedy *ErrorRemedy
	}
)

// ResolveErrorResponse selects the complete error's named response owner and
// remediation metadata. Ordinary wrappers and joins with one nonnil cause are
// transparent. Multiple independent causes have no shared named owner unless an
// explicit outer named error supplies one. Cause lookup ends at that owner.
//
// A remedy-only wrapper can supply metadata without taking over the underlying
// typed body. ErrorSafeMessage separately selects public message overrides.
// Diagnostic helpers such as ExtractErrorRemedy retain their broader contract.
//
//nolint:errorlint // Direct interfaces establish ownership before any cause traversal.
func ResolveErrorResponse(err error) ErrorResponse {
	var response ErrorResponse
	remedySeen := false
	for node := err; node != nil; node = singleErrorCause(node) {
		if remedier, ok := node.(LoomErrorRemedier); ok && !remedySeen {
			response.Remedy = remedier.LoomErrorRemedy()
			remedySeen = true
		}
		if named, ok := node.(LoomErrorNamer); ok {
			response.Owner = node
			response.Name = named.LoomErrorName()
			return response
		}
	}
	return response
}

// singleErrorCause follows transparent wrapping but stops at independent joins.
//
//nolint:errorlint,nilerr // Inspect this node directly; nil means there is no single owned cause.
func singleErrorCause(err error) error {
	switch wrapped := err.(type) {
	case interface{ Unwrap() []error }:
		var child error
		for _, candidate := range wrapped.Unwrap() {
			if candidate == nil {
				continue
			}
			if child != nil {
				return nil
			}
			child = candidate
		}
		return child
	case interface{ Unwrap() error }:
		return wrapped.Unwrap()
	default:
		return nil
	}
}
