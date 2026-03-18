package goa

import (
	"errors"
	"testing"
)

type testRemediableError struct{}

func (testRemediableError) Error() string { return "unsafe detail" }

func (testRemediableError) GoaErrorRemedy() *ErrorRemedy {
	return &ErrorRemedy{
		Code:        "test.fix",
		SafeMessage: "safe detail",
		RetryHint:   "retry later",
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

func TestAsError(t *testing.T) {
	err := MissingFieldError("foo", "bar")
	se := asError(err)
	if !errors.Is(err, se) {
		t.Errorf("got %#v, want %#v", se, err)
	}
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
}
