package expr

type (
	// HTTPResponseCookieExpr describes a single HTTP response cookie.
	HTTPResponseCookieExpr struct {
		// MappedAttributeExpr contains the mapped result attribute for the cookie
		// value.
		*MappedAttributeExpr
		// Path defines the cookie Path attribute.
		Path string
		// Domain defines the cookie Domain attribute.
		Domain string
		// MaxAge defines the cookie Max-Age attribute.
		MaxAge string
		// Secure indicates whether the cookie should include the Secure flag.
		Secure bool
		// HTTPOnly indicates whether the cookie should include the HttpOnly flag.
		HTTPOnly bool
		// SameSite defines the cookie SameSite attribute.
		SameSite CookieSameSiteValue
	}
)

// NewHTTPResponseCookieExpr creates an empty HTTP response cookie expression.
func NewHTTPResponseCookieExpr() *HTTPResponseCookieExpr {
	return &HTTPResponseCookieExpr{MappedAttributeExpr: NewEmptyMappedAttributeExpr()}
}

// Dup creates a copy of the response cookie expression.
func (c *HTTPResponseCookieExpr) Dup() *HTTPResponseCookieExpr {
	if c == nil {
		return nil
	}
	return &HTTPResponseCookieExpr{
		MappedAttributeExpr: DupMappedAtt(c.MappedAttributeExpr),
		Path:                c.Path,
		Domain:              c.Domain,
		MaxAge:              c.MaxAge,
		Secure:              c.Secure,
		HTTPOnly:            c.HTTPOnly,
		SameSite:            c.SameSite,
	}
}

// IsEmpty returns true if the cookie contains no mapped attribute.
func (c *HTTPResponseCookieExpr) IsEmpty() bool {
	return c == nil || c.MappedAttributeExpr == nil || c.MappedAttributeExpr.IsEmpty()
}

// AttributeName returns the mapped result attribute name.
func (c *HTTPResponseCookieExpr) AttributeName() string {
	if c.IsEmpty() {
		return ""
	}
	o := AsObject(c.Type)
	if len(*o) == 0 {
		return ""
	}
	return (*o)[0].Name
}

// HTTPName returns the cookie name written on the wire.
func (c *HTTPResponseCookieExpr) HTTPName() string {
	name := c.AttributeName()
	if name == "" {
		return ""
	}
	return c.ElemName(name)
}

// Attribute returns the mapped cookie attribute.
func (c *HTTPResponseCookieExpr) Attribute() *AttributeExpr {
	name := c.AttributeName()
	if name == "" {
		return nil
	}
	return c.Find(name)
}
