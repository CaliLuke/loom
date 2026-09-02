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

func TestRootExprFinalizeStabilizesImplicitServiceOrder(t *testing.T) {
	alpha := &ServiceExpr{Name: "alpha"}
	zebra := &ServiceExpr{Name: "zebra"}
	root := &RootExpr{
		API: NewAPIExpr("catalog", nil),
		Services: []*ServiceExpr{
			zebra,
			alpha,
		},
	}
	root.API.HTTP.Services = []*HTTPServiceExpr{
		{ServiceExpr: zebra},
		{ServiceExpr: alpha},
	}
	previousRoot := Root
	Root = root
	defer func() {
		Root = previousRoot
	}()

	root.Finalize()

	if root.Services[0].Name != "alpha" || root.Services[1].Name != "zebra" {
		t.Errorf("unexpected service order: %q, %q", root.Services[0].Name, root.Services[1].Name)
	}
	if root.API.HTTP.Services[0].Name() != "alpha" || root.API.HTTP.Services[1].Name() != "zebra" {
		t.Errorf(
			"unexpected HTTP service order: %q, %q",
			root.API.HTTP.Services[0].Name(),
			root.API.HTTP.Services[1].Name(),
		)
	}
	if root.API.Servers[0].Services[0] != "alpha" || root.API.Servers[0].Services[1] != "zebra" {
		t.Errorf(
			"unexpected implicit server service order: %q, %q",
			root.API.Servers[0].Services[0],
			root.API.Servers[0].Services[1],
		)
	}
}

func TestRootExprFinalizePreservesExplicitServerServiceOrder(t *testing.T) {
	root := &RootExpr{
		API: NewAPIExpr("catalog", nil),
		Services: []*ServiceExpr{
			{Name: "zebra"},
			{Name: "alpha"},
		},
	}
	root.API.Servers = []*ServerExpr{{Services: []string{"zebra", "alpha"}}}
	previousRoot := Root
	Root = root
	defer func() {
		Root = previousRoot
	}()

	root.Finalize()

	if root.API.Servers[0].Services[0] != "zebra" || root.API.Servers[0].Services[1] != "alpha" {
		t.Errorf(
			"unexpected explicit server service order: %q, %q",
			root.API.Servers[0].Services[0],
			root.API.Servers[0].Services[1],
		)
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
