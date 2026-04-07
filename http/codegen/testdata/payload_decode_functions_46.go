package testdata


var PayloadQueryStringDefaultDecodeCode = `// DecodeMethodQueryStringDefaultRequest returns a decoder for requests sent to
// the ServiceQueryStringDefault MethodQueryStringDefault endpoint.
func DecodeMethodQueryStringDefaultRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q string
		)
		qRaw := r.URL.Query().Get("q")
		if qRaw != "" {
			q = qRaw
		} else {
			q = "def"
		}
		payload := NewMethodQueryStringDefaultPayload(q)

		return payload, nil
	}
}
`


