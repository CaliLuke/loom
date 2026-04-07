package testdata


var PayloadHeaderStringDecodeCode = `// DecodeMethodHeaderStringRequest returns a decoder for requests sent to the
// ServiceHeaderString MethodHeaderString endpoint.
func DecodeMethodHeaderStringRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			h *string
		)
		hRaw := r.Header.Get("h")
		if hRaw != "" {
			h = &hRaw
		}
		payload := NewMethodHeaderStringPayload(h)

		return payload, nil
	}
}
`


