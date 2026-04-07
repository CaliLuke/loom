package testdata


var PayloadQueryArrayBytesDecodeCode = `// DecodeMethodQueryArrayBytesRequest returns a decoder for requests sent to
// the ServiceQueryArrayBytes MethodQueryArrayBytes endpoint.
func DecodeMethodQueryArrayBytesRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			q [][]byte
		)
		{
			qRaw := r.URL.Query()["q"]
			if qRaw != nil {
				q = make([][]byte, len(qRaw))
				for i, rv := range qRaw {
					q[i] = []byte(rv)
				}
			}
		}
		payload := NewMethodQueryArrayBytesPayload(q)

		return payload, nil
	}
}
`


