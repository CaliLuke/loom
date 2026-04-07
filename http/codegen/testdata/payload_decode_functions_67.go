package testdata


var PayloadHeaderStringDefaultDecodeCode = `// DecodeMethodHeaderStringDefaultRequest returns a decoder for requests sent
// to the ServiceHeaderStringDefault MethodHeaderStringDefault endpoint.
func DecodeMethodHeaderStringDefaultRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			h string
		)
		hRaw := r.Header.Get("h")
		if hRaw != "" {
			h = hRaw
		} else {
			h = "def"
		}
		payload := NewMethodHeaderStringDefaultPayload(h)

		return payload, nil
	}
}
`


