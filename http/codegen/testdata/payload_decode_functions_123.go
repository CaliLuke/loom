package testdata


var PayloadMultipartPrimitiveDecodeCode = `// DecodeMethodMultipartPrimitiveRequest returns a decoder for requests sent to
// the ServiceMultipartPrimitive MethodMultipartPrimitive endpoint.
func DecodeMethodMultipartPrimitiveRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var payload string
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


