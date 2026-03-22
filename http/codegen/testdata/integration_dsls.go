package testdata

import (
	"fmt"

	. "github.com/CaliLuke/loom/dsl"
)

// LargeErrorSetHTTPClientDSL defines an HTTP service with a large declared
// error set to stress shared client generation inside a full temp-module loop.
var LargeErrorSetHTTPClientDSL = func() {
	var Account = Type("Account", func() {
		Attribute("id", String)
		Attribute("name", String)
		Required("id", "name")
	})
	var AdminAuthenticatedRequest = Type("AdminAuthenticatedRequest", func() {
		Attribute("project_id", String)
		Attribute("account_id", String)
		Required("project_id", "account_id")
	})

	Service("rest_admin", func() {
		Method("ListAccounts", func() {
			Payload(AdminAuthenticatedRequest)
			Result(ArrayOf(Account))
			for i := 0; i < 60; i++ {
				Error(fmt.Sprintf("error_%02d", i))
			}
			HTTP(func() {
				GET("/accounts/{project_id}/{account_id}")
				Response(StatusOK)
			})
		})
	})
}
