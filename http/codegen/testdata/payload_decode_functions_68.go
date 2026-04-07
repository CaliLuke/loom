package testdata


var PayloadHeaderStringDefaultValidateDecodeCode = `// DecodeMethodHeaderStringDefaultValidateRequest returns a decoder for
// requests sent to the ServiceHeaderStringDefaultValidate
// MethodHeaderStringDefaultValidate endpoint.
func DecodeMethodHeaderStringDefaultValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			h   string
			err error
		)
		hRaw := r.Header.Get("h")
		if hRaw != "" {
			h = hRaw
		} else {
			h = "def"
		}
		if !(h == "def") {
			err = loom.MergeErrors(err, loom.InvalidEnumValueError("h", h, []any{"def"}))
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodHeaderStringDefaultValidatePayload(h)

		return payload, nil
	}
}
`


