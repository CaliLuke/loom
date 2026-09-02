package loom

import (
	"errors"
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/internal/identifier"
)

type (
	// ErrorRemedy describes machine-consumable remediation guidance attached to
	// an error.
	ErrorRemedy struct {
		// Code is the stable remediation code consumers may use to classify the
		// failure.
		Code string `json:"code,omitempty" xml:"code,omitempty" form:"code,omitempty"`
		// SafeMessage is the safe, user-facing message to surface without
		// leaking internal details.
		SafeMessage string `json:"safe_message,omitempty" xml:"safe_message,omitempty" form:"safe_message,omitempty"`
		// RetryHint is concise guidance on how to correct the request or retry.
		RetryHint string `json:"retry_hint,omitempty" xml:"retry_hint,omitempty" form:"retry_hint,omitempty"`
	}

	// ServiceError is the default error type used by the Loom package to
	// encode and decode error responses.
	ServiceError struct {
		// Name is a name for that class of errors.
		Name string
		// ID is a unique value for each occurrence of the error.
		ID string
		// Pointer to the field that caused this error, if appropriate
		Field *string
		// Message contains the specific error details.
		Message string
		// Is the error a timeout?
		Timeout bool
		// Is the error temporary?
		Temporary bool
		// Is the error a server-side fault?
		Fault bool
		// Remedy contains optional remediation guidance for the error.
		Remedy *ErrorRemedy
		// History tracks all the individual errors that were built into this error, should
		// this error have been merged.
		history []*ServiceError
		// err holds the original error if exists.
		err error
	}

	// LoomErrorNamer is an interface implemented by generated error structs that
	// exposes the name of the error as defined in the design.
	LoomErrorNamer interface {
		LoomErrorName() string
	}

	// LoomErrorRemedier is implemented by errors that expose remediation
	// guidance defined in the design.
	LoomErrorRemedier interface {
		LoomErrorRemedy() *ErrorRemedy
	}

	// LoomErrorStatuser is implemented by errors that expose an HTTP status code.
	LoomErrorStatuser interface {
		StatusCode() int
	}

	// LoomErrorStatusReporter is implemented by errors that expose an HTTP
	// status code via Status.
	LoomErrorStatusReporter interface {
		Status() int
	}
)

const (
	// InvalidFieldType is the error name for invalid field type errors.
	InvalidFieldType = "invalid_field_type"
	// MissingField is the error name for missing field errors.
	MissingField = "missing_field"
	// InvalidEnumValue is the error name for invalid enum value errors.
	InvalidEnumValue = "invalid_enum_value"
	// InvalidFormat is the error name for invalid format errors.
	InvalidFormat = "invalid_format"
	// InvalidPattern is the error name for invalid pattern errors.
	InvalidPattern = "invalid_pattern"
	// InvalidRange is the error name for invalid range errors.
	InvalidRange = "invalid_range"
	// InvalidLength is the error name for invalid length errors.
	InvalidLength = "invalid_length"
	// UnsupportedMediaType is the error name returned by the Loom decoder
	// when the content type of the HTTP request body is not supported.
	UnsupportedMediaType = "unsupported_media_type"
	// DecodePayload is the error name for decode payload errors.
	DecodePayload = "decode_payload"
	// RequestBodyTooLarge is the error name returned by the Loom decoder when
	// the HTTP request body exceeds the framework limit.
	RequestBodyTooLarge = "request_too_large"
	// NotAcceptable is the error name returned by strict HTTP response
	// negotiation when the request excludes every supported media type.
	NotAcceptable = "not_acceptable"
	// MissingPayload is the error name for missing payload errors.
	MissingPayload = "missing_payload"
)

// NewServiceError creates an error.
func NewServiceError(err error, name string, timeout, temporary, fault bool) *ServiceError {
	return &ServiceError{
		Name:      name,
		ID:        NewErrorID(),
		Message:   err.Error(),
		Timeout:   timeout,
		Temporary: temporary,
		Fault:     fault,
		err:       err,
	}
}

// Fault creates an error given a format and values a la fmt.Printf. The error
// has the Fault field set to true.
func Fault(format string, v ...any) *ServiceError {
	return newError("fault", false, false, true, format, v...)
}

// PermanentError creates an error given a name and a format and values a la
// fmt.Printf.
func PermanentError(name, format string, v ...any) *ServiceError {
	return newError(name, false, false, false, format, v...)
}

// TemporaryError is an error class that indicates that the error is temporary
// and that retrying the request may be successful. TemporaryError creates an
// error given a name and a format and values a la fmt.Printf. The error has the
// Temporary field set to true.
func TemporaryError(name, format string, v ...any) *ServiceError {
	return newError(name, false, true, false, format, v...)
}

// PermanentTimeoutError creates an error given a name and a format and values a
// la fmt.Printf. The error has the Timeout field set to true.
func PermanentTimeoutError(name, format string, v ...any) *ServiceError {
	return newError(name, true, false, false, format, v...)
}

// TemporaryTimeoutError creates an error given a name and a format and values a
// la fmt.Printf. The error has both the Timeout and Temporary fields set to
// true.
func TemporaryTimeoutError(name, format string, v ...any) *ServiceError {
	return newError(name, true, true, false, format, v...)
}

// MissingPayloadError is the error produced by the generated code when a
// request is missing a required payload.
func MissingPayloadError() error {
	return validationError(PermanentError(MissingPayload, "missing required payload"))
}

// DecodePayloadError is the error produced by the generated code when a request
// body cannot be decoded successfully.
func DecodePayloadError(msg string) error {
	return PermanentError(DecodePayload, "%s", msg)
}

// RequestBodyTooLargeError is the error produced by the Loom decoder when an
// HTTP request body exceeds the framework limit.
func RequestBodyTooLargeError() error {
	return PermanentError(RequestBodyTooLarge, "request body too large")
}

// NotAcceptableError is the error produced by strict HTTP response negotiation
// when the Accept header excludes every supported response media type.
func NotAcceptableError(accept string) error {
	return PermanentError(NotAcceptable, "no supported response media type satisfies Accept %q", accept)
}

// UnsupportedMediaTypeError is the error produced by the Loom decoder when the
// content type of the HTTP request body is not supported.
func UnsupportedMediaTypeError(ct string) error {
	return PermanentError(UnsupportedMediaType, "unsupported media type %s", ct)
}

// InvalidFieldTypeError is the error produced by the generated code when the
// type of a payload field does not match the type defined in the design.
func InvalidFieldTypeError(name string, val any, expected string) error {
	return validationError(withField(name, PermanentError(
		InvalidFieldType, "invalid value %#v for %q, must be a %s", val, name, expected)))
}

// InvalidNullElementError reports a null member in an array whose element
// contract does not allow null. The field path includes the concrete index.
func InvalidNullElementError(name string, index int) error {
	field := fmt.Sprintf("%s[%d]", name, index)
	message := fmt.Sprintf("invalid null value for %q; array element must be non-null", field)
	return validationErrorWithSafe(
		withField(field, PermanentError(InvalidFieldType, "%s", message)),
		message,
	)
}

// InvalidNullMapValueError reports a null map value whose contract does not
// allow null. The field path identifies the map key position.
func InvalidNullMapValueError(name string) error {
	message := fmt.Sprintf("invalid null value for %q; map value must be non-null", name)
	return validationErrorWithSafe(
		withField(name, PermanentError(InvalidFieldType, "%s", message)),
		message,
	)
}

// MissingFieldError is the error produced by the generated code when a payload
// is missing a required field.
func MissingFieldError(name, context string) error {
	return validationErrorWithSafe(
		withField(name, PermanentError(MissingField, "%q is missing from %s", name, context)),
		fmt.Sprintf("Missing required field: %s", name),
	)
}

// InvalidEnumValueError is the error produced by the generated code when the
// value of a payload field does not match one the values defined in the design
// Enum validation.
func InvalidEnumValueError(name string, val any, allowed []any) error {
	elems := make([]string, len(allowed))
	for i, a := range allowed {
		elems[i] = fmt.Sprintf("%#v", a)
	}
	message := fmt.Sprintf("invalid value for %q: got %#v, expected one of %s", name, val, strings.Join(elems, ", "))
	return validationErrorWithSafe(
		withField(name, PermanentError(InvalidEnumValue, "%s", message)),
		message,
	)
}

// InvalidFormatError is the error produced by the generated code when the value
// of a payload field does not match the format validation defined in the
// design.
func InvalidFormatError(name, target string, format Format, formatError error) error {
	return validationError(withField(name, PermanentError(
		InvalidFormat, "%s must be formatted as a %s but got value %q, %s", name, format, target, formatError.Error())))
}

// InvalidPatternError is the error produced by the generated code when the
// value of a payload field does not match the pattern validation defined in the
// design.
func InvalidPatternError(name, target, pattern string) error {
	return validationError(withField(name, PermanentError(
		InvalidPattern, "%s must match the regexp %q but got value %q", name, pattern, target)))
}

// InvalidRangeError is the error produced by the generated code when the value
// of a payload field does not match the range validation defined in the design.
// value may be an int or a float64.
func InvalidRangeError(name string, target, value any, min bool) error {
	comp := "greater or equal"
	if !min {
		comp = "lesser or equal"
	}
	return validationError(withField(name, PermanentError(
		InvalidRange, "%s must be %s than %d but got value %#v", name, comp, value, target)))
}

// InvalidLengthError is the error produced by the generated code when the value
// of a payload field does not match the length validation defined in the
// design.
func InvalidLengthError(name string, target any, ln, value int, min bool) error {
	comp := "greater or equal"
	if !min {
		comp = "lesser or equal"
	}
	return validationError(withField(name, PermanentError(
		InvalidLength, "length of %s must be %s than %d but got value %#v (len=%d)", name, comp, value, target, ln)))
}

func validationError(err *ServiceError) *ServiceError {
	return WithErrorRemedy(err, &ErrorRemedy{SafeMessage: "validation error"})
}

func validationErrorWithSafe(err *ServiceError, safeMessage string) *ServiceError {
	return WithErrorRemedy(err, &ErrorRemedy{SafeMessage: safeMessage})
}

// NewErrorID creates a unique 8 character ID that is well suited to use as an
// error identifier. It panics if operating-system entropy is unavailable.
func NewErrorID() string {
	// for the curious - simplifying a bit - the probability of 2 values
	// being equal for n 6-bytes values is n^2 / 2^49. For n = 1 million
	// this gives around 1 chance in 500. 6 bytes seems to be a good
	// trade-off between probability of clashes and length of ID (6 * 4/3 =
	// 8 chars) since clashes are not catastrophic.
	return identifier.MustBase64(6)
}

// MergeErrors updates an error by merging another into it. It first converts
// other into a ServiceError if not already one. The merge algorithm then:
//
// * uses the name of err if a ServiceError, the name of other otherwise.
//
// * appends both error messages.
//
// * computes Timeout and Temporary by "and"ing the fields of both errors.
//
// Merge returns the updated error. This makes it possible to return other when
// err is nil.
func MergeErrors(err, other error) error {
	if err == nil {
		if other == nil {
			return nil
		}
		return other
	}
	if other == nil {
		return err
	}
	e := asError(err)
	o := asError(other)
	if e.Name == "error" {
		e.Name = o.Name
	}

	// Combine error lineage. We only ever put original errors into the history slice, so we
	// don't need to worry about gaining intermediate merges.
	//
	// Do this before we modify ourselves, as History() may include us!
	e.history = append(cloneServiceErrorHistory(e.History()), cloneServiceErrorHistory(o.History())...)
	e.err = errors.Join(e.err, o.err)

	e.Message = e.Message + "; " + o.Message
	e.Timeout = e.Timeout && o.Timeout
	e.Temporary = e.Temporary && o.Temporary
	e.Fault = e.Fault && o.Fault

	return e
}

// History returns the history of error revisions, ignoring the result of any merges.
func (e *ServiceError) History() []*ServiceError {
	if len(e.history) > 0 {
		return e.history
	}

	return []*ServiceError{e}
}

// WithErrorHistory returns err with the provided original error history.
func WithErrorHistory(err *ServiceError, history ...*ServiceError) *ServiceError {
	if err == nil {
		return nil
	}
	err.history = cloneServiceErrorHistory(history)
	return err
}

func cloneServiceErrorHistory(history []*ServiceError) []*ServiceError {
	if len(history) == 0 {
		return nil
	}
	clones := make([]*ServiceError, 0, len(history))
	for _, entry := range history {
		if entry == nil {
			continue
		}
		clone := *entry
		if entry.Field != nil {
			field := *entry.Field
			clone.Field = &field
		}
		clones = append(clones, &clone)
	}
	return clones
}

// Error returns the error message.
func (e *ServiceError) Error() string { return e.Message }

// LoomErrorName returns the error name.
func (e *ServiceError) LoomErrorName() string { return e.Name }

// LoomErrorRemedy returns the remediation guidance attached to the error, if
// any.
func (e *ServiceError) LoomErrorRemedy() *ErrorRemedy {
	if e == nil {
		return nil
	}
	return e.Remedy
}

func (e *ServiceError) Unwrap() error { return e.err }

func withField(field string, err *ServiceError) *ServiceError {
	err.Field = &field
	return err
}

func newError(name string, timeout, temporary, fault bool, format string, v ...any) *ServiceError {
	return &ServiceError{
		Name:      name,
		ID:        NewErrorID(),
		Message:   fmt.Sprintf(format, v...),
		Timeout:   timeout,
		Temporary: temporary,
		Fault:     fault,
	}
}

func asError(err error) *ServiceError {
	var e *ServiceError
	if !errors.As(err, &e) {
		return &ServiceError{
			Name:    "error",
			ID:      NewErrorID(),
			Message: err.Error(),
			Fault:   true, // Default to fault for unexpected errors
			err:     err,
		}
	}
	return e
}
