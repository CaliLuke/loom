package dsl

// Username defines the attribute used to provide the username to an endpoint
// secured with basic authentication. The parameters and usage of Username are
// the same as the Loom DSL Attribute function.
//
// The generated code produced by Loom uses the value of the corresponding
// payload field to compute the basic authentication Authorization header value.
//
// Username must appear in Payload or Type.
//
// Example:
//
//	Method("login", func() {
//	    Security(Basic)
//	    Payload(func() {
//	        Username("user", String)
//	        Password("pass", String)
//	    })
//	    HTTP(func() {
//	        // The "Authorization" header is defined implicitly.
//	        POST("/login")
//	    })
//	})
func Username(name string, args ...any) {
	args = useDSL(args, func() { Meta("security:username") })
	Attribute(name, args...)
}

// UsernameField is syntactic sugar to define a username attribute with the
// "rpc:tag" meta set with the value of the first argument.
//
// UsernameField takes the same arguments as Username with the addition of the
// tag value as the first argument.
func UsernameField(tag any, name string, args ...any) {
	args = useDSL(args, func() { Meta("security:username") })
	Field(tag, name, args...)
}

// Password defines the attribute used to provide the password to an endpoint
// secured with basic authentication. The parameters and usage of Password are
// the same as the Loom DSL Attribute function.
//
// The generated code produced by Loom uses the value of the corresponding
// payload field to compute the basic authentication Authorization header value.
//
// Password must appear in Payload or Type.
//
// Example:
//
//	Method("login", func() {
//	    Security(Basic)
//	    Payload(func() {
//	        Username("user", String)
//	        Password("pass", String)
//	    })
//	    HTTP(func() {
//	        // The "Authorization" header is defined implicitly.
//	        POST("/login")
//	    })
//	})
func Password(name string, args ...any) {
	args = useDSL(args, func() { Meta("security:password") })
	Attribute(name, args...)
}

// PasswordField is syntactic sugar to define a password attribute with the
// "rpc:tag" meta set with the value of the first argument.
//
// PasswordField takes the same arguments as Password with the addition of the
// tag value as the first argument.
func PasswordField(tag any, name string, args ...any) {
	args = useDSL(args, func() { Meta("security:password") })
	Field(tag, name, args...)
}

// })
func APIKey(scheme, name string, args ...any) {
	args = useDSL(args, func() { Meta("security:apikey:"+scheme, scheme) })
	Attribute(name, args...)
}

// APIKeyField is syntactic sugar to define an API key attribute with the
// "rpc:tag" meta set with the value of the first argument.
//
// APIKeyField takes the same arguments as APIKey with the addition of the
// tag value as the first argument.
func APIKeyField(tag any, scheme, name string, args ...any) {
	args = useDSL(args, func() { Meta("security:apikey:"+scheme, scheme) })
	Field(tag, name, args...)
}

// AccessToken defines the attribute used to provide the access token to an
// endpoint secured with OAuth2. The parameters and usage of AccessToken are the
// same as the Loom DSL Attribute function.
//
// The generated code produced by Loom uses the value of the corresponding
// payload field to initialize the Authorization header.
//
// AccessToken must appear in Payload or Type.
//
// Example:
//
//	Method("secured", func() {
//	    Security(OAuth2)
//	    Payload(func() {
//	        AccessToken("token", String, "OAuth2 access token used to perform authorization")
//	        Required("token")
//	    })
//	    Result(String)
//	    HTTP(func() {
//	        // The "Authorization" header is defined implicitly.
//	        GET("/")
//	    })
//	})
func AccessToken(name string, args ...any) {
	args = useDSL(args, func() { Meta("security:accesstoken") })
	Attribute(name, args...)
}

// AccessTokenField is syntactic sugar to define an access token attribute with the
// "rpc:tag" meta set with the value of the first argument.
//
// AccessTokenField takes the same arguments as AccessToken with the addition of the
// tag value as the first argument.
func AccessTokenField(tag any, name string, args ...any) {
	args = useDSL(args, func() { Meta("security:accesstoken") })
	Field(tag, name, args...)
}

// Token defines the attribute used to provide the JWT to an endpoint secured
// via JWT. The parameters and usage of Token are the same as the Loom DSL
// Attribute function.
//
// The generated code produced by Loom uses the value of the corresponding
// payload field to initialize the Authorization header.
//
// Example:
//
//	Method("secured", func() {
//	    Security(JWT)
//	    Payload(func() {
//	        Token("token", String, "JWT token used to perform authorization")
//	        Required("token")
//	    })
//	    Result(String)
//	    HTTP(func() {
//	        // The "Authorization" header is defined implicitly.
//	        GET("/")
//	    })
//	})
func Token(name string, args ...any) {
	args = useDSL(args, func() { Meta("security:token") })
	Attribute(name, args...)
}

// TokenField is syntactic sugar to define a JWT token attribute with the
// "rpc:tag" meta set with the value of the first argument.
//
// TokenField takes the same arguments as Token with the addition of the
// tag value as the first argument.
func TokenField(tag any, name string, args ...any) {
	args = useDSL(args, func() { Meta("security:token") })
	Field(tag, name, args...)
}
