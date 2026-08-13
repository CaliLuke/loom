package loom

import (
	"errors"
	"reflect"
	"testing"
)

type testRemediableError struct{}
type testStatusCodeError struct{}
type testStatusError struct{}

func (testRemediableError) Error() string { return "unsafe detail" }
func (testStatusCodeError) Error() string { return "status code detail" }
func (testStatusError) Error() string     { return "status detail" }
func (testStatusCodeError) StatusCode() int {
	return 422
}
func (testStatusError) Status() int {
	return 404
}

func (testRemediableError) LoomErrorRemedy() *ErrorRemedy {
	return &ErrorRemedy{
		Code:        "test.fix",
		SafeMessage: "safe detail",
		RetryHint:   "retry later",
	}
}

func TestServiceErrorUsesLoomErrorNameOnly(t *testing.T) {
	typeOfServiceError := reflect.TypeOf(&ServiceError{})
	if _, ok := typeOfServiceError.MethodByName("ErrorName"); ok {
		t.Error("ServiceError exposes the removed ErrorName compatibility method")
	}
	if _, ok := typeOfServiceError.MethodByName("LoomErrorName"); !ok {
		t.Error("ServiceError does not expose LoomErrorName")
	}
}

func TestRequestBodyTooLargeError(t *testing.T) {
	err := RequestBodyTooLargeError()

	var serviceErr *ServiceError
	if !errors.As(err, &serviceErr) {
		t.Fatalf("got %T, want *ServiceError", err)
	}
	if serviceErr.Name != RequestBodyTooLarge {
		t.Errorf("got name %q, want %q", serviceErr.Name, RequestBodyTooLarge)
	}
	if serviceErr.Message != "request body too large" {
		t.Errorf("got message %q, want request body too large", serviceErr.Message)
	}
}

func TestServiceErrorUnwrap(t *testing.T) {
	var (
		errFoo          = errors.New("foo")
		errBar          = errors.New("bar")
		serviceErrorFoo = NewServiceError(errFoo, "foo", false, false, false)
		serviceErrorBar = NewServiceError(errBar, "bar", false, false, false)
	)
	cases := map[string]struct {
		err  error
		want error
	}{
		"service error": {
			err:  serviceErrorFoo,
			want: errFoo,
		},
		"merged service error": {
			err:  MergeErrors(serviceErrorFoo, serviceErrorBar),
			want: errors.Join(errFoo, errBar),
		},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			got := errors.Unwrap(tc.err)
			if errs, ok := tc.want.(interface{ Unwrap() []error }); ok {
				for _, e := range errs.Unwrap() {
					if !errors.Is(got, e) {
						t.Errorf("got %#v, want %#v", got, tc.want)
					}
				}
			} else if !errors.Is(got, tc.want) {
				t.Errorf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestMergeErrorsPreservesHistoryEntryMessages(t *testing.T) {
	first := MissingFieldError("username", "body")
	second := InvalidFieldTypeError("age", "old", "integer")
	firstMessage := first.Error()
	secondMessage := second.Error()

	err := MergeErrors(first, second)
	var serviceErr *ServiceError
	if !errors.As(err, &serviceErr) {
		t.Fatalf("got %T, want *ServiceError", err)
	}
	history := serviceErr.History()
	if len(history) != 2 {
		t.Fatalf("got history length %d, want 2", len(history))
	}
	if history[0].Message != firstMessage {
		t.Errorf("got first history message %q, want %q", history[0].Message, firstMessage)
	}
	if history[1].Message != secondMessage {
		t.Errorf("got second history message %q, want %q", history[1].Message, secondMessage)
	}
}

func TestWithErrorHistoryPreservesSingleEntry(t *testing.T) {
	top := NewServiceError(errors.New("top"), "top", false, false, false)
	var entry *ServiceError
	if !errors.As(MissingFieldError("username", "body"), &entry) {
		t.Fatal("missing field error did not produce ServiceError")
	}

	got := WithErrorHistory(top, entry)
	history := got.History()
	if len(history) != 1 {
		t.Fatalf("got history length %d, want 1", len(history))
	}
	if history[0].Message != entry.Error() {
		t.Errorf("got history message %q, want %q", history[0].Message, entry.Error())
	}
	if got.Message != "top" {
		t.Errorf("got top-level message %q, want top", got.Message)
	}
}

func TestAsError(t *testing.T) {
	err := MissingFieldError("foo", "bar")
	se := asError(err)
	if !errors.Is(err, se) {
		t.Errorf("got %#v, want %#v", se, err)
	}
}

func TestGeneratedValidationErrorSafeMessages(t *testing.T) {
	t.Run("missing field", func(t *testing.T) {
		err := MissingFieldError("value", "body")
		if got := ErrorSafeMessage(err); got != "Missing required field: value" {
			t.Errorf("got safe message %q", got)
		}
		if got := err.Error(); got != `"value" is missing from body` {
			t.Errorf("got error %q", got)
		}
	})

	t.Run("invalid enum", func(t *testing.T) {
		err := InvalidEnumValueError("action", "GetActive", []any{"list", "get_active"})
		want := `invalid value for "action": got "GetActive", expected one of "list", "get_active"`
		if got := ErrorSafeMessage(err); got != want {
			t.Errorf("got safe message %q", got)
		}
		if got := err.Error(); got != want {
			t.Errorf("got error %q", got)
		}
	})
}

func TestExtractErrorRemedy(t *testing.T) {
	t.Run("service error with remedy", func(t *testing.T) {
		err := WithErrorRemedy(NewServiceError(errors.New("boom"), "boom", false, false, false), &ErrorRemedy{
			Code:        "boom.fix",
			SafeMessage: "safe boom",
			RetryHint:   "retry safely",
		})
		remedy := ExtractErrorRemedy(err)
		if remedy == nil {
			t.Fatal("expected remedy")
		}
		if remedy.Code != "boom.fix" {
			t.Errorf("got code %q", remedy.Code)
		}
		if ErrorSafeMessage(err) != "safe boom" {
			t.Errorf("got safe message %q", ErrorSafeMessage(err))
		}
		if ErrorRetryHint(err) != "retry safely" {
			t.Errorf("got retry hint %q", ErrorRetryHint(err))
		}
	})

	t.Run("custom remedier", func(t *testing.T) {
		err := testRemediableError{}
		if got := ErrorRemedyCode(err); got != "test.fix" {
			t.Errorf("got code %q", got)
		}
		if got := ErrorSafeMessage(err); got != "safe detail" {
			t.Errorf("got safe message %q", got)
		}
	})

	t.Run("fallbacks stay conservative", func(t *testing.T) {
		err := errors.New("raw detail")
		if remedy := ExtractErrorRemedy(err); remedy != nil {
			t.Fatalf("expected nil remedy, got %#v", remedy)
		}
		if got := ErrorSafeMessage(err); got != "raw detail" {
			t.Errorf("got safe message %q", got)
		}
		if got := ErrorRetryHint(err); got != "" {
			t.Errorf("got retry hint %q", got)
		}
	})

	t.Run("generic status code extraction", func(t *testing.T) {
		if got, ok := ErrorStatusCode(testStatusCodeError{}); !ok || got != 422 {
			t.Errorf("got status code (%d, %t)", got, ok)
		}
		if got, ok := ErrorStatusCode(testStatusError{}); !ok || got != 404 {
			t.Errorf("got status (%d, %t)", got, ok)
		}
		if got, ok := ErrorStatusCode(errors.New("raw detail")); ok || got != 0 {
			t.Errorf("got unexpected status (%d, %t)", got, ok)
		}
	})
}
