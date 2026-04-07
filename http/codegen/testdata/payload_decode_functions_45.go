package testdata


var PayloadQueryStringMappedDecodeCode = `// DecodeMethodQueryStringMappedRequest returns a decoder for requests sent to
// the ServiceQueryStringMapped MethodQueryStringMapped endpoint.
func DecodeMethodQueryStringMappedRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			query *string
		)
		queryRaw := r.URL.Query().Get("q")
		if queryRaw != "" {
			query = &queryRaw
		}
		payload := NewMethodQueryStringMappedPayload(query)

		return payload, nil
	}
}
`


