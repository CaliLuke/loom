package testdata


var PayloadBodyArrayUserValidateDecodeCode = `// DecodeMethodBodyArrayUserValidateRequest returns a decoder for requests sent
// to the ServiceBodyArrayUserValidate MethodBodyArrayUserValidate endpoint.
func DecodeMethodBodyArrayUserValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			body MethodBodyArrayUserValidateRequestBody
			err  error
		)
		err = decoder(r).Decode(&body)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil, loom.MissingPayloadError()
			}
			var gerr *loom.ServiceError
			if errors.As(err, &gerr) {
				return nil, gerr
			}
			return nil, loom.DecodePayloadError(err.Error())
		}
		err = ValidateMethodBodyArrayUserValidateRequestBody(&body)
		if err != nil {
			return nil, err
		}
		payload := NewMethodBodyArrayUserValidatePayload(&body)

		return payload, nil
	}
}
`


