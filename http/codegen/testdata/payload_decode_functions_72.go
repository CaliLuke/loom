package testdata


var PayloadCookiePrimitiveStringValidateDecodeCode = `// DecodeMethodCookiePrimitiveStringValidateRequest returns a decoder for
// requests sent to the ServiceCookiePrimitiveStringValidate
// MethodCookiePrimitiveStringValidate endpoint.
func DecodeMethodCookiePrimitiveStringValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
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
		if !(c2 == "val") {
			err = loom.MergeErrors(err, loom.InvalidEnumValueError("c", c2, []any{"val"}))
		}
		if err != nil {
			return nil, err
		}
		payload := c

		return payload, nil
	}
}
`


