package design

import . "github.com/CaliLuke/loom/dsl"

var _ = API("quality", func() {
	Title("gRPC Generated Code Quality Fixture")
})

var Account = ResultType("application/vnd.loom.grpc.quality.account", func() {
	Field(1, "id", String)
	Field(2, "email", String, func() {
		Format(FormatEmail)
	})
	Field(3, "display_name", String)
	Field(4, "revision", Int64)
	Required("id", "email", "display_name", "revision")

	View("default", func() {
		Attribute("id")
		Attribute("email")
		Attribute("display_name")
	})
	View("extended", func() {
		Attribute("id")
		Attribute("email")
		Attribute("display_name")
		Attribute("revision")
	})
})

var AccountRequest = Type("AccountRequest", func() {
	Field(1, "account_id", String)
	Field(2, "include_inactive", Boolean)
	Field(3, "request_id", String)
	Field(4, "trace_level", Int, func() {
		Minimum(0)
		Maximum(5)
	})
	Required("account_id", "request_id")
})

var AccountUpdate = Type("AccountUpdate", func() {
	Field(1, "account_id", String)
	Field(2, "display_name", String, func() {
		MinLength(1)
	})
	Required("account_id", "display_name")
})

var AccountEvent = Type("AccountEvent", func() {
	Field(1, "id", String)
	Field(2, "kind", String, func() {
		Enum("created", "updated", "deleted")
	})
	Field(3, "revision", Int64)
	Required("id", "kind", "revision")
})

var QualityError = Type("QualityError", func() {
	Field(1, "message", String)
	Required("message")
})

var _ = Service("accounts", func() {
	Method("show", func() {
		Payload(AccountRequest)
		Result(Account)
		Error("not_found", QualityError)
		GRPC(func() {
			Metadata(func() {
				Attribute("request_id:X-Request-ID")
				Attribute("trace_level:X-Trace-Level")
			})
			Response(CodeOK, func() {
				Headers(func() {
					Attribute("revision:X-Account-Revision")
				})
			})
			Response(CodeNotFound, "not_found")
		})
	})

	Method("watch", func() {
		Payload(func() {
			Field(1, "account_id", String)
			Field(2, "request_id", String)
			Required("account_id", "request_id")
		})
		StreamingResult(Account, func() {
			View("extended")
		})
		GRPC(func() {
			Metadata(func() {
				Attribute("request_id:X-Request-ID")
			})
			Response(CodeOK)
		})
	})

	Method("sync", func() {
		Payload(func() {
			Field(1, "request_id", String)
			Required("request_id")
		})
		StreamingPayload(AccountUpdate)
		StreamingResult(AccountEvent)
		Error("invalid_update", QualityError)
		GRPC(func() {
			Metadata(func() {
				Attribute("request_id:X-Request-ID")
			})
			Response(CodeOK)
			Response(CodeInvalidArgument, "invalid_update")
		})
	})
})
