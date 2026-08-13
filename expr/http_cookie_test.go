package expr_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/expr/testdata"
)

func TestHTTPResponseCookie(t *testing.T) {
	cases := []struct {
		Name    string
		DSL     func()
		Check   func(*testing.T, []*expr.HTTPResponseCookieExpr)
		ErrText string
	}{
		{
			Name: "cookie object result",
			DSL:  testdata.CookieObjectResultDSL,
			Check: func(t *testing.T, cookies []*expr.HTTPResponseCookieExpr) {
				t.Helper()
				assertResponseCookie(t, cookies[0], "cookie", "cookie", "", "", "", false, false, "")
			},
		},
		{
			Name: "cookie string result",
			DSL:  testdata.CookieStringResultDSL,
			Check: func(t *testing.T, cookies []*expr.HTTPResponseCookieExpr) {
				t.Helper()
				assertResponseCookie(t, cookies[0], "cookie", "cookie", "", "", "", false, false, "")
			},
		},
		{
			Name: "max-age",
			DSL:  testdata.CookieMaxAgeDSL,
			Check: func(t *testing.T, cookies []*expr.HTTPResponseCookieExpr) {
				t.Helper()
				assertResponseCookie(t, cookies[0], "cookie", "cookie", "", "", fmt.Sprintf("%d", testdata.CookieMaxAgeValue), false, false, "")
			},
		},
		{
			Name: "domain",
			DSL:  testdata.CookieDomainDSL,
			Check: func(t *testing.T, cookies []*expr.HTTPResponseCookieExpr) {
				t.Helper()
				assertResponseCookie(t, cookies[0], "cookie", "cookie", "", testdata.CookieDomainValue, "", false, false, "")
			},
		},
		{
			Name: "path",
			DSL:  testdata.CookiePathDSL,
			Check: func(t *testing.T, cookies []*expr.HTTPResponseCookieExpr) {
				t.Helper()
				assertResponseCookie(t, cookies[0], "cookie", "cookie", testdata.CookiePathValue, "", "", false, false, "")
			},
		},
		{
			Name: "secure",
			DSL:  testdata.CookieSecureDSL,
			Check: func(t *testing.T, cookies []*expr.HTTPResponseCookieExpr) {
				t.Helper()
				assertResponseCookie(t, cookies[0], "cookie", "cookie", "", "", "", true, false, "")
			},
		},
		{
			Name: "session cookie insecure override",
			DSL:  testdata.CookieInsecureDSL,
			Check: func(t *testing.T, cookies []*expr.HTTPResponseCookieExpr) {
				t.Helper()
				assertResponseCookie(t, cookies[0], "cookie", "cookie", "/", "", "", false, true, expr.CookieSameSiteLax)
			},
		},
		{
			Name: "secure setter overrides insecure setter",
			DSL:  testdata.CookieSecureAfterInsecureDSL,
			Check: func(t *testing.T, cookies []*expr.HTTPResponseCookieExpr) {
				t.Helper()
				assertResponseCookie(t, cookies[0], "cookie", "cookie", "", "", "", true, false, "")
			},
		},
		{
			Name: "http-only",
			DSL:  testdata.CookieHTTPOnlyDSL,
			Check: func(t *testing.T, cookies []*expr.HTTPResponseCookieExpr) {
				t.Helper()
				assertResponseCookie(t, cookies[0], "cookie", "cookie", "", "", "", false, true, "")
			},
		},
		{
			Name: "same-site",
			DSL:  testdata.CookieSameSiteDSL,
			Check: func(t *testing.T, cookies []*expr.HTTPResponseCookieExpr) {
				t.Helper()
				assertResponseCookie(t, cookies[0], "cookie", "cookie", "", "", "", false, false, testdata.CookieSameSiteValue)
			},
		},
		{
			Name: "session cookie defaults",
			DSL:  testdata.SessionCookieDefaultsDSL,
			Check: func(t *testing.T, cookies []*expr.HTTPResponseCookieExpr) {
				t.Helper()
				assertResponseCookie(t, cookies[0], "cookie", "cookie", "/", "", "", true, true, expr.CookieSameSiteLax)
			},
		},
		{
			Name: "session cookie overrides",
			DSL:  testdata.SessionCookieOverrideDSL,
			Check: func(t *testing.T, cookies []*expr.HTTPResponseCookieExpr) {
				t.Helper()
				assertResponseCookie(t, cookies[0], "cookie", "cookie", testdata.SessionCookieOverridePathValue, "", "", true, true, testdata.SessionCookieOverrideSameSiteValue)
			},
		},
		{
			Name: "session cookie overrides all",
			DSL:  testdata.SessionCookieOverrideAllDSL,
			Check: func(t *testing.T, cookies []*expr.HTTPResponseCookieExpr) {
				t.Helper()
				assertResponseCookie(t, cookies[0], "cookie", "cookie", testdata.SessionCookieOverridePathValue, testdata.SessionCookieOverrideDomainValue, fmt.Sprintf("%d", testdata.SessionCookieOverrideMaxAgeValue), true, true, testdata.SessionCookieOverrideSameSiteValue)
			},
		},
		{
			Name: "session cookie repeated setter",
			DSL:  testdata.SessionCookieRepeatedSetterDSL,
			Check: func(t *testing.T, cookies []*expr.HTTPResponseCookieExpr) {
				t.Helper()
				assertResponseCookie(t, cookies[0], "cookie", "cookie", testdata.SessionCookieOverridePathValue, "", "", true, true, testdata.SessionCookieOverrideSameSiteValue)
			},
		},
		{
			Name: "multiple response cookies",
			DSL:  testdata.MultipleResponseCookiesDSL,
			Check: func(t *testing.T, cookies []*expr.HTTPResponseCookieExpr) {
				t.Helper()
				if len(cookies) != 2 {
					t.Fatalf("got %d cookies, expected 2", len(cookies))
				}
				assertResponseCookie(t, cookies[0], "session", "__Host-ak_session", "/", "", "", true, true, expr.CookieSameSiteLax)
				assertResponseCookie(t, cookies[1], "refresh", "ak_refresh", "/tokens", "accounts.loom.design", "", false, false, "")
			},
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := expr.RunDSL(t, c.DSL)
			e := root.API.HTTP.Services[len(root.API.HTTP.Services)-1].HTTPEndpoints[0]
			c.Check(t, e.Responses[0].Cookies)
		})
	}
}

func TestSessionCookieInvalidPlacement(t *testing.T) {
	err := expr.RunInvalidDSL(t, testdata.InvalidSessionCookiePlacementDSL)
	if actual := err.Error(); actual == "" || !strings.Contains(actual, "invalid use of SessionCookie") {
		t.Fatalf("got error %q, expected it to contain %q", actual, "invalid use of SessionCookie")
	}
}

func TestResponseCookieValidation(t *testing.T) {
	t.Run("cookie setters require a declared response cookie", func(t *testing.T) {
		err := expr.RunInvalidDSL(t, testdata.InvalidCookieSetterPlacementDSL)
		if actual := err.Error(); actual == "" || !strings.Contains(actual, "cookie attributes must be declared after Cookie or SessionCookie in an HTTP response") {
			t.Fatalf("got error %q, expected it to contain %q", actual, "cookie attributes must be declared after Cookie or SessionCookie in an HTTP response")
		}
	})

	t.Run("duplicate cookie names are rejected", func(t *testing.T) {
		err := expr.RunInvalidDSL(t, testdata.DuplicateResponseCookieNameDSL)
		if actual := err.Error(); actual == "" || !strings.Contains(actual, `response defines duplicate cookie "ak"`) {
			t.Fatalf("got error %q, expected it to contain %q", actual, `response defines duplicate cookie "ak"`)
		}
	})

	t.Run("insecure setter requires a declared response cookie", func(t *testing.T) {
		err := expr.RunInvalidDSL(t, testdata.InvalidCookieInsecurePlacementDSL)
		if actual := err.Error(); actual == "" || !strings.Contains(actual, "cookie attributes must be declared after Cookie or SessionCookie in an HTTP response") {
			t.Fatalf("got error %q, expected it to contain %q", actual, "cookie attributes must be declared after Cookie or SessionCookie in an HTTP response")
		}
	})
}

func assertResponseCookie(
	t *testing.T,
	cookie *expr.HTTPResponseCookieExpr,
	attrName string,
	httpName string,
	path string,
	domain string,
	maxAge string,
	secure bool,
	httpOnly bool,
	sameSite expr.CookieSameSiteValue,
) {
	t.Helper()

	if actual := cookie.AttributeName(); actual != attrName {
		t.Errorf("got attribute name %q, expected %q", actual, attrName)
	}
	if actual := cookie.HTTPName(); actual != httpName {
		t.Errorf("got HTTP name %q, expected %q", actual, httpName)
	}
	if actual := cookie.Path; actual != path {
		t.Errorf("got path %q, expected %q", actual, path)
	}
	if actual := cookie.Domain; actual != domain {
		t.Errorf("got domain %q, expected %q", actual, domain)
	}
	if actual := cookie.MaxAge; actual != maxAge {
		t.Errorf("got max-age %q, expected %q", actual, maxAge)
	}
	if actual := cookie.Secure; actual != secure {
		t.Errorf("got secure %t, expected %t", actual, secure)
	}
	if actual := cookie.HTTPOnly; actual != httpOnly {
		t.Errorf("got httpOnly %t, expected %t", actual, httpOnly)
	}
	if actual := cookie.SameSite; actual != sameSite {
		t.Errorf("got same-site %q, expected %q", actual, sameSite)
	}
}
