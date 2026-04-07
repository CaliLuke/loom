package testdata


var PayloadQueryCustomNameDecodeCode = `// DecodeMethodQueryCustomNameRequest returns a decoder for requests sent to
// the ServiceQueryCustomName MethodQueryCustomName endpoint.
func DecodeMethodQueryCustomNameRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q *string
		)
		qRaw := r.URL.Query().Get("q")
		if qRaw != "" {
			q = &qRaw
		}
		payload := NewMethodQueryCustomNamePayload(q)

		return payload, nil
	}
}
`


