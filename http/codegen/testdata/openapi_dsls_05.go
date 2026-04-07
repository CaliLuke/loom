package testdata

import . "github.com/CaliLuke/loom/dsl"


var OpenAPIRequestResponseSplitDSL = func() {
	var Account = Type("Account", func() {
		Attribute("id", String, func() {
			Meta("openapi:readOnly", "true")
			Example("acct_123")
		})
		Attribute("email", String, func() {
			Format(FormatEmail)
			Example("user@example.test")
		})
		Attribute("password", String, func() {
			Meta("openapi:writeOnly", "true")
			Example("hunter2")
		})
		Required("id", "email", "password")
	})

	Service("accounts", func() {
		Method("create", func() {
			Payload(Account)
			Result(Account)
			HTTP(func() {
				POST("/accounts")
				Response(StatusCreated)
			})
		})
	})
}


