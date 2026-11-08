package http

import (
	"fmt"
	"slices"
)

type (
	// StatusSelector identifies an exact HTTP status, a status class, or the
	// fallback response. Exactly one of Code and Class is set.
	StatusSelector struct {
		// Code is an exact status in the range 100 through 599, or zero when
		// Class is set.
		Code int
		// Class is a status class digit from 1 through 5, -1 for fallback,
		// or zero when Code is set.
		Class int
	}

	// StatusViolationError reports a status outside a response selector's final
	// domain. It is a framework error and carries no designed error name.
	StatusViolationError struct {
		// Code is the attempted response status.
		Code int
		// Selector is the response selector whose domain rejected Code.
		Selector StatusSelector
	}
)

// SelectFinal selects a declared response for code in the range 200 through
// 599. Exact status takes precedence over class, then fallback, independently
// of declaration order. It returns false when no declared selector matches.
func SelectFinal(keys []StatusSelector, code int) (StatusSelector, bool) {
	if code < 200 || code > 599 {
		return StatusSelector{}, false
	}
	for _, candidate := range [...]StatusSelector{{Code: code}, {Class: code / 100}, {Class: -1}} {
		if slices.Contains(keys, candidate) {
			return candidate, true
		}
	}
	return StatusSelector{}, false
}

// SelectInformational selects a declared informational response for code in
// the range 100 through 199, excluding 101. Exact status takes precedence over
// class 1; fallback is never informational. This selects observed responses,
// including 100, rather than defining the server's authored emission domain.
func SelectInformational(keys []StatusSelector, code int) (StatusSelector, bool) {
	if code < 100 || code > 199 || code == 101 {
		return StatusSelector{}, false
	}
	for _, candidate := range [...]StatusSelector{{Code: code}, {Class: 1}} {
		if slices.Contains(keys, candidate) {
			return candidate, true
		}
	}
	return StatusSelector{}, false
}

// FinalStatusDomain returns a new ascending slice of final statuses selected
// by selector among keys. A body-bearing response excludes 204 and 304. An
// undeclared selector has an empty domain.
func FinalStatusDomain(keys []StatusSelector, selector StatusSelector, hasBody bool) []int {
	var domain []int
	for code := 200; code <= 599; code++ {
		if finalStatusAllowed(keys, selector, code, hasBody) {
			domain = append(domain, code)
		}
	}
	return domain
}

// CheckFinalStatus returns nil if code belongs to selector's final domain among
// keys, or a *StatusViolationError otherwise. A body-bearing response excludes
// 204 and 304. This check does not write headers or select a different response.
func CheckFinalStatus(keys []StatusSelector, selector StatusSelector, code int, hasBody bool) error {
	if finalStatusAllowed(keys, selector, code, hasBody) {
		return nil
	}
	return &StatusViolationError{Code: code, Selector: selector}
}

// Error describes the rejected status and response selector.
func (e *StatusViolationError) Error() string {
	return fmt.Sprintf("HTTP status %d is outside the final response domain for selector {Code:%d Class:%d}",
		e.Code, e.Selector.Code, e.Selector.Class)
}

func finalStatusAllowed(keys []StatusSelector, selector StatusSelector, code int, hasBody bool) bool {
	if hasBody && (code == 204 || code == 304) {
		return false
	}
	selected, ok := SelectFinal(keys, code)
	return ok && selected == selector
}
