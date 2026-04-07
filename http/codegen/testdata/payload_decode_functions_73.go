package testdata


var PayloadCookiePrimitiveBoolValidateDecodeCode = `// DecodeMethodCookiePrimitiveBoolValidateRequest returns a decoder for
// requests sent to the ServiceCookiePrimitiveBoolValidate
// MethodCookiePrimitiveBoolValidate endpoint.
func DecodeMethodCookiePrimitiveBoolValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			c2  bool
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
			var c2Raw string
			if c != nil {
				c2Raw = c.Value
			}
			if c2Raw == "" {
				err = loom.MergeErrors(err, loom.MissingFieldError("c", "cookie"))
			}
			v, err2 := strconv.ParseBool(c2Raw)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError("c", c2Raw, "boolean"))
			}
			c2 = v
		}
		if !(c2 == true) {
			err = loom.MergeErrors(err, loom.InvalidEnumValueError("c", c2, []any{true}))
		}
		if err != nil {
			return nil, err
		}
		payload := c

		return payload, nil
	}
}
`


