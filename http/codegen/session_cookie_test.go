package codegen

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/publicsuffix"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testutil"
	"github.com/CaliLuke/loom/dsl"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
)

func TestSessionCookie(t *testing.T) {
	t.Run("server encode uses secure session cookie defaults", func(t *testing.T) {
		root := RunHTTPDSL(t, sessionCookieResponseDSL)
		services := CreateHTTPServices(root)

		serverFiles := ServerFiles("", services)
		require.Len(t, serverFiles, 2)
		serverEncode := codegen.SectionCode(t, serverFiles[1].AllSections()[1])
		require.Contains(t, serverEncode, `loomhttp.SetResponseCookie(ctx, w, &http.Cookie{`)
		require.Contains(t, serverEncode, `Secure:   true`)
		require.Contains(t, serverEncode, `HttpOnly: true`)
		testutil.NewGoldenFile(t, filepath.Join("testdata", "golden")).
			StringContent(serverEncode).
			Path("session_cookie_encode-defaults.golden").
			CompareContent()
	})

	t.Run("explicit cookie settings override session defaults", func(t *testing.T) {
		root := RunHTTPDSL(t, sessionCookieResponseOverrideDSL)
		services := CreateHTTPServices(root)

		serverFiles := ServerFiles("", services)
		require.Len(t, serverFiles, 2)
		serverEncode := codegen.SectionCode(t, serverFiles[1].AllSections()[1])
		require.Contains(t, serverEncode, `Path:     "/session"`)
		require.Contains(t, serverEncode, `SameSite: http.SameSiteStrictMode`)
		testutil.NewGoldenFile(t, filepath.Join("testdata", "golden")).
			StringContent(serverEncode).
			Path("session_cookie_encode-override.golden").
			CompareContent()
	})

	t.Run("insecure override omits secure from generated contracts", func(t *testing.T) {
		root := RunHTTPDSL(t, sessionCookieResponseInsecureDSL)
		services := CreateHTTPServices(root)

		serverFiles := ServerFiles("", services)
		require.Len(t, serverFiles, 2)
		serverEncode := codegen.SectionCode(t, serverFiles[1].AllSections()[1])
		require.NotContains(t, serverEncode, `Secure:`)
		require.Contains(t, serverEncode, `HttpOnly: true`)

		v3JSON := renderOpenAPIJSON(t, openapiv3.Files, root)
		doc := parseOpenAPIV3Document(t, v3JSON)
		pathItem, ok := doc.Paths.PathItems.Get("/session")
		require.True(t, ok)
		require.NotNil(t, pathItem.Post)
		okResp, ok := pathItem.Post.Responses.Codes.Get("201")
		require.True(t, ok)
		header, ok := okResp.Headers.Get("Set-Cookie")
		require.True(t, ok)
		require.NotNil(t, header)
		require.Contains(t, header.Description, `Policy: Path=/; HttpOnly; SameSite=Lax.`)
		require.NotContains(t, header.Description, "Secure")
		require.NotNil(t, header.Example)
		require.NotContains(t, header.Example.Value, "Secure")
	})

	t.Run("all explicit cookie settings override defaults", func(t *testing.T) {
		root := RunHTTPDSL(t, sessionCookieResponseOverrideAllDSL)
		services := CreateHTTPServices(root)

		serverFiles := ServerFiles("", services)
		require.Len(t, serverFiles, 2)
		serverEncode := codegen.SectionCode(t, serverFiles[1].AllSections()[1])
		require.Contains(t, serverEncode, `Domain:   "session.loom.design"`)
		require.Contains(t, serverEncode, `MaxAge:   7200`)
		testutil.NewGoldenFile(t, filepath.Join("testdata", "golden")).
			StringContent(serverEncode).
			Path("session_cookie_encode-override-all.golden").
			CompareContent()
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
		require.Equal(t, "session.loom.design", parsed[0].Domain)
		require.True(t, parsed[0].Secure)
		require.True(t, parsed[0].HttpOnly)
		require.Equal(t, http.SameSiteStrictMode, parsed[0].SameSite)
		require.Equal(t, 7200, parsed[0].MaxAge)

		jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
		require.NoError(t, err)

		origin, err := url.Parse("https://session.loom.design/session")
		require.NoError(t, err)
		jar.SetCookies(origin, parsed)

		stored := jar.Cookies(origin)
		require.Len(t, stored, 1)
		require.Equal(t, "__Host-ak_session", stored[0].Name)
		require.Equal(t, "session", stored[0].Value)

		httpOrigin, err := url.Parse("http://session.loom.design/session")
		require.NoError(t, err)
		require.Empty(t, jar.Cookies(httpOrigin))
	})

	t.Run("multiple response cookies keep independent policies", func(t *testing.T) {
		root := RunHTTPDSL(t, multiSessionCookieResponseDSL)
		services := CreateHTTPServices(root)

		serverFiles := ServerFiles("", services)
		require.Len(t, serverFiles, 2)
		serverEncode := codegen.SectionCode(t, serverFiles[1].AllSections()[1])
		require.Contains(t, serverEncode, `"__Host-ak_session"`)
		require.Contains(t, serverEncode, `"ak_refresh"`)
		testutil.NewGoldenFile(t, filepath.Join("testdata", "golden")).
			StringContent(serverEncode).
			Path("session_cookie_encode-multi.golden").
			CompareContent()

		endpoint := services.Get("multiSessionCookieResponse").Endpoint("create")
		require.NotNil(t, endpoint)
		require.Len(t, endpoint.Result.Responses, 1)
		require.Len(t, endpoint.Result.Responses[0].Cookies, 2)

		sessionCookie := cookieFromData(t, endpoint.Result.Responses[0].Cookies[0]).String()
		refreshCookie := cookieFromData(t, endpoint.Result.Responses[0].Cookies[1]).String()
		resp := &http.Response{
			Header: http.Header{
				"Set-Cookie": []string{sessionCookie, refreshCookie},
			},
		}
		parsed := resp.Cookies()
		require.Len(t, parsed, 2)
		require.Equal(t, "__Host-ak_session", parsed[0].Name)
		require.Equal(t, "/", parsed[0].Path)
		require.True(t, parsed[0].Secure)
		require.True(t, parsed[0].HttpOnly)
		require.Equal(t, "ak_refresh", parsed[1].Name)
		require.Equal(t, "/tokens", parsed[1].Path)
		require.Equal(t, "accounts.loom.design", parsed[1].Domain)
		require.False(t, parsed[1].Secure)
		require.False(t, parsed[1].HttpOnly)

		jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
		require.NoError(t, err)

		origin, err := url.Parse("https://accounts.loom.design/tokens")
		require.NoError(t, err)
		jar.SetCookies(origin, parsed)
		stored := jar.Cookies(origin)
		require.Len(t, stored, 2)
		storedByName := make(map[string]*http.Cookie, len(stored))
		for _, cookie := range stored {
			storedByName[cookie.Name] = cookie
		}
		require.Contains(t, storedByName, "__Host-ak_session")
		require.Contains(t, storedByName, "ak_refresh")
	})

	t.Run("openapi documents concrete response cookies", func(t *testing.T) {
		root := RunHTTPDSL(t, multiSessionCookieResponseDSL)
		v3JSON := renderOpenAPIJSON(t, openapiv3.Files, root)
		doc := parseOpenAPIV3Document(t, v3JSON)

		pathItem, ok := doc.Paths.PathItems.Get("/session")
		require.True(t, ok)
		require.NotNil(t, pathItem.Post)
		okResp, ok := pathItem.Post.Responses.Codes.Get("201")
		require.True(t, ok)
		require.NotNil(t, okResp)
		header, ok := okResp.Headers.Get("Set-Cookie")
		require.True(t, ok)
		require.NotNil(t, header)
		require.Contains(t, header.Description, "__Host-ak_session")
		require.Contains(t, header.Description, "ak_refresh")
		require.NotNil(t, header.Schema)
		require.NotNil(t, header.Examples)
		require.Equal(t, 2, header.Examples.Len())
	})

	t.Run("openapi clear-cookie docs stay consistent with serialized example", func(t *testing.T) {
		root := RunHTTPDSL(t, sessionCookieClearDSL)
		v3JSON := renderOpenAPIJSON(t, openapiv3.Files, root)
		doc := parseOpenAPIV3Document(t, v3JSON)

		pathItem, ok := doc.Paths.PathItems.Get("/session/logout")
		require.True(t, ok)
		require.NotNil(t, pathItem.Post)
		noContentResp, ok := pathItem.Post.Responses.Codes.Get("204")
		require.True(t, ok)
		require.NotNil(t, noContentResp)
		header, ok := noContentResp.Headers.Get("Set-Cookie")
		require.True(t, ok)
		require.NotNil(t, header)
		require.Contains(t, header.Description, `Sets the "__Host-ak_session" cookie.`)
		require.Contains(t, header.Description, `Policy: Path=/; Max-Age=0; Secure; HttpOnly; SameSite=Lax.`)
		require.NotNil(t, header.Example)
		example := header.Example.Value
		cookies := (&http.Response{Header: http.Header{"Set-Cookie": []string{example}}}).Cookies()
		require.Len(t, cookies, 1)
		require.Equal(t, "__Host-ak_session", cookies[0].Name)
		require.NotEmpty(t, cookies[0].Value)
		require.Equal(t, "/", cookies[0].Path)
		require.True(t, cookies[0].HttpOnly)
		require.True(t, cookies[0].Secure)
		require.Equal(t, http.SameSiteLaxMode, cookies[0].SameSite)
		require.Contains(t, example, "; Max-Age=0;")
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

var sessionCookieResponseInsecureDSL = func() {
	dsl.Service("sessionCookieResponseInsecure", func() {
		dsl.Method("create", func() {
			dsl.Result(func() {
				dsl.Attribute("session", dsl.String)
			})
			dsl.HTTP(func() {
				dsl.POST("/session")
				dsl.Response(dsl.StatusCreated, func() {
					dsl.SessionCookie("session:app_session")
					dsl.CookieInsecure()
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
					dsl.CookieDomain("session.loom.design")
					dsl.CookieMaxAge(7200)
					dsl.CookieSameSite(dsl.CookieSameSiteStrict)
				})
			})
		})
	})
}

var multiSessionCookieResponseDSL = func() {
	dsl.Service("multiSessionCookieResponse", func() {
		dsl.Method("create", func() {
			dsl.Result(func() {
				dsl.Attribute("session", dsl.String)
				dsl.Attribute("refresh", dsl.String)
			})
			dsl.HTTP(func() {
				dsl.POST("/session")
				dsl.Response(dsl.StatusCreated, func() {
					dsl.SessionCookie("session:__Host-ak_session")
					dsl.Cookie("refresh:ak_refresh")
					dsl.CookiePath("/tokens")
					dsl.CookieDomain("accounts.loom.design")
				})
			})
		})
	})
}

var sessionCookieClearDSL = func() {
	dsl.Service("sessionCookieClear", func() {
		dsl.Method("logout", func() {
			dsl.Result(func() {
				dsl.Attribute("session", dsl.String)
			})
			dsl.HTTP(func() {
				dsl.POST("/session/logout")
				dsl.Response(dsl.StatusNoContent, func() {
					dsl.SessionCookie("session:__Host-ak_session")
					dsl.CookieMaxAge(-1)
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
