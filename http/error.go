package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode"

	loom "github.com/CaliLuke/loom/pkg"
)

const (
	// ProblemJSONContentType is the default media type for Loom HTTP problem
	// responses.
	ProblemJSONContentType = "application/problem+json"

	defaultProblemTypeBaseURI = "https://github.com/CaliLuke/loom/problems/"
)

type (
	// ProblemResponse is the default RFC 9457 problem document encoded in HTTP
	// responses for Loom service errors.
	ProblemResponse struct {
		Type      string  `json:"type" xml:"type" form:"type"`
		Title     string  `json:"title" xml:"title" form:"title"`
		Status    int     `json:"status" xml:"status" form:"status"`
		Detail    string  `json:"detail" xml:"detail" form:"detail"`
		Instance  string  `json:"instance" xml:"instance" form:"instance"`
		Code      string  `json:"code" xml:"code" form:"code"`
		RetryHint *string `json:"retry_hint,omitempty" xml:"retry_hint,omitempty" form:"retry_hint,omitempty"`
	}

	// ErrorResponse is kept as an alias for compatibility with older Loom code.
	ErrorResponse = ProblemResponse

	// Statuser is implemented by error response object to provide the response
	// HTTP status code.
	Statuser interface {
		// StatusCode return the HTTP status code used to encode the response
		// when not defined in the design.
		StatusCode() int
	}
)

// NewErrorResponse creates the default RFC 9457 problem document for err.
func NewErrorResponse(ctx context.Context, err error) Statuser {
	return NewProblemResponse(ctx, err, 0, "", "")
}

// NewProblemResponse creates a problem document from err using the supplied
// status and optional explicit type/title overrides.
func NewProblemResponse(_ context.Context, err error, status int, problemType, problemTitle string) *ProblemResponse {
	var gerr *loom.ServiceError
	detailErr := err
	if !errors.As(err, &gerr) {
		gerr = loom.Fault("internal server error")
		detailErr = gerr
	}
	if status == 0 {
		status = inferProblemStatus(gerr)
	}
	problemType, problemTitle = ResolveProblemTypeAndTitle(gerr.Name, status, problemType, problemTitle)
	resp := &ProblemResponse{
		Type:     problemType,
		Title:    problemTitle,
		Status:   status,
		Detail:   loom.ErrorSafeMessage(detailErr),
		Instance: ProblemInstanceURI(gerr.ID),
		Code:     gerr.Name,
	}
	if retryHint := loom.ErrorRetryHint(err); retryHint != "" {
		resp.RetryHint = &retryHint
	}
	return resp
}

// ResolveProblemTypeAndTitle resolves the final RFC 9457 type/title pair for a
// problem code and HTTP status. Explicit overrides win over generated values.
func ResolveProblemTypeAndTitle(code string, status int, explicitType, explicitTitle string) (string, string) {
	problemType := strings.TrimSpace(explicitType)
	title := strings.TrimSpace(explicitTitle)
	if problemType == "" {
		if isGenericHTTPProblem(code, status) {
			problemType = "about:blank"
		} else {
			problemType = DefaultProblemTypeURI(code)
		}
	}
	if title == "" {
		if problemType == "about:blank" {
			title = http.StatusText(status)
		} else {
			title = DefaultProblemTitle(code, status)
		}
	}
	return problemType, title
}

// DefaultProblemTypeURI returns the deterministic default problem type URI for
// code when no explicit override is provided.
func DefaultProblemTypeURI(code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return "about:blank"
	}
	return defaultProblemTypeBaseURI + strings.ReplaceAll(code, "_", "-")
}

// DefaultProblemTitle returns the deterministic default problem title for code
// and status when no explicit title override is provided.
func DefaultProblemTitle(code string, status int) string {
	if isGenericHTTPProblem(code, status) {
		if title := http.StatusText(status); title != "" {
			return title
		}
	}
	if humanized := humanizeProblemCode(code); humanized != "" {
		return humanized
	}
	return http.StatusText(status)
}

// ProblemInstanceURI returns a stable URI reference for a specific problem
// occurrence.
func ProblemInstanceURI(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	return "urn:loom:error:" + id
}

// ProblemInstanceID extracts the Loom error instance ID from a problem
// instance URI when it uses the framework default URN form.
func ProblemInstanceID(instance string) string {
	const prefix = "urn:loom:error:"
	if strings.HasPrefix(instance, prefix) {
		return strings.TrimPrefix(instance, prefix)
	}
	return instance
}

// StatusCode returns the HTTP status code carried by the problem document.
func (resp *ProblemResponse) StatusCode() int {
	if resp == nil {
		return http.StatusInternalServerError
	}
	if resp.Status != 0 {
		return resp.Status
	}
	return http.StatusInternalServerError
}

func inferProblemStatus(err *loom.ServiceError) int {
	if err.Name == loom.RequestBodyTooLarge {
		return http.StatusRequestEntityTooLarge
	}
	if err.Name == loom.UnsupportedMediaType {
		return http.StatusUnsupportedMediaType
	}
	if err.Name == loom.NotAcceptable {
		return http.StatusNotAcceptable
	}
	if err.Fault {
		return http.StatusInternalServerError
	}
	if err.Timeout {
		if err.Temporary {
			return http.StatusGatewayTimeout
		}
		return http.StatusRequestTimeout
	}
	if err.Temporary {
		return http.StatusServiceUnavailable
	}
	return http.StatusBadRequest
}

func isGenericHTTPProblem(code string, status int) bool {
	switch status {
	case http.StatusBadRequest:
		return code == "bad_request"
	case http.StatusUnauthorized:
		return code == "unauthorized"
	case http.StatusForbidden:
		return code == "forbidden"
	case http.StatusNotFound:
		return code == "not_found"
	case http.StatusConflict:
		return code == "conflict"
	case http.StatusRequestTimeout:
		return code == "request_timeout"
	case http.StatusRequestEntityTooLarge:
		return code == loom.RequestBodyTooLarge
	case http.StatusUnsupportedMediaType:
		return code == loom.UnsupportedMediaType
	case http.StatusNotAcceptable:
		return code == loom.NotAcceptable
	case http.StatusUnprocessableEntity:
		return code == "unprocessable_entity"
	case http.StatusTooManyRequests:
		return code == "too_many_requests" || code == "rate_limited"
	case http.StatusInternalServerError:
		return code == "internal_error" || code == "fault"
	case http.StatusServiceUnavailable:
		return code == "service_unavailable"
	case http.StatusGatewayTimeout:
		return code == "gateway_timeout"
	default:
		return false
	}
}

func humanizeProblemCode(code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return ""
	}
	parts := strings.FieldsFunc(code, func(r rune) bool {
		return r == '_' || r == '-' || unicode.IsSpace(r)
	})
	if len(parts) == 0 {
		return ""
	}
	for i, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(strings.ToLower(part))
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, " ")
}

// ProblemErrorFromBody rebuilds a Loom service error from a decoded problem
// document body.
func ProblemErrorFromBody(
	code string,
	status int,
	detail string,
	instance string,
	retryHint *string,
) *loom.ServiceError {
	timeout, temporary, fault := inferProblemFlags(status)
	err := loom.NewServiceError(fmt.Errorf("%s", detail), code, timeout, temporary, fault)
	err.ID = ProblemInstanceID(instance)
	if retryHint != nil && *retryHint != "" {
		err = loom.WithErrorRemedy(err, &loom.ErrorRemedy{
			SafeMessage: detail,
			RetryHint:   *retryHint,
		})
	}
	return err
}

func inferProblemFlags(status int) (timeout, temporary, fault bool) {
	switch status {
	case http.StatusInternalServerError:
		return false, false, true
	case http.StatusRequestTimeout:
		return true, false, false
	case http.StatusGatewayTimeout:
		return true, true, false
	case http.StatusServiceUnavailable, http.StatusTooManyRequests:
		return false, true, false
	default:
		return false, false, false
	}
}
