package testdata


var PayloadHeaderPrimitiveStringDefaultDecodeCode = `// DecodeMethodHeaderPrimitiveStringDefaultRequest returns a decoder for
// requests sent to the ServiceHeaderPrimitiveStringDefault
// MethodHeaderPrimitiveStringDefault endpoint.
func DecodeMethodHeaderPrimitiveStringDefaultRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			h   string
			err error
		)
		h = r.Header.Get("h")
		if h == "" {
			err = loom.MergeErrors(err, loom.MissingFieldError("h", "header"))
		}
		if err != nil {
			return nil, err
		}
		payload := h

		return payload, nil
	}
}
`


