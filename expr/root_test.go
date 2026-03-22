package expr

import (
	"errors"
	"fmt"
	"testing"

	"github.com/CaliLuke/loom/eval"
)

func TestRootExprValidate(t *testing.T) {
	cases := map[string]struct {
		api          *APIExpr
		sessionAuths []*SessionAuthExpr
		expected     *eval.ValidationErrors
	}{
		"no error": {
			api: &APIExpr{
				Name: "foo",
			},
			expected: &eval.ValidationErrors{
				Errors: []error{},
			},
		},
		"missing api declaration": {
			api: nil,
			expected: &eval.ValidationErrors{
				Errors: []error{fmt.Errorf("Missing API declaration")},
			},
		},
		"invalid session auth": {
			api: &APIExpr{
				Name: "foo",
			},
			sessionAuths: []*SessionAuthExpr{
				{
					Name: "broken",
					Transports: []*SessionTransportExpr{
						{
							Kind:      SessionCookieTransportKind,
							Scheme:    &SchemeExpr{Kind: JWTKind, SchemeName: "jwt"},
							FieldName: "browser_session",
						},
					},
				},
			},
			expected: &eval.ValidationErrors{
				Errors: []error{fmt.Errorf(`cookie transport must use an API key security scheme`)},
			},
		},
	}

	for k, tc := range cases {
		e := RootExpr{
			API:          tc.api,
			SessionAuths: tc.sessionAuths,
		}
		var actual *eval.ValidationErrors
		if errors.As(e.Validate(), &actual); len(tc.expected.Errors) != len(actual.Errors) {
			t.Errorf("%s: expected the number of error values to match %d got %d ", k, len(tc.expected.Errors), len(actual.Errors))
		} else {
			for i, err := range actual.Errors {
				if err.Error() != tc.expected.Errors[i].Error() {
					t.Errorf("%s: got %#v, expected %#v at index %d", k, err, tc.expected.Errors[i], i)
				}
			}
		}
	}
}

func TestMetaExpr_Last(t *testing.T) {
	tt := map[string]struct {
		meta  MetaExpr
		value string
		ok    bool
	}{
		"no-key": {
			MetaExpr{},
			"",
			false,
		},
		"key-no-values": {
			MetaExpr{
				"test:key": []string{},
			},
			"",
			false,
		},
		"key-with-one-value": {
			MetaExpr{
				"test:key": []string{
					"value-one",
				},
			},
			"value-one",
			true,
		},
		"key-with-multiple-values": {
			MetaExpr{
				"test:key": []string{
					"value-one",
					"value-two",
					"value-n",
				},
			},
			"value-n",
			true,
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			value, ok := tc.meta.Last("test:key")
			if tc.ok != ok {
				t.Errorf("expected ok to be %v, got %v", tc.ok, ok)
			}
			if tc.value != value {
				t.Errorf("expected value to be %s, got %s", value, value)
			}
		})
	}
}
