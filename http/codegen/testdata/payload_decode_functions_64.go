package testdata


var PayloadHeaderPrimitiveBoolValidateDecodeCode = `// DecodeMethodHeaderPrimitiveBoolValidateRequest returns a decoder for
// requests sent to the ServiceHeaderPrimitiveBoolValidate
// MethodHeaderPrimitiveBoolValidate endpoint.
func DecodeMethodHeaderPrimitiveBoolValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			h   bool
			err error
		)
		{
			hRaw := r.Header.Get("h")
			if hRaw == "" {
				err = loom.MergeErrors(err, loom.MissingFieldError("h", "header"))
			}
			v, err2 := strconv.ParseBool(hRaw)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError("h", hRaw, "boolean"))
			}
			h = v
		}
		if !(h == true) {
			err = loom.MergeErrors(err, loom.InvalidEnumValueError("h", h, []any{true}))
		}
		if err != nil {
			return nil, err
		}
		payload := h

		return payload, nil
	}
}
`


