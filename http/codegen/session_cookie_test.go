package codegen

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/publicsuffix"

	"goa.design/goa/v3/codegen"
	"goa.design/goa/v3/dsl"
)

func TestSessionCookie(t *testing.T) {
	t.Run("server encode uses secure session cookie defaults", func(t *testing.T) {
		root := RunHTTPDSL(t, sessionCookieResponseDSL)
		services := CreateHTTPServices(root)

		serverFiles := ServerFiles("", services)
		require.Len(t, serverFiles, 2)
		serverEncode := codegen.SectionCode(t, serverFiles[1].SectionTemplates[1])
		require.Contains(t, serverEncode, `Path:     "/"`)
		require.Contains(t, serverEncode, `Secure:   true`)
		require.Contains(t, serverEncode, `HttpOnly: true`)
		require.Contains(t, serverEncode, `SameSite: http.SameSiteLaxMode`)
	})

	t.Run("explicit cookie settings override session defaults", func(t *testing.T) {
		root := RunHTTPDSL(t, sessionCookieResponseOverrideDSL)
		services := CreateHTTPServices(root)

		serverFiles := ServerFiles("", services)
		require.Len(t, serverFiles, 2)
		serverEncode := codegen.SectionCode(t, serverFiles[1].SectionTemplates[1])
		require.Contains(t, serverEncode, `Path:     "/session"`)
		require.Contains(t, serverEncode, `SameSite: http.SameSiteStrictMode`)
		require.Contains(t, serverEncode, `Secure:   true`)
		require.Contains(t, serverEncode, `HttpOnly: true`)
	})

	t.Run("all explicit cookie settings override defaults", func(t *testing.T) {
		root := RunHTTPDSL(t, sessionCookieResponseOverrideAllDSL)
		services := CreateHTTPServices(root)

		serverFiles := ServerFiles("", services)
		require.Len(t, serverFiles, 2)
		serverEncode := codegen.SectionCode(t, serverFiles[1].SectionTemplates[1])
		require.Contains(t, serverEncode, `Path:     "/session"`)
		require.Contains(t, serverEncode, `Domain:   "session.goa.design"`)
		require.Contains(t, serverEncode, `MaxAge:   7200`)
		require.Contains(t, serverEncode, `SameSite: http.SameSiteStrictMode`)
	})

	t.Run("set-cookie round trip survives parser and cookie jar", func(t *testing.T) {
		root := RunHTTPDSL(t, sessionCookieResponseOverrideAllDSL)
		services := CreateHTTPServices(root)
		endpoint := services.Get("sessionCookieResponseOverrideAll").Endpoint("create")
		require.NotNil(t, endpoint)
		require.NotNil(t, endpoint.Result)
		require.Len(t, endpoint.Result.Responses, 1)
		require.Len(t, endpoint.Result.Responses[0].Cookies, 1)

		encoded := cookieFromData(t, endpoint.Result.Responses[0].Cookies[0]).String()
		resp := &http.Response{
			Header: http.Header{
				"Set-Cookie": []string{encoded},
			},
		}
		parsed := resp.Cookies()
		require.Len(t, parsed, 1)
		require.Equal(t, "__Host-ak_session", parsed[0].Name)
		require.Equal(t, "/session", parsed[0].Path)
		require.Equal(t, "session.goa.design", parsed[0].Domain)
		require.True(t, parsed[0].Secure)
		require.True(t, parsed[0].HttpOnly)
		require.Equal(t, http.SameSiteStrictMode, parsed[0].SameSite)
		require.Equal(t, 7200, parsed[0].MaxAge)

		jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
		require.NoError(t, err)

		origin, err := url.Parse("https://session.goa.design/session")
		require.NoError(t, err)
		jar.SetCookies(origin, parsed)

		stored := jar.Cookies(origin)
		require.Len(t, stored, 1)
		require.Equal(t, "__Host-ak_session", stored[0].Name)
		require.Equal(t, "session", stored[0].Value)

		httpOrigin, err := url.Parse("http://session.goa.design/session")
		require.NoError(t, err)
		require.Empty(t, jar.Cookies(httpOrigin))
	})
}

var sessionCookieResponseDSL = func() {
	dsl.Service("sessionCookieResponse", func() {
		dsl.Method("create", func() {
			dsl.Result(func() {
				dsl.Attribute("session", dsl.String)
			})
			dsl.HTTP(func() {
				dsl.POST("/session")
				dsl.Response(dsl.StatusCreated, func() {
					dsl.SessionCookie("session:__Host-ak_session")
				})
			})
		})
	})
}

var sessionCookieResponseOverrideDSL = func() {
	dsl.Service("sessionCookieResponseOverride", func() {
		dsl.Method("create", func() {
			dsl.Result(func() {
				dsl.Attribute("session", dsl.String)
			})
			dsl.HTTP(func() {
				dsl.POST("/session")
				dsl.Response(dsl.StatusCreated, func() {
					dsl.SessionCookie("session:__Host-ak_session")
					dsl.CookiePath("/session")
					dsl.CookieSameSite(dsl.CookieSameSiteStrict)
				})
			})
		})
	})
}

var sessionCookieResponseOverrideAllDSL = func() {
	dsl.Service("sessionCookieResponseOverrideAll", func() {
		dsl.Method("create", func() {
			dsl.Result(func() {
				dsl.Attribute("session", dsl.String)
			})
			dsl.HTTP(func() {
				dsl.POST("/session")
				dsl.Response(dsl.StatusCreated, func() {
					dsl.SessionCookie("session:__Host-ak_session")
					dsl.CookiePath("/session")
					dsl.CookieDomain("session.goa.design")
					dsl.CookieMaxAge(7200)
					dsl.CookieSameSite(dsl.CookieSameSiteStrict)
				})
			})
		})
	})
}

func cookieFromData(t *testing.T, data *CookieData) *http.Cookie {
	t.Helper()

	cookie := &http.Cookie{
		Name:     data.HTTPName,
		Value:    "session",
		Path:     data.Path,
		Domain:   data.Domain,
		Secure:   data.Secure,
		HttpOnly: data.HTTPOnly,
	}
	if data.MaxAge != "" {
		maxAge, err := strconv.Atoi(data.MaxAge)
		require.NoError(t, err)
		cookie.MaxAge = maxAge
	}
	switch data.SameSite {
	case "http.SameSiteLaxMode":
		cookie.SameSite = http.SameSiteLaxMode
	case "http.SameSiteStrictMode":
		cookie.SameSite = http.SameSiteStrictMode
	case "http.SameSiteNoneMode":
		cookie.SameSite = http.SameSiteNoneMode
	case "http.SameSiteDefaultMode":
		cookie.SameSite = http.SameSiteDefaultMode
	}
	return cookie
}
