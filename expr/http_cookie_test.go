package expr_test

import (
	"fmt"
	"strings"
	"testing"

	"goa.design/goa/v3/expr"
	"goa.design/goa/v3/expr/testdata"
)

func TestHTTPResponseCookie(t *testing.T) {
	type Props map[string]any

	cases := []struct {
		Name  string
		DSL   func()
		Props Props
	}{
		{"cookie", testdata.CookieObjectResultDSL, nil},
		{"cookie", testdata.CookieStringResultDSL, nil},
		{"max-age", testdata.CookieMaxAgeDSL, Props{"cookie:max-age": testdata.CookieMaxAgeValue}},
		{"domain", testdata.CookieDomainDSL, Props{"cookie:domain": testdata.CookieDomainValue}},
		{"path", testdata.CookiePathDSL, Props{"cookie:path": testdata.CookiePathValue}},
		{"secure", testdata.CookieSecureDSL, Props{"cookie:secure": "Secure"}},
		{"http-only", testdata.CookieHTTPOnlyDSL, Props{"cookie:http-only": "HttpOnly"}},
		{"same-site", testdata.CookieSameSiteDSL, Props{"cookie:same-site": testdata.CookieSameSiteValue}},
		{"session-cookie-defaults", testdata.SessionCookieDefaultsDSL, Props{
			"cookie:path":      "/",
			"cookie:secure":    "Secure",
			"cookie:http-only": "HttpOnly",
			"cookie:same-site": expr.CookieSameSiteLax,
		}},
		{"session-cookie-overrides", testdata.SessionCookieOverrideDSL, Props{
			"cookie:path":      testdata.SessionCookieOverridePathValue,
			"cookie:secure":    "Secure",
			"cookie:http-only": "HttpOnly",
			"cookie:same-site": testdata.SessionCookieOverrideSameSiteValue,
		}},
		{"session-cookie-overrides-all", testdata.SessionCookieOverrideAllDSL, Props{
			"cookie:path":      testdata.SessionCookieOverridePathValue,
			"cookie:domain":    testdata.SessionCookieOverrideDomainValue,
			"cookie:max-age":   testdata.SessionCookieOverrideMaxAgeValue,
			"cookie:secure":    "Secure",
			"cookie:http-only": "HttpOnly",
			"cookie:same-site": testdata.SessionCookieOverrideSameSiteValue,
		}},
		{"session-cookie-repeated-setter", testdata.SessionCookieRepeatedSetterDSL, Props{
			"cookie:path":      testdata.SessionCookieOverridePathValue,
			"cookie:secure":    "Secure",
			"cookie:http-only": "HttpOnly",
			"cookie:same-site": testdata.SessionCookieOverrideSameSiteValue,
		}},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := expr.RunDSL(t, c.DSL)
			e := root.API.HTTP.Services[len(root.API.HTTP.Services)-1].HTTPEndpoints[0]
			cookies := e.Responses[0].Cookies.AttributeExpr
			if len(*expr.AsObject(cookies.Type)) != 1 {
				t.Errorf("got %d cookie(s), expected exactly one", len(*expr.AsObject(cookies.Type)))
			} else {
				m := cookies.Meta
				if len(c.Props) == 0 {
					if len(m) != 0 {
						t.Errorf("got cookies metadata with length %d, expected 0", len(m))
					}
					return
				}
				if len(m) != len(c.Props) {
					t.Errorf("got cookies metadata with length %d, expected %d", len(m), len(c.Props))
				}
				for n, v := range c.Props {
					switch {
					case len(m[n]) != 1:
						t.Errorf("got cookies metadata %q with length %d, expected 1", n, len(m[n]))
					case m[n][0] != fmt.Sprintf("%v", v):
						t.Errorf("got value %q for cookies metadata %q, expected %q", m[n][0], n, fmt.Sprintf("%v", v))
					}
				}
			}
		})
	}
}

func TestSessionCookieInvalidPlacement(t *testing.T) {
	err := expr.RunInvalidDSL(t, testdata.InvalidSessionCookiePlacementDSL)
	if actual := err.Error(); actual == "" || !strings.Contains(actual, "invalid use of SessionCookie") {
		t.Fatalf("got error %q, expected it to contain %q", actual, "invalid use of SessionCookie")
	}
}
