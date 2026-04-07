package testdata


var PayloadQueryStringSliceDefaultDecodeCode = `// DecodeMethodQueryStringSliceDefaultRequest returns a decoder for requests
// sent to the ServiceQueryStringSliceDefault MethodQueryStringSliceDefault
// endpoint.
func DecodeMethodQueryStringSliceDefaultRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q []string
		)
		q = r.URL.Query()["q"]
		if q == nil {
			q = []string{"hello", "goodbye"}
		}
		payload := NewMethodQueryStringSliceDefaultPayload(q)

		return payload, nil
	}
}
`


