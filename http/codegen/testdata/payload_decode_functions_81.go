package testdata


var PayloadBodyNestedUserDecodeCode = `// DecodeMethodBodyUserRequest returns a decoder for requests sent to the
// ServiceBodyUser MethodBodyUser endpoint.
func DecodeMethodBodyUserRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			body MethodBodyUserRequestBody
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
		err = ValidateMethodBodyUserRequestBody(&body)
		if err != nil {
			return nil, err
		}
		payload := NewMethodBodyUserPayloadType(&body)

		return payload, nil
	}
}
`


