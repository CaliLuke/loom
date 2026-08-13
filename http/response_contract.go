package http

import (
	"fmt"
	"mime"
	"net/http"
	"net/textproto"
	"slices"
	"strings"
)

type (
	// ResponseContractCaseKind identifies whether a generated response contract
	// case describes a successful result or a service error.
	ResponseContractCaseKind string

	// ResponseContractCase describes the HTTP wire invariants for one declared
	// response branch. It does not describe how application code reaches that
	// branch.
	ResponseContractCase struct {
		// ID is stable while the service response contract is unchanged.
		ID string
		// Kind identifies a successful result or service error response.
		Kind ResponseContractCaseKind
		// StatusCode is the exact declared HTTP status code.
		StatusCode int
		// ErrorName is the expected Loom-Error header value for an error case.
		ErrorName string
		// ContentTypes lists the allowed response media types.
		ContentTypes []string
		// RequiredHeaders lists declared response headers that must be present.
		RequiredHeaders []string
		// RequiredCookies lists declared response cookies that must be present.
		RequiredCookies []string
	}
)

const (
	// ResponseContractSuccess identifies a successful response contract case.
	ResponseContractSuccess ResponseContractCaseKind = "success"
	// ResponseContractError identifies an error response contract case.
	ResponseContractError ResponseContractCaseKind = "error"
)

// ValidateResponseContract validates the transport-owned wire invariants in
// contract against resp. Applications remain responsible for arranging the
// service state and request that produce the response.
func ValidateResponseContract(resp *http.Response, contract ResponseContractCase) error {
	prefix := fmt.Sprintf("response contract %q", contract.ID)
	if resp == nil {
		return fmt.Errorf("%s: response is nil", prefix)
	}
	if resp.StatusCode != contract.StatusCode {
		return fmt.Errorf("%s: status is %d, want %d", prefix, resp.StatusCode, contract.StatusCode)
	}
	if err := validateResponseContractContentType(resp.Header, contract.ContentTypes); err != nil {
		return fmt.Errorf("%s: %w", prefix, err)
	}
	actualErrorName := resp.Header.Get("Loom-Error")
	if contract.Kind == ResponseContractError {
		if actualErrorName != contract.ErrorName {
			return fmt.Errorf("%s: Loom-Error is %q, want %q", prefix, actualErrorName, contract.ErrorName)
		}
	} else if actualErrorName != "" {
		return fmt.Errorf("%s: Loom-Error is %q, want empty", prefix, actualErrorName)
	}
	for _, name := range contract.RequiredHeaders {
		if !responseHeaderPresent(resp.Header, name) {
			return fmt.Errorf("%s: required header %q is missing", prefix, name)
		}
	}
	for _, name := range contract.RequiredCookies {
		if !responseCookiePresent(resp, name) {
			return fmt.Errorf("%s: required cookie %q is missing", prefix, name)
		}
	}
	return nil
}

func validateResponseContractContentType(header http.Header, declared []string) error {
	if len(declared) == 0 {
		return nil
	}
	actual, _, err := mime.ParseMediaType(header.Get("Content-Type"))
	if err != nil {
		return fmt.Errorf("parse Content-Type %q: %w", header.Get("Content-Type"), err)
	}
	expected := make([]string, 0, len(declared))
	for _, contentType := range declared {
		mediaType, _, err := mime.ParseMediaType(contentType)
		if err != nil {
			return fmt.Errorf("parse declared Content-Type %q: %w", contentType, err)
		}
		if !slices.Contains(expected, mediaType) {
			expected = append(expected, mediaType)
		}
	}
	if !slices.Contains(expected, actual) {
		return fmt.Errorf("Content-Type is %q, want one of %v", actual, expected)
	}
	return nil
}

func responseHeaderPresent(header http.Header, target string) bool {
	target = textproto.CanonicalMIMEHeaderKey(target)
	for name := range header {
		if strings.EqualFold(textproto.CanonicalMIMEHeaderKey(name), target) {
			return true
		}
	}
	return false
}

func responseCookiePresent(resp *http.Response, target string) bool {
	for _, cookie := range resp.Cookies() {
		if cookie.Name == target {
			return true
		}
	}
	return false
}
