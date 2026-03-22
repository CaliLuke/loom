package dsl

import (
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

const (
	StatusContinue           = expr.StatusContinue
	StatusSwitchingProtocols = expr.StatusSwitchingProtocols
	StatusProcessing         = expr.StatusProcessing

	StatusOK                   = expr.StatusOK
	StatusCreated              = expr.StatusCreated
	StatusAccepted             = expr.StatusAccepted
	StatusNonAuthoritativeInfo = expr.StatusNonAuthoritativeInfo
	StatusNoContent            = expr.StatusNoContent
	StatusResetContent         = expr.StatusResetContent
	StatusPartialContent       = expr.StatusPartialContent
	StatusMultiStatus          = expr.StatusMultiStatus
	StatusAlreadyReported      = expr.StatusAlreadyReported
	StatusIMUsed               = expr.StatusIMUsed

	StatusMultipleChoices  = expr.StatusMultipleChoices
	StatusMovedPermanently = expr.StatusMovedPermanently
	StatusFound            = expr.StatusFound
	StatusSeeOther         = expr.StatusSeeOther
	StatusNotModified      = expr.StatusNotModified
	StatusUseProxy         = expr.StatusUseProxy

	StatusTemporaryRedirect = expr.StatusTemporaryRedirect
	StatusPermanentRedirect = expr.StatusPermanentRedirect

	StatusBadRequest                   = expr.StatusBadRequest
	StatusUnauthorized                 = expr.StatusUnauthorized
	StatusPaymentRequired              = expr.StatusPaymentRequired
	StatusForbidden                    = expr.StatusForbidden
	StatusNotFound                     = expr.StatusNotFound
	StatusMethodNotAllowed             = expr.StatusMethodNotAllowed
	StatusNotAcceptable                = expr.StatusNotAcceptable
	StatusProxyAuthRequired            = expr.StatusProxyAuthRequired
	StatusRequestTimeout               = expr.StatusRequestTimeout
	StatusConflict                     = expr.StatusConflict
	StatusGone                         = expr.StatusGone
	StatusLengthRequired               = expr.StatusLengthRequired
	StatusPreconditionFailed           = expr.StatusPreconditionFailed
	StatusRequestEntityTooLarge        = expr.StatusRequestEntityTooLarge
	StatusRequestURITooLong            = expr.StatusRequestURITooLong
	StatusUnsupportedMediaType         = expr.StatusUnsupportedMediaType
	StatusRequestedRangeNotSatisfiable = expr.StatusRequestedRangeNotSatisfiable
	StatusExpectationFailed            = expr.StatusExpectationFailed
	StatusTeapot                       = expr.StatusTeapot
	StatusUnprocessableEntity          = expr.StatusUnprocessableEntity
	StatusLocked                       = expr.StatusLocked
	StatusFailedDependency             = expr.StatusFailedDependency
	StatusUpgradeRequired              = expr.StatusUpgradeRequired
	StatusPreconditionRequired         = expr.StatusPreconditionRequired
	StatusTooManyRequests              = expr.StatusTooManyRequests
	StatusRequestHeaderFieldsTooLarge  = expr.StatusRequestHeaderFieldsTooLarge
	StatusUnavailableForLegalReasons   = expr.StatusUnavailableForLegalReasons

	StatusInternalServerError           = expr.StatusInternalServerError
	StatusNotImplemented                = expr.StatusNotImplemented
	StatusBadGateway                    = expr.StatusBadGateway
	StatusServiceUnavailable            = expr.StatusServiceUnavailable
	StatusGatewayTimeout                = expr.StatusGatewayTimeout
	StatusHTTPVersionNotSupported       = expr.StatusHTTPVersionNotSupported
	StatusVariantAlsoNegotiates         = expr.StatusVariantAlsoNegotiates
	StatusInsufficientStorage           = expr.StatusInsufficientStorage
	StatusLoopDetected                  = expr.StatusLoopDetected
	StatusNotExtended                   = expr.StatusNotExtended
	StatusNetworkAuthenticationRequired = expr.StatusNetworkAuthenticationRequired
)

const (
	CookieSameSiteStrict  = expr.CookieSameSiteStrict
	CookieSameSiteLax     = expr.CookieSameSiteLax
	CookieSameSiteNone    = expr.CookieSameSiteNone
	CookieSameSiteDefault = expr.CookieSameSiteDefault
)

// HTTP defines the HTTP transport specific properties of an API, a service or a
// single method. The function maps the method payload and result types to HTTP
// properties such as parameters (via path wildcards or query strings), request
// or response headers, request or response bodies as well as response status
// code. HTTP also defines HTTP specific properties such as the method endpoint
// URLs and HTTP methods.
//
// The functions that appear in HTTP such as Header, Param or Body may take
// advantage of the method payload or result types (depending on whether they
// appear when describing the HTTP request or response). The properties of the
// header, parameter or body attributes inherit the properties of the attributes
// with the same names that appear in the method payload or result types.
//
// HTTP must appear in an API, a Service, or a Method expression.
//
// HTTP accepts an optional argument which is the defining DSL function.
//
// Example:
//
//	var _ = API("calc", func() {
//	    HTTP(func() {
//	        Path("/api") // Prefix to HTTP path of all requests.
//	    })
//	})
//
// Example:
//
//	var _ = Service("calculator", func() {
//	    Error("unauthorized")
//
//	    HTTP(func() {
//	        Path("/calc")      // Prefix to all request paths
//	        Error("unauthorized", StatusUnauthorized) // Define "unauthorized"
//	                           // error HTTP response status code.
//	        Parent("account")  // Parent service, used to prefix request
//	                           // paths.
//	        CanonicalMethod("show") // Method whose path is used to prefix
//	                                // the paths of child service.
//	    })
//
//	    Method("div", func() {
//	        Description("Divide two operands.")
//	        Payload(Operands)
//	        Error("div_by_zero")
//
//	        HTTP(func() {
//	            GET("/div/{left}/{right}") // Define HTTP route. The "left"
//	                                       // and "right" parameter properties
//	                                       // are inherited from the
//	                                       // corresponding Operands attributes.
//	            Param("integer:int")       // Load "integer" attribute of
//	                                       // Operands from "int" query string.
//	            Header("requestID:X-RequestId")  // Load "requestID" attribute
//	                                             // of Operands from
//	                                             // X-RequestId header
//	            Response(StatusOK)               // Use status 200 on success
//	            Error("div_by_zero", BadRequest) // Use status code 400 for
//	                                             // "div_by_zero" responses
//	        })
//	    })
//	})
func HTTP(fns ...func()) {
	if len(fns) > 1 {
		eval.TooManyArgError()
		return
	}
	fn := func() {}
	if len(fns) == 1 {
		fn = fns[0]
	}
	switch actual := eval.Current().(type) {
	case *expr.APIExpr:
		eval.Execute(fn, expr.Root)
	case *expr.ServiceExpr:
		res := expr.Root.API.HTTP.ServiceFor(actual, expr.Root.API.HTTP)
		res.DSLFunc = fn
	case *expr.MethodExpr:
		res := expr.Root.API.HTTP.ServiceFor(actual.Service, expr.Root.API.HTTP)
		act := res.EndpointFor(actual)
		act.DSLFunc = fn
	default:
		eval.IncompatibleDSL()
	}
}

// Consumes adds a MIME type to the list of MIME types the APIs supports when
// accepting requests. While the DSL supports any MIME type, the code generator
// only knows to generate the code for "application/json", "application/xml" and
// "application/gob". The service code must provide the decoders for other MIME
// types.
//
// Consumes must appear in the HTTP expression of API.
//
// Consumes accepts one or more strings corresponding to the MIME types.
//
// Example:
//
//	API("cellar", func() {
//	    // ...
//	    HTTP(func() {
//	        Consumes("application/json", "application/xml")
//	        // ...
//	    })
//	})
func Consumes(args ...string) {
	switch e := eval.Current().(type) {
	case *expr.RootExpr:
		e.API.HTTP.Consumes = append(e.API.HTTP.Consumes, args...)
	default:
		eval.IncompatibleDSL()
	}
}

// Produces adds a MIME type to the list of MIME types the APIs supports when
// writing responses. While the DSL supports any MIME type, the code generator
// only knows to generate the code for "application/json", "application/xml" and
// "application/gob". The service code must provide the encoders for other MIME
// types.
//
// Produces must appear in the HTTP expression of API.
//
// Produces accepts one or more strings corresponding to the MIME types.
//
// Example:
//
//	API("cellar", func() {
//	    // ...
//	    HTTP(func() {
//	        Produces("application/json", "application/xml")
//	        // ...
//	    })
//	})
func Produces(args ...string) {
	switch e := eval.Current().(type) {
	case *expr.RootExpr:
		e.API.HTTP.Produces = append(e.API.HTTP.Produces, args...)
	default:
		eval.IncompatibleDSL()
	}
}

// Path defines an API or service base path, i.e. a common HTTP path prefix to
// all the API or service methods. The path may define wildcards (see GET for a
// description of the wildcard syntax). The corresponding parameters must be
// described using Params. Multiple base paths may be defined for services.
//
// GET("/") does not add a trailing slash when the base path is defined by Path.
// For example, when Path('foo') is defined, the path generated by GET("/") will be '/foo'.
// As a special case, if you want to generate a path with a trailing slash, you can use
// GET("/./") to generate a path such as '/foo/'.
//
// Path must appear in an API HTTP expression or a Service HTTP expression.
//
// Path accepts one argument: the HTTP path prefix.
func Path(val string) {
	switch def := eval.Current().(type) {
	case *expr.RootExpr:
		if expr.Root.API.HTTP.Path != "" {
			eval.ReportError(`only one base path may be specified for an API, got base paths %q and %q`, expr.Root.API.HTTP.Path, val)
		}
		expr.Root.API.HTTP.Path = val
	case *expr.HTTPServiceExpr:
		if !strings.HasPrefix(val, "//") {
			rp := def.Root.Path
			awcs := expr.ExtractHTTPWildcards(rp)
			wcs := expr.ExtractHTTPWildcards(val)
			for _, awc := range awcs {
				for _, wc := range wcs {
					if awc == wc {
						eval.ReportError(`duplicate wildcard "%s" in API and service base paths`, wc)
					}
				}
			}
		}
		def.Paths = append(def.Paths, val)
	default:
		eval.IncompatibleDSL()
	}
}

// GET defines a route using the GET HTTP method. The route may use wildcards to
// define path parameters. Wildcards start with '{' or with '{*' and end with
// '}'. They must appear after a '/'.
//
// A wildcard that starts with '{' matches a section of the path (the value in
// between two slashes).
//
// A wildcard that starts with '{*' matches the rest of the path. Such wildcards
// must terminate the path.
//
// GET must appear in a method HTTP function.
//
// GET accepts one argument which is the request path.
//
// Example:
//
//	var _ = Service("Manager", func() {
//	    Method("GetAccount", func() {
//	        Payload(GetAccount)
//	        Result(Account)
//	        HTTP(func() {
//	            GET("/{accountID}/details")
//	            GET("/{*accountPath}")
//	        })
//	    })
//	})
func GET(path string) *expr.RouteExpr {
	return route("GET", path)
}

// HEAD creates a route using the HEAD HTTP method. See GET.
func HEAD(path string) *expr.RouteExpr {
	return route("HEAD", path)
}

// POST creates a route using the POST HTTP method. See GET.
func POST(path string) *expr.RouteExpr {
	return route("POST", path)
}

// PUT creates a route using the PUT HTTP method. See GET.
func PUT(path string) *expr.RouteExpr {
	return route("PUT", path)
}

// DELETE creates a route using the DELETE HTTP method. See GET.
func DELETE(path string) *expr.RouteExpr {
	return route("DELETE", path)
}

// OPTIONS creates a route using the OPTIONS HTTP method. See GET.
func OPTIONS(path string) *expr.RouteExpr {
	return route("OPTIONS", path)
}

// TRACE creates a route using the TRACE HTTP method. See GET.
func TRACE(path string) *expr.RouteExpr {
	return route("TRACE", path)
}

// CONNECT creates a route using the CONNECT HTTP method. See GET.
func CONNECT(path string) *expr.RouteExpr {
	return route("CONNECT", path)
}

// PATCH creates a route using the PATCH HTTP method. See GET.
func PATCH(path string) *expr.RouteExpr {
	return route("PATCH", path)
}

func route(method, path string) *expr.RouteExpr {
	r := &expr.RouteExpr{Method: method, Path: path}

	switch e := eval.Current().(type) {
	case *expr.HTTPServiceExpr:
		// Service-level route - only allowed for JSON-RPC services
		if e.ServiceExpr.Meta == nil || e.ServiceExpr.Meta["jsonrpc:service"] == nil {
			eval.ReportError("routes at the service level are only allowed for JSON-RPC services. Use method-level routes instead.")
			return r
		}
		// For JSON-RPC services, store the route in the service
		e.JSONRPCRoute = r
		return r

	case *expr.HTTPEndpointExpr:
		// Method-level route - not allowed for JSON-RPC endpoints
		if e.IsJSONRPC() {
			eval.ReportError("JSON-RPC endpoints cannot define routes at the method level. Define routes at the service level using JSONRPC(func() { GET(\"/path\") })")
			return r
		}
		r.Endpoint = e
		e.Routes = append(e.Routes, r)
		return r

	default:
		eval.IncompatibleDSL()
		return r
	}
}

// Header describes a single HTTP header or gRPC metadata header. The properties
// (description, type, validation etc.) of a header are inherited from the
// request or response type attribute with the same name by default.
//
// Header must appear in the API HTTP or JSONRPC expression (to define request
// headers common to all the API endpoints), a service HTTP or JSONRPC
// expression (to define request headers common to all the service endpoints) a
// specific method HTTP or JSONRPC expression (to define request headers) or a
// Response expression (to define the response headers). Header may also appear
// in a method GRPC expression (to define headers sent in message metadata), or
// in a Response expression (to define headers sent in result metadata). Finally
// Header may also appear in a Headers expression.
//
// Header accepts the same arguments as the Attribute function. The header name
// may define a mapping between the attribute name and the HTTP header name when
// they differ. The mapping syntax is "name of attribute:name of header".
//
// Example:
//
//	var _ = Service("account", func() {
//	    Method("create", func() {
//	        Payload(CreatePayload)
//	        Result(Account)
//	        HTTP(func() {
//	            Header("auth:Authorization", String, "Auth token", func() {
//	                Pattern("^Bearer [^ ]+$")
//	            })
//	            Response(StatusCreated, func() {
//	                Header("href") // Inherits description, type, validations
//	                               // etc. from Account href attribute
//	            })
//	        })
//	    })
//	})
func Header(name string, args ...any) {
	h := headers(eval.Current())
	if h == nil {
		eval.IncompatibleDSL()
		return
	}
	if name == "" {
		eval.ReportError("header name cannot be empty")
	}
	eval.Execute(func() { Attribute(name, args...) }, h.AttributeExpr)
	h.Remap()
}

// Cookie identifies a HTTP cookie. When used within a Response the Cookie DSL
// also makes it possible to define the cookie attributes.
//
// Cookie must appear in the API HTTP or JSONRPC expression (to define request
// cookies common to all the API endpoints), a service HTTP or JSONRPC
// expression (to define request cookies common to all the service endpoints) a
// specific method HTTP or JSONRPC expression (to define request cookies) or a
// Response expression (to define the response cookies).
//
// Cookie accepts the same arguments as the Attribute function. The cookie name
// may define a mapping between the attribute name and the cookie name. The
// mapping syntax is "name of attribute:name of cookie".
//
// Example:
//
//	var _ = Service("account", func() {
//	    Method("create", func() {
//	        Payload(func() {
//	            Attribute("session", String, "ID of current session")
//	        })
//	        Result(Account)
//	        HTTP(func() {
//	            // Initialize payload's "session" attribute with the value of
//	            // the SID cookie after validating that's it's a valid GUID.
//	            Cookie("session:SID", String, func() {
//	                Format(FormatGUID)
//	            })
//	            Response(StatusCreated, func() {
//	                // Write the value of the result "session" attribute to
//	                // the cookie named "SID" and initialize the cookie
//	                // "max-age", "domain", "path", "secure" and "http-only"
//	                // attributes. When reading the cookie value client
//	                // side validate that's it is a GUID.
//	                Cookie("session:SID", String, func() {
//	                    Format(FormatGUID)      // Cookie value validations
//	                })
//	                CookieMaxAge(3600)          // Cookie attributes
//	                CookieDomain("goa.design")
//	                CookiePath("/session")
//	                CookieSecure()
//	                CookieHTTPOnly()
//	            })
//	        })
//	    })
//	})
func Cookie(name string, args ...any) {
	if resp, ok := eval.Current().(*expr.HTTPResponseExpr); ok {
		if name == "" {
			eval.ReportError("cookie name cannot be empty")
		}
		cookie := expr.NewHTTPResponseCookieExpr()
		eval.Execute(func() { Attribute(name, args...) }, cookie.AttributeExpr)
		cookie.Remap()
		resp.AddCookie(cookie)
		return
	}
	h := cookies(eval.Current())
	if h == nil {
		eval.IncompatibleDSL()
		return
	}
	if name == "" {
		eval.ReportError("header name cannot be empty")
	}
	eval.Execute(func() { Attribute(name, args...) }, h.AttributeExpr)
	h.Remap()
}

// SessionCookie defines a HTTP response cookie using secure session defaults.
//
// SessionCookie must appear in a HTTP response expression. It behaves like
// Cookie and additionally applies the defaults `Path("/")`, `CookieSecure()`,
// `CookieHTTPOnly()`, and `CookieSameSite(CookieSameSiteLax)`.
//
// Explicit cookie setters invoked after SessionCookie override these defaults.
func SessionCookie(name string, args ...any) {
	_, ok := eval.Current().(*expr.HTTPResponseExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	Cookie(name, args...)
	CookiePath("/")
	CookieSecure()
	CookieHTTPOnly()
	CookieSameSite(CookieSameSiteLax)
}

// CookieMaxAge defines the "max-age" attribute of a HTTP response cookie.
//
// CookieMaxAge must appear in a Cookie expression.
//
// CookieMaxAge accepts one argument which is the max-age value.
//
// Example:
//
//	var _ = Service("account", func() {
//	    Method("create", func() {
//	        Result(Account)
//	        HTTP(func() {
//	            Response(StatusCreated, func() {
//	                Cookie("session:SID", String)
//	                CookieMaxAge(3600)
//	            })
//	        })
//	    })
//	})
func CookieMaxAge(n int) {
	cookieAttribute(func(c *expr.HTTPResponseCookieExpr) {
		c.MaxAge = strconv.Itoa(n)
	})
}

// CookieDomain defines the "domain" attribute of a HTTP response cookie.
//
// CookieDomain must appear in a Cookie expression.
//
// CookieDomain accepts one argument which is the path value.
//
// Example:
//
//	var _ = Service("account", func() {
//	    Method("create", func() {
//	        Result(Account)
//	        HTTP(func() {
//	            Response(StatusCreated, func() {
//	                Cookie("session:SID", String)
//	                CookieDomain("goa.design")
//	            })
//	        })
//	    })
//	})
func CookieDomain(d string) {
	cookieAttribute(func(c *expr.HTTPResponseCookieExpr) {
		c.Domain = d
	})
}

// CookiePath defines the "path" attribute of a HTTP response cookie.
//
// CookiePath must appear in a Cookie expression.
//
// CookiePath accepts one argument which is the path value.
//
// Example:
//
//	var _ = Service("account", func() {
//	    Method("create", func() {
//	        Result(Account)
//	        HTTP(func() {
//	            Response(StatusCreated, func() {
//	                Cookie("session:SID", String)
//	                CookiePath("/session")
//	            })
//	        })
//	    })
//	})
func CookiePath(p string) {
	cookieAttribute(func(c *expr.HTTPResponseCookieExpr) {
		c.Path = p
	})
}

// CookieSecure initializes the "secure" attribute of a HTTP response cookie
// with "Secure".
//
// CookieSecure must appear in a Cookie expression.
//
// Example:
//
//	var _ = Service("account", func() {
//	    Method("create", func() {
//	        Result(Account)
//	        HTTP(func() {
//	            Response(StatusCreated, func() {
//	                Cookie("session:SID", String)
//	                CookieSecure()
//	            })
//	        })
//	    })
//	})
func CookieSecure() {
	cookieAttribute(func(c *expr.HTTPResponseCookieExpr) {
		c.Secure = true
	})
}

// CookieHTTPOnly initializes the "http-only" attribute of a HTTP response
// cookie with "HttpOnly".
//
// CookieHTTPOnly must appear in a Cookie expression.
//
// Example:
//
//	var _ = Service("account", func() {
//	    Method("create", func() {
//	        Result(Account)
//	        HTTP(func() {
//	            Response(StatusCreated, func() {
//	                Cookie("session:SID", String)
//	                CookieHTTPOnly()
//	            })
//	        })
//	    })
//	})
func CookieHTTPOnly() {
	cookieAttribute(func(c *expr.HTTPResponseCookieExpr) {
		c.HTTPOnly = true
	})
}

// CookieSameSite initializes the "same-site" attribute of a HTTP response
// cookie with "CookieSameSiteStrict", "CookieSameSiteLax", "CookieSameSiteNone",
// or "CookieSameSiteDefault".
//
// CookieSameSite must appear in a Cookie expression.
//
// Example:
//
//	var _ = Service("account", func() {
//	    Method("create", func() {
//	        Result(Account)
//	        HTTP(func() {
//	            Response(StatusCreated, func() {
//	                Cookie("session:SID", String)
//	                CookieSameSite(CookieSameSiteStrict)
//	            })
//	        })
//	    })
//	})
func CookieSameSite(s expr.CookieSameSiteValue) {
	cookieAttribute(func(c *expr.HTTPResponseCookieExpr) {
		c.SameSite = s
	})
}

// Params groups a set of Param expressions. It makes it possible to list
// required parameters using the Required function.
//
// Params must appear in an API or Service HTTP expression to define the API or
// service base path and query string parameters. Params may also appear in an
// method HTTP expression to define the HTTP endpoint path and query string
// parameters.
//
// Params accepts one argument which is a function listing the parameters.
//
// Example:
//
//	var _ = API("cellar", func() {
//	    HTTP(func() {
//	        Params(func() {
//	            Param("version", String, "API version", func() {
//	                Enum("1.0", "2.0")
//	            })
//	            Required("version")
//	        })
//	    })
//	})
func Params(args any) {
	p := params(eval.Current())
	if p == nil {
		eval.IncompatibleDSL()
		return
	}
	fn, ok := args.(func())
	if !ok {
		eval.InvalidArgError("function", args)
		return
	}
	eval.Execute(fn, p)
}

// Param describes a single HTTP request path or query string parameter.
//
// Param must appear in the API HTTP expression (to define request parameters
// common to all the API endpoints), a service HTTP expression to define common
// parameters to all the service methods or a specific method HTTP
// expression. Param may also appear in a Params expression.
//
// Param accepts the same arguments as the Function Attribute.
//
// The name may be of the form "name of attribute:name of parameter" to define a
// mapping between the attribute and parameter names when they differ.
//
// Example:
//
//	var ShowPayload = Type("ShowPayload", func() {
//	    Attribute("parentID", UInt64, "ID of parent account")
//	    Attribute("id", UInt64, "Account ID")
//	    Attribute("version", String, "Version", func() {
//	        Enum("1.0", "2.0")
//	    })
//	})
//
//	var _ = Service("account", func() {
//	    HTTP(func() {
//	        Path("/{parentID}") // HTTP request uses ShowPayload "parentID"
//	        // attribute to define "parentID" parameter.
//	    })
//	    Method("show", func() {  // default response type.
//	        Payload(ShowPayload)
//	        Result(AccountResult)
//	        HTTP(func() {
//	            GET("/{id}")           // HTTP request uses ShowPayload "id"
//	                                   // attribute to define "id" parameter.
//	            Params(func() {        // Params makes it possible to group
//	                                   // Param expressions.
//	                Param("version:v") // "version" of ShowPayload to define
//	                                   // path and query string parameters.
//	                                   // Query string "v" maps to attribute
//	                                   // "version" of ShowPayload.
//	            })
//	        })
//	    })
//	})
func Param(name string, args ...any) {
	p := params(eval.Current())
	if p == nil {
		eval.IncompatibleDSL()
		return
	}
	if name == "" {
		eval.ReportError("parameter name cannot be empty")
	}
	eval.Execute(func() { Attribute(name, args...) }, p.AttributeExpr)
	p.Remap()
}

// MapParams describes the query string parameters in a HTTP request.
//
// MapParams must appear in a Method HTTP expression to map the query string
// parameters with the Method's Payload.
//
// MapParams accepts one optional argument which specifes the Payload
// attribute to which the query string parameters must be mapped. This Payload
// attribute must be a map. If no argument is specified, the query string
// parameters are mapped with the entire Payload (the Payload must be a map).
//
// Example:
//
//	 var _ = Service("account", func() {
//	     Method("index", func() {
//	         Payload(MapOf(String, Int))
//	         HTTP(func() {
//	             GET("/")
//	             MapParams()
//	         })
//	     })
//	})
//
//	var _ = Service("account", func() {
//	    Method("show", func() {
//	        Payload(func() {
//	            Attribute("p", MapOf(String, String))
//	            Attribute("id", String)
//	        })
//	        HTTP(func() {
//	            GET("/{id}")
//	            MapParams("p")
//	        })
//	    })
//	})
func MapParams(args ...any) {
	if len(args) > 1 {
		eval.TooManyArgError()
		return
	}
	e, ok := eval.Current().(*expr.HTTPEndpointExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	var mapName string
	if len(args) > 0 {
		mapName, ok = args[0].(string)
		if !ok {
			eval.InvalidArgError("string", args[0])
		}
	}
	e.MapQueryParams = &mapName
}

// Parent sets the name of the parent service. The parent service canonical
// method path is used as prefix for all the service HTTP endpoint paths.
//
// Attributes of the parent method payload that map to parent path parameters
// are automatically merged into the child method payload type if not already
// defined.
//
// Parent must appear in the HTTP expresssion of a Service.
//
// Parent accepts one argument: the name of the parent service.
func Parent(name string) {
	r, ok := eval.Current().(*expr.HTTPServiceExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	r.ParentName = name
}

// CanonicalMethod sets the name of the service canonical method. The canonical
// method endpoint HTTP path is used to prefix the paths to child service
// endpoints (a child service is a service that uses the Parent function). The
// default value is "show".
//
// CanonicalMethod must appear in the HTTP expresssion of a Service.
//
// CanonicalMethod accepts one argument: the name of the canonical service
// method.
func CanonicalMethod(name string) {
	r, ok := eval.Current().(*expr.HTTPServiceExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	r.CanonicalEndpointName = name
}

// Tag identifies a method result type field and a value. The algorithm that
// encodes the result into the HTTP response iterates through the responses and
// uses the first response that has a matching tag (that is for which the result
// field with the tag name matches the tag value). There must be one and only
// one response with no Tag expression, this response is used when no other tag
// matches.
//
// Tag must appear in Response.
//
// Tag accepts two arguments: the name of the field and the (string) value.
//
// Example:
//
//	Method("create", func() {
//	    Result(CreateResult)
//	    HTTP(func() {
//	        Response(StatusCreated, func() {
//	            Tag("outcome", "created") // Assumes CreateResult has attribute
//	                                      // "outcome" which may be "created"
//	                                      // or "accepted"
//	        })
//
//	        Response(StatusAccepted, func() {
//	            Tag("outcome", "accepted")
//	        })
//
//	        Response(StatusOK)            // Default response if "outcome" is
//	                                      // neither "created" nor "accepted"
//	    })
//	})
func Tag(name, value string) {
	res, ok := eval.Current().(*expr.HTTPResponseExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	res.Tag = [2]string{name, value}
}

// ContentType sets the value of the Content-Type response header.
//
// ContentType must appear in a Response expression.
// ContentType accepts one argument: the mime type as defined by RFC 6838.
//
//	   var _ = Method("add", func() {
//		      HTTP(func() {
//	           Response(StatusOK, func() {
//	               ContentType("application/json")
//	           })
//	       })
//	   })
func ContentType(typ string) {
	switch actual := eval.Current().(type) {
	case *expr.ResultTypeExpr:
		actual.ContentType = typ // deprecated
	case *expr.HTTPResponseExpr:
		actual.ContentType = typ
	default:
		eval.IncompatibleDSL()
	}
}

// cookieAttribute mutates the active response cookie.
func cookieAttribute(update func(*expr.HTTPResponseCookieExpr)) {
	cookie, ok := currentResponseCookie()
	if !ok {
		if _, isResponse := eval.Current().(*expr.HTTPResponseExpr); !isResponse {
			eval.IncompatibleDSL()
		} else {
			eval.ReportError("cookie attributes must be declared after Cookie or SessionCookie in an HTTP response")
		}
		return
	}
	update(cookie)
}

func currentResponseCookie() (*expr.HTTPResponseCookieExpr, bool) {
	resp, ok := eval.Current().(*expr.HTTPResponseExpr)
	if !ok {
		return nil, false
	}
	cookie := resp.CurrentCookie()
	if cookie == nil {
		return nil, false
	}
	return cookie, true
}
