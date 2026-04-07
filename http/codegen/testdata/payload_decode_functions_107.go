package testdata


var PayloadBodyPrimitiveFieldArrayUserValidateDecodeCode = `// DecodeMethodBodyPrimitiveArrayUserValidateRequest returns a decoder for
// requests sent to the ServiceBodyPrimitiveArrayUserValidate
// MethodBodyPrimitiveArrayUserValidate endpoint.
func DecodeMethodBodyPrimitiveArrayUserValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			body []string
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
		if len(body) < 1 {
			err = loom.MergeErrors(err, loom.InvalidLengthError("body", body, len(body), 1, true))
		}
		for _, e := range body {
			err = loom.MergeErrors(err, loom.ValidatePattern("body[*]", e, "pattern"))
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodBodyPrimitiveArrayUserValidatePayloadType(body)

		return payload, nil
	}
}
`


