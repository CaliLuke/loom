package goa

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
	var remedier GoaErrorRemedier
	if errors.As(err, &remedier) {
		return remedier.GoaErrorRemedy()
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

// ErrorSafeMessage returns the safe, user-facing message associated with err.
// It falls back to err.Error when no safe message was declared.
func ErrorSafeMessage(err error) string {
	if remedy := ExtractErrorRemedy(err); remedy != nil && remedy.SafeMessage != "" {
		return remedy.SafeMessage
	}
	if err == nil {
		return ""
	}
	return err.Error()
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
	var statuser GoaErrorStatuser
	if errors.As(err, &statuser) {
		return statuser.StatusCode(), true
	}
	var reporter GoaErrorStatusReporter
	if errors.As(err, &reporter) {
		return reporter.Status(), true
	}
	return 0, false
}
