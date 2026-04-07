package testdata


var PayloadBodyPrimitiveFieldStringDecodeCode = `// DecodeMethodBodyPrimitiveArrayUserRequest returns a decoder for requests
// sent to the ServiceBodyPrimitiveArrayUser MethodBodyPrimitiveArrayUser
// endpoint.
func DecodeMethodBodyPrimitiveArrayUserRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			body string
			err  error
		)
		err = decoder(r).Decode(&body)
		if err != nil {
			if errors.Is(err, io.EOF) {
				err = nil
			} else {
				var gerr *loom.ServiceError
				if errors.As(err, &gerr) {
					return nil, gerr
				}
				return nil, loom.DecodePayloadError(err.Error())
			}
		}
		payload := NewMethodBodyPrimitiveArrayUserPayloadType(body)

		return payload, nil
	}
}
`


