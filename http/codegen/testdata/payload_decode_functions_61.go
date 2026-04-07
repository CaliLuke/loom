package testdata


var PayloadHeaderArrayStringDecodeCode = `// DecodeMethodHeaderArrayStringRequest returns a decoder for requests sent to
// the ServiceHeaderArrayString MethodHeaderArrayString endpoint.
func DecodeMethodHeaderArrayStringRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			h []string
		)
		h = r.Header["H"]
		payload := NewMethodHeaderArrayStringPayload(h)

		return payload, nil
	}
}
`


