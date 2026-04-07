package testdata


var PayloadHeaderCustomNameDecodeCode = `// DecodeMethodHeaderCustomNameRequest returns a decoder for requests sent to
// the ServiceHeaderCustomName MethodHeaderCustomName endpoint.
func DecodeMethodHeaderCustomNameRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			h *string
		)
		hRaw := r.Header.Get("h")
		if hRaw != "" {
			h = &hRaw
		}
		payload := NewMethodHeaderCustomNamePayload(h)

		return payload, nil
	}
}
`


