package testdata


var PayloadHeaderArrayStringValidateDecodeCode = `// DecodeMethodHeaderArrayStringValidateRequest returns a decoder for requests
// sent to the ServiceHeaderArrayStringValidate MethodHeaderArrayStringValidate
// endpoint.
func DecodeMethodHeaderArrayStringValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			h   []string
			err error
		)
		h = r.Header["H"]
		for _, e := range h {
			if !(e == "val") {
				err = loom.MergeErrors(err, loom.InvalidEnumValueError("h[*]", e, []any{"val"}))
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodHeaderArrayStringValidatePayload(h)

		return payload, nil
	}
}
`


