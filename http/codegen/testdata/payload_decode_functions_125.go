package testdata


var PayloadMultipartArrayTypeDecodeCode = `// DecodeMethodMultipartArrayTypeRequest returns a decoder for requests sent to
// the ServiceMultipartArrayType MethodMultipartArrayType endpoint.
func DecodeMethodMultipartArrayTypeRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var payload []*servicemultipartarraytype.PayloadType
		if err := decoder(r).Decode(&payload); err != nil {
			var gerr *loom.ServiceError
			if errors.As(err, &gerr) {
				return nil, gerr
			}
			return nil, loom.DecodePayloadError(err.Error())
		}

		return payload, nil
	}
}
`


