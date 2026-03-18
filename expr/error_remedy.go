package expr

import "goa.design/goa/v3/eval"

type (
	// ErrorRemedyExpr describes machine-consumable remediation guidance for an
	// API error.
	ErrorRemedyExpr struct {
		// Code is the stable remediation code consumers may use to classify the
		// failure.
		Code string
		// SafeMessage is the safe, user-facing message to surface without leaking
		// internal details.
		SafeMessage string
		// RetryHint is concise guidance on how to correct the request or retry the
		// operation.
		RetryHint string
	}
)

// Validate checks that a remedy, when declared, contains at least one field.
func (r *ErrorRemedyExpr) Validate() *eval.ValidationErrors {
	if r == nil {
		return nil
	}
	if r.Code == "" && r.SafeMessage == "" && r.RetryHint == "" {
		verr := new(eval.ValidationErrors)
		verr.Add(r, "error remedy must define at least one of code, safe message, or retry hint")
		return verr
	}
	return nil
}

// EvalName returns the generic expression name used in validation errors.
func (*ErrorRemedyExpr) EvalName() string {
	return "error remedy"
}
