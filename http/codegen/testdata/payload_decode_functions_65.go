package testdata


var PayloadHeaderPrimitiveArrayStringValidateDecodeCode = `// DecodeMethodHeaderPrimitiveArrayStringValidateRequest returns a decoder for
// requests sent to the ServiceHeaderPrimitiveArrayStringValidate
// MethodHeaderPrimitiveArrayStringValidate endpoint.
func DecodeMethodHeaderPrimitiveArrayStringValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			h   []string
			err error
		)
		h = r.Header["H"]
		if h == nil {
			err = loom.MergeErrors(err, loom.MissingFieldError("h", "header"))
		}
		if len(h) < 1 {
			err = loom.MergeErrors(err, loom.InvalidLengthError("h", h, len(h), 1, true))
		}
		for _, e := range h {
			err = loom.MergeErrors(err, loom.ValidatePattern("h[*]", e, "val"))
		}
		if err != nil {
			return nil, err
		}
		payload := h

		return payload, nil
	}
}
`


