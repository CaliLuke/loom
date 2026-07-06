package design

import . "github.com/CaliLuke/loom/dsl"

var _ = API("quality", func() {
	Title("HTTP Generated Code Quality Fixture")
})

var Account = ResultType("application/vnd.loom.quality.account", func() {
	Attribute("id", String)
	Attribute("email", String, func() {
		Format(FormatEmail)
	})
	Attribute("display_name", String)
	Required("id", "email", "display_name")

	View("default", func() {
		Attribute("id")
		Attribute("email")
		Attribute("display_name")
	})
	View("extended", func() {
		Attribute("id")
		Attribute("email")
		Attribute("display_name")
	})
})

var AccountSession = Type("AccountSession", func() {
	Attribute("id", String)
	Attribute("etag", String)
	Attribute("session", String)
	Attribute("refresh", String)
	Required("id", "etag", "session", "refresh")
})

var AccountCreatePayload = Type("AccountCreatePayload", func() {
	Attribute("email", String, func() {
		Format(FormatEmail)
	})
	Attribute("display_name", String, func() {
		MinLength(1)
	})
	Required("email", "display_name")
})

var NotFound = Type("NotFound", func() {
	Attribute("message", String)
	Attribute("trace_id", String)
	Required("message", "trace_id")
})

var bearer = JWTSecurity("session_bearer", func() {
	Description("Application bearer token")
})

var browserSessionCookie = APIKeySecurity("browser_session_cookie", func() {
	Description("Browser session cookie")
})

var appSession = SessionAuth("app_session", func() {
	BearerTransport(bearer, "auth")
	CookieTransport(browserSessionCookie, "browser_session", func() {
		CookieName("__Host-ak_session")
	})
})

var _ = Service("accounts", func() {
	Method("show", func() {
		SessionSecurity(appSession)
		Payload(func() {
			Attribute("project_id", String)
			Attribute("account_id", String)
			Attribute("include_inactive", Boolean)
			Attribute("request_id", String)
			Attribute("locale", String, func() {
				Default("en-US")
			})
			Required("project_id", "account_id", "request_id")
		})
		Result(Account)
		Error("not_found", NotFound)
		HTTP(func() {
			GET("/projects/{project_id}/accounts/{account_id}")
			Param("project_id")
			Param("account_id")
			Param("include_inactive")
			Header("request_id:X-Request-ID")
			Cookie("locale:locale")
			Response(StatusOK)
			Response("not_found", StatusNotFound, func() {
				Header("trace_id:X-Trace-ID")
			})
		})
	})

	Method("create", func() {
		SessionSecurity(appSession)
		Payload(AccountCreatePayload)
		Result(AccountSession)
		HTTP(func() {
			POST("/accounts")
			Body(AccountCreatePayload)
			Response(StatusCreated, func() {
				Header("etag:ETag")
				SessionCookie("session:__Host-ak_session")
				Cookie("refresh:ak_refresh")
				CookiePath("/tokens")
				CookieDomain("accounts.loom.design")
			})
		})
	})
})
