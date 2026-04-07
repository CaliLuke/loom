package testdata


var PayloadPathStringValidateDecodeCode = `// DecodeMethodPathStringValidateRequest returns a decoder for requests sent to
// the ServicePathStringValidate MethodPathStringValidate endpoint.
func DecodeMethodPathStringValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			p   string
			err error

			params = mux.Vars(r)
		)
		p = params["p"]
		if !(p == "val") {
			err = loom.MergeErrors(err, loom.InvalidEnumValueError("p", p, []any{"val"}))
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodPathStringValidatePayload(p)

		return payload, nil
	}
}
`


