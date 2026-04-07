package testdata


var PayloadBodyUnionValidateDecodeCode = `// DecodeMethodBodyUnionValidateRequest returns a decoder for requests sent to
// the ServiceBodyUnionValidate MethodBodyUnionValidate endpoint.
func DecodeMethodBodyUnionValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			body MethodBodyUnionValidateRequestBody
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
		err = ValidateMethodBodyUnionValidateRequestBody(&body)
		if err != nil {
			return nil, err
		}
		payload := NewMethodBodyUnionValidatePayload(&body)

		return payload, nil
	}
}
`


