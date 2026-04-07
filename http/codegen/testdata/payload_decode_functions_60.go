package testdata


var PayloadHeaderStringValidateDecodeCode = `// DecodeMethodHeaderStringValidateRequest returns a decoder for requests sent
// to the ServiceHeaderStringValidate MethodHeaderStringValidate endpoint.
func DecodeMethodHeaderStringValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			h   *string
			err error
		)
		hRaw := r.Header.Get("h")
		if hRaw != "" {
			h = &hRaw
		}
		if h != nil {
			err = loom.MergeErrors(err, loom.ValidatePattern("h", *h, "header"))
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodHeaderStringValidatePayload(h)

		return payload, nil
	}
}
`


