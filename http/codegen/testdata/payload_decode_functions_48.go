package testdata


var PayloadQueryStringDefaultValidateDecodeCode = `// DecodeMethodQueryStringDefaultValidateRequest returns a decoder for requests
// sent to the ServiceQueryStringDefaultValidate
// MethodQueryStringDefaultValidate endpoint.
func DecodeMethodQueryStringDefaultValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q   string
			err error
		)
		qRaw := r.URL.Query().Get("q")
		if qRaw != "" {
			q = qRaw
		} else {
			q = "def"
		}
		if !(q == "def") {
			err = loom.MergeErrors(err, loom.InvalidEnumValueError("q", q, []any{"def"}))
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodQueryStringDefaultValidatePayload(q)

		return payload, nil
	}
}
`


