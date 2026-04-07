package testdata


var PayloadBodyPrimitiveArrayUserValidateDecodeCode = `// DecodeMethodBodyPrimitiveArrayUserValidateRequest returns a decoder for
// requests sent to the ServiceBodyPrimitiveArrayUserValidate
// MethodBodyPrimitiveArrayUserValidate endpoint.
func DecodeMethodBodyPrimitiveArrayUserValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			body []*PayloadTypeRequestBody
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
			if e != nil {
				if err2 := ValidatePayloadTypeRequestBody(e); err2 != nil {
					err = loom.MergeErrors(err, err2)
				}
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodBodyPrimitiveArrayUserValidatePayloadType(body)

		return payload, nil
	}
}
`


