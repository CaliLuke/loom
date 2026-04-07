package testdata


var PayloadPathStringDecodeCode = `// DecodeMethodPathStringRequest returns a decoder for requests sent to the
// ServicePathString MethodPathString endpoint.
func DecodeMethodPathStringRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			p string

			params = mux.Vars(r)
		)
		p = params["p"]
		payload := NewMethodPathStringPayload(p)

		return payload, nil
	}
}
`


