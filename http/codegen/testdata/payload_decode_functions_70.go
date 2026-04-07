package testdata


var PayloadCookieStringDecodeCode = `// DecodeMethodCookieStringRequest returns a decoder for requests sent to the
// ServiceCookieString MethodCookieString endpoint.
func DecodeMethodCookieStringRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			c2 *string
			c  *http.Cookie
		)
		{
			c, cookieErr := r.Cookie("c")
			if cookieErr != nil {
				if errors.Is(cookieErr, http.ErrNoCookie) {
				} else {
					return payload, cookieErr
				}
			}
			var c2Raw string
			if c != nil {
				c2Raw = c.Value
			}
			if c2Raw != "" {
				c2 = &c2Raw
			}
		}
		payload := NewMethodCookieStringPayload(c2)

		return payload, nil
	}
}
`


