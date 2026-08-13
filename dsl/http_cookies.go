package dsl

import (
	"strconv"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

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
//	                CookieDomain("loom.dev")
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
// CookieMaxAge must appear after Cookie or SessionCookie in a HTTP response
// expression.
//
// CookieMaxAge accepts one argument which is the max-age value in seconds. A
// positive value sets the cookie lifetime, zero omits the Max-Age attribute,
// and a negative value deletes the cookie immediately.
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
//	            Response(StatusNoContent, func() {
//	                SessionCookie("expired_session:SID", String)
//	                CookieMaxAge(-1) // Delete the cookie immediately.
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
//	                CookieDomain("loom.dev")
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

// CookieInsecure removes the "secure" attribute from a HTTP response cookie.
//
// CookieInsecure must appear after Cookie or SessionCookie in a HTTP response
// expression. Use it only for plain-HTTP local development; production session
// cookies should retain the secure default. CookieInsecure must not be used with
// cookie names that start with "__Host-" or "__Secure-", or with
// CookieSameSite(CookieSameSiteNone), because browsers require those cookies to
// be secure.
//
// Example:
//
//	var _ = Service("account", func() {
//	    Method("create", func() {
//	        Result(Account)
//	        HTTP(func() {
//	            Response(StatusCreated, func() {
//	                SessionCookie("session:SID", String)
//	                CookieInsecure() // Plain-HTTP local development only.
//	            })
//	        })
//	    })
//	})
func CookieInsecure() {
	cookieAttribute(func(c *expr.HTTPResponseCookieExpr) {
		c.Secure = false
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
