package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

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
