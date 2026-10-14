package loom

import "errors"

// WithErrorRemedy attaches remediation guidance to the given service error and
// returns it.
func WithErrorRemedy(err *ServiceError, remedy *ErrorRemedy) *ServiceError {
	if err == nil || remedy == nil {
		return err
	}
	err.Remedy = remedy
	return err
}

// ExtractErrorRemedy returns the remediation guidance associated with err, if
// any.
func ExtractErrorRemedy(err error) *ErrorRemedy {
	if err == nil {
		return nil
	}
	var remedier LoomErrorRemedier
	if errors.As(err, &remedier) {
		return remedier.LoomErrorRemedy()
	}
	return nil
}

// ErrorRemedyCode returns the stable remediation code associated with err.
func ErrorRemedyCode(err error) string {
	if remedy := ExtractErrorRemedy(err); remedy != nil {
		return remedy.Code
	}
	return ""
}

// ErrorSafeMessage returns the public message selected by ResolveErrorMessage.
// It is the message-only form for transport encoders that do not emit history.
func ErrorSafeMessage(err error) string {
	message, _ := ResolveErrorMessage(err)
	return message
}

// ResolveErrorMessage returns the public message owned by err and whether it
// selected an explicit, nonempty SafeMessage override. The flag is false for a
// named error's own message, the generic fallback, or nil. Encoders use it to
// mask diagnostic history while preserving entries with their own safe override.
//
// An explicit safe remedy takes precedence over a named service or generated
// error's message. Single-cause wrappers are transparent: their diagnostic text
// is not public. Unknown errors and independent joins without an explicit outer
// owner return "internal server error". A nil error returns an empty string.
// The original error and its causes are never modified.
//
//nolint:errorlint // Public ownership stops at joins; tree-wide matching selects a branch.
func ResolveErrorMessage(err error) (message string, safeOverride bool) {
	if err == nil {
		return "", false
	}
	for node := err; node != nil; {
		if remedier, ok := node.(LoomErrorRemedier); ok {
			if remedy := remedier.LoomErrorRemedy(); remedy != nil && remedy.SafeMessage != "" {
				return remedy.SafeMessage, true
			}
		}
		if _, ok := node.(LoomErrorNamer); ok {
			return node.Error(), false
		}
		node = singleErrorCause(node)
	}
	return "internal server error", false
}

// ErrorRetryHint returns retry or correction guidance associated with err.
func ErrorRetryHint(err error) string {
	if remedy := ExtractErrorRemedy(err); remedy != nil {
		return remedy.RetryHint
	}
	return ""
}

// ErrorStatusCode returns the status code associated with err when the error
// exposes one via a generic status interface.
func ErrorStatusCode(err error) (int, bool) {
	if err == nil {
		return 0, false
	}
	var statuser LoomErrorStatuser
	if errors.As(err, &statuser) {
		return statuser.StatusCode(), true
	}
	var reporter LoomErrorStatusReporter
	if errors.As(err, &reporter) {
		return reporter.Status(), true
	}
	return 0, false
}
