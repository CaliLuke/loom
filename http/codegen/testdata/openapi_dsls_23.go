package testdata

import . "github.com/CaliLuke/loom/dsl"


var AsyncSessionSecurityDSL = func() {
	var browserSession = APIKeySecurity("browser_session_cookie", func() {
		Description("Browser session cookie used by first-party async clients.")
	})
	var appSession = SessionAuth("async_session", func() {
		CookieTransport(browserSession, "", func() {
			CookieName("__Host-ak_session")
		})
	})
	var RealtimeEnvelope = Type("AsyncSessionRealtimeEnvelope", func() {
		Attribute("event", String, func() {
			Example("project.updated")
		})
		Attribute("project_id", String, func() {
			Example("proj_42")
		})
		Required("event", "project_id")
	})
	var RealtimeSSEEvent = Type("AsyncSessionRealtimeSSEEvent", func() {
		Attribute("id", String, func() {
			Example("evt_9")
		})
		Attribute("event", String, func() {
			Example("project.updated")
		})
		Attribute("project_id", String, func() {
			Example("proj_42")
		})
		Required("id", "event", "project_id")
	})
	var _ = API("async-session-security", func() {
		Title("Async Session Security API")
		Description("Exercises async streaming contracts secured by cookie-backed session auth.")
		Meta("openapi:closed-objects", "true")
		Server("async-session-security", func() {
			Host("api", func() {
				URI("https://api.async-session-security.example.test")
			})
		})
	})
	Service("asyncSessionSecurity", func() {
		Description("Async routes protected by the browser session cookie.")
		SessionSecurity(appSession)

		Method("projectSocket", func() {
			Payload(func() {
				Attribute("project_id", String, func() {
					Example("proj_42")
				})
				Required("project_id")
			})
			StreamingPayload(String)
			StreamingResult(RealtimeEnvelope)
			HTTP(func() {
				GET("/ws/projects/{project_id}")
				Param("project_id")
				Response(StatusOK)
			})
		})

		Method("events", func() {
			Payload(func() {
				Attribute("project_id", String, func() {
					Example("proj_42")
				})
				Attribute("last_event_id", String, func() {
					Example("evt_8")
				})
				Required("project_id")
			})
			StreamingResult(RealtimeSSEEvent)
			HTTP(func() {
				GET("/events/{project_id}")
				Param("project_id")
				Param("last_event_id")
				ServerSentEvents(func() {
					SSERequestID("last_event_id")
					SSEEventID("id")
				})
				Response(StatusOK)
			})
		})
	})
}

