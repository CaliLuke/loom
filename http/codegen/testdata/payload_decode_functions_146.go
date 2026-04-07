package testdata


var PayloadPathCustomNameDecodeCode = `// DecodeMethodPathCustomNameRequest returns a decoder for requests sent to the
// ServicePathCustomName MethodPathCustomName endpoint.
func DecodeMethodPathCustomNameRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			p string

			params = mux.Vars(r)
		)
		p = params["p"]
		payload := NewMethodPathCustomNamePayload(p)

		return payload, nil
	}
}
`


