package expr

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"
)

type (
	// HTTPStatusClass identifies a hundred-status range or the final fallback.
	HTTPStatusClass int

	// HTTPStatusSelector identifies an exact HTTP status, a class or the fallback.
	// Exactly one of Code and Class must be set.
	HTTPStatusSelector struct {
		// Code is an exact HTTP status from 100 through 599, or zero for a class.
		Code int
		// Class is the selected range or fallback, or zero for an exact status.
		Class HTTPStatusClass
	}

	// HTTPResponsePhase distinguishes informational, upgrade and final responses.
	// The zero value does not identify a valid phase.
	HTTPResponsePhase int
)

const (
	// Status1XX selects informational statuses other than the 101 upgrade.
	Status1XX HTTPStatusClass = 1
	// Status2XX selects successful final statuses.
	Status2XX HTTPStatusClass = 2
	// Status3XX selects redirection final statuses.
	Status3XX HTTPStatusClass = 3
	// Status4XX selects client-error final statuses.
	Status4XX HTTPStatusClass = 4
	// Status5XX selects server-error final statuses.
	Status5XX HTTPStatusClass = 5
	// StatusFallback selects final statuses not covered by an exact key or class.
	StatusFallback HTTPStatusClass = -1
)

const (
	// PhaseInformational identifies interim responses other than upgrades.
	PhaseInformational HTTPResponsePhase = iota + 1
	// PhaseUpgrade identifies the 101 Switching Protocols handshake.
	PhaseUpgrade
	// PhaseFinal identifies final responses from 200 through 599.
	PhaseFinal
)

// ParseOpenAPIStatusKey parses an exact three-digit status from 100 through 599,
// an uppercase range key from 1XX through 5XX, or the case-sensitive "default".
func ParseOpenAPIStatusKey(key string) (HTTPStatusSelector, error) {
	if key == "default" {
		return HTTPStatusSelector{Class: StatusFallback}, nil
	}
	if len(key) == 3 && key[0] >= '1' && key[0] <= '5' {
		if key[1:] == "XX" {
			return HTTPStatusSelector{Class: HTTPStatusClass(key[0] - '0')}, nil
		}
		if key[1] >= '0' && key[1] <= '9' && key[2] >= '0' && key[2] <= '9' {
			return HTTPStatusSelector{Code: int(key[0]-'0')*100 + int(key[1]-'0')*10 + int(key[2]-'0')}, nil
		}
	}
	return HTTPStatusSelector{}, fmt.Errorf("invalid OpenAPI status key %q: expected 100..599, 1XX..5XX or default", key)
}

// SelectFinal selects a declared key for code using exact, class, then fallback
// precedence, independent of declaration order. Codes outside 200..599 fail.
func SelectFinal(keys []HTTPStatusSelector, code int) (HTTPStatusSelector, bool) {
	if code < 200 || code > 599 {
		return HTTPStatusSelector{}, false
	}
	for _, selector := range [...]HTTPStatusSelector{{Code: code}, {Class: HTTPStatusClass(code / 100)}, {Class: StatusFallback}} {
		if slices.Contains(keys, selector) {
			return selector, true
		}
	}
	return HTTPStatusSelector{}, false
}

// SelectInformational selects an exact key or 1XX for code. It accepts 100..199
// except 101, and never selects the final fallback.
func SelectInformational(keys []HTTPStatusSelector, code int) (HTTPStatusSelector, bool) {
	if code < 100 || code > 199 || code == 101 {
		return HTTPStatusSelector{}, false
	}
	for _, selector := range [...]HTTPStatusSelector{{Code: code}, {Class: Status1XX}} {
		if slices.Contains(keys, selector) {
			return selector, true
		}
	}
	return HTTPStatusSelector{}, false
}

// FinalRuntimeDomain returns a new ascending slice of final statuses selected by
// selector among keys. Body-bearing responses exclude 204 and 304.
func FinalRuntimeDomain(keys []HTTPStatusSelector, selector HTTPStatusSelector, hasBody bool) []int {
	var domain []int
	for code := 200; code <= 599; code++ {
		if hasBody && (code == 204 || code == 304) {
			continue
		}
		if selected, ok := SelectFinal(keys, code); ok && selected == selector {
			domain = append(domain, code)
		}
	}
	return domain
}

// InformationalEmissionDomain returns a new ascending slice of authored interim
// statuses selected by selector among keys. Automatic 100 and upgrade 101 are
// excluded, even though the client can select 100 as informational.
func InformationalEmissionDomain(keys []HTTPStatusSelector, selector HTTPStatusSelector) []int {
	var domain []int
	for code := 102; code <= 199; code++ {
		if selected, ok := SelectInformational(keys, code); ok && selected == selector {
			domain = append(domain, code)
		}
	}
	return domain
}

// informationalObservationDomain includes automatic statuses that a declared
// channel can receive. It is independent from the narrower server emission domain.
func informationalObservationDomain(keys []HTTPStatusSelector, selector HTTPStatusSelector) []int {
	var domain []int
	for code := 100; code <= 199; code++ {
		if selected, ok := SelectInformational(keys, code); ok && selected == selector {
			domain = append(domain, code)
		}
	}
	return domain
}

// Exact returns the status and true for a valid exact selector, otherwise zero
// and false.
func (s HTTPStatusSelector) Exact() (int, bool) {
	if !s.IsExact() {
		return 0, false
	}
	return s.Code, true
}

// IsExact reports whether the selector identifies a valid exact status.
func (s HTTPStatusSelector) IsExact() bool {
	return s.Class == 0 && s.Code >= 100 && s.Code <= 599
}

// Phase returns the selector's response phase, or zero for an invalid selector.
func (s HTTPStatusSelector) Phase() HTTPResponsePhase {
	if s.IsExact() {
		switch {
		case s.Code == 101:
			return PhaseUpgrade
		case s.Code < 200:
			return PhaseInformational
		default:
			return PhaseFinal
		}
	}
	if s.Code == 0 {
		switch {
		case s.Class == Status1XX:
			return PhaseInformational
		case s.Class >= Status2XX && s.Class <= Status5XX || s.Class == StatusFallback:
			return PhaseFinal
		}
	}
	return 0
}

// OpenAPIKey returns the exact status, uppercase class key or "default".
// It returns an empty string for an invalid selector.
func (s HTTPStatusSelector) OpenAPIKey() string {
	if s.IsExact() {
		return strconv.Itoa(s.Code)
	}
	if s.Code == 0 {
		switch {
		case s.Class == StatusFallback:
			return "default"
		case s.Class >= Status1XX && s.Class <= Status5XX:
			return strconv.Itoa(int(s.Class)) + "XX"
		}
	}
	return ""
}

// NameSuffix returns the shared response-body naming suffix for the selector.
// Known exact statuses retain http.StatusText verbatim; unknown exact statuses
// use Status followed by their number, classes use Status1XX through Status5XX,
// and the fallback uses StatusDefault. Invalid selectors return an empty string.
// Consumers apply their ordinary name normalization after composing this suffix.
func (s HTTPStatusSelector) NameSuffix() string {
	if s.IsExact() {
		if text := http.StatusText(s.Code); text != "" {
			return text
		}
		return "Status" + strconv.Itoa(s.Code)
	}
	if s.Code == 0 {
		switch {
		case s.Class == StatusFallback:
			return "StatusDefault"
		case s.Class >= Status1XX && s.Class <= Status5XX:
			return "Status" + s.OpenAPIKey()
		}
	}
	return ""
}
