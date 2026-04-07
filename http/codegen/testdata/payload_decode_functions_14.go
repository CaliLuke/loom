package testdata


var PayloadQueryArrayStringDecodeCode = `// DecodeMethodQueryArrayStringRequest returns a decoder for requests sent to
// the ServiceQueryArrayString MethodQueryArrayString endpoint.
func DecodeMethodQueryArrayStringRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q []string
		)
		q = r.URL.Query()["q"]
		payload := NewMethodQueryArrayStringPayload(q)

		return payload, nil
	}
}
`


