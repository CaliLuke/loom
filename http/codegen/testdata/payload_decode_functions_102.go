package testdata


var PayloadBodyPrimitiveArrayBoolValidateDecodeCode = `// DecodeMethodBodyPrimitiveArrayBoolValidateRequest returns a decoder for
// requests sent to the ServiceBodyPrimitiveArrayBoolValidate
// MethodBodyPrimitiveArrayBoolValidate endpoint.
func DecodeMethodBodyPrimitiveArrayBoolValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			body []bool
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
			if !(e == true) {
				err = loom.MergeErrors(err, loom.InvalidEnumValueError("body[*]", e, []any{true}))
			}
		}
		if err != nil {
			return nil, err
		}
		payload := body

		return payload, nil
	}
}
`


