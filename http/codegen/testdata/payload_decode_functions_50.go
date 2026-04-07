package testdata


var PayloadExtendedQueryStringDecodeCode = `// DecodeMethodQueryStringExtendedPayloadRequest returns a decoder for requests
// sent to the ServiceQueryStringExtendedPayload
// MethodQueryStringExtendedPayload endpoint.
func DecodeMethodQueryStringExtendedPayloadRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q *string
		)
		qRaw := r.URL.Query().Get("q")
		if qRaw != "" {
			q = &qRaw
		}
		payload := NewMethodQueryStringExtendedPayloadPayload(q)

		return payload, nil
	}
}
`


