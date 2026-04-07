package testdata


var PayloadCookiePrimitiveStringDefaultDecodeCode = `// DecodeMethodCookiePrimitiveStringDefaultRequest returns a decoder for
// requests sent to the ServiceCookiePrimitiveStringDefault
// MethodCookiePrimitiveStringDefault endpoint.
func DecodeMethodCookiePrimitiveStringDefaultRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			c2  string
			err error
			c   *http.Cookie
		)
		{
			c, cookieErr := r.Cookie("c")
			if cookieErr != nil {
				if errors.Is(cookieErr, http.ErrNoCookie) {
					err = loom.MergeErrors(err, loom.MissingFieldError("c", "cookie"))
				} else {
					return payload, cookieErr
				}
			}
			if c != nil {
				c2 = c.Value
			}
		}
		if err != nil {
			return nil, err
		}
		payload := c

		return payload, nil
	}
}
`


