package testdata


var PayloadCookieStringDefaultValidateDecodeCode = `// DecodeMethodCookieStringDefaultValidateRequest returns a decoder for
// requests sent to the ServiceCookieStringDefaultValidate
// MethodCookieStringDefaultValidate endpoint.
func DecodeMethodCookieStringDefaultValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
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
				} else {
					return payload, cookieErr
				}
			}
			var c2Raw string
			if c != nil {
				c2Raw = c.Value
			}
			if c2Raw != "" {
				c2 = c2Raw
			} else {
				c2 = "def"
			}
		}
		if !(c2 == "def") {
			err = loom.MergeErrors(err, loom.InvalidEnumValueError("c", c2, []any{"def"}))
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodCookieStringDefaultValidatePayload(c2)

		return payload, nil
	}
}
`


