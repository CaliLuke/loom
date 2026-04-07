package testdata


var PayloadBodyStringValidateDecodeCode = `// DecodeMethodBodyStringValidateRequest returns a decoder for requests sent to
// the ServiceBodyStringValidate MethodBodyStringValidate endpoint.
func DecodeMethodBodyStringValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			body MethodBodyStringValidateRequestBody
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
		err = ValidateMethodBodyStringValidateRequestBody(&body)
		if err != nil {
			return nil, err
		}
		payload := NewMethodBodyStringValidatePayload(&body)

		return payload, nil
	}
}
`


