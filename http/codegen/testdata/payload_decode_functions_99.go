package testdata


var PayloadBodyPrimitiveStringValidateDecodeCode = `// DecodeMethodBodyPrimitiveStringValidateRequest returns a decoder for
// requests sent to the ServiceBodyPrimitiveStringValidate
// MethodBodyPrimitiveStringValidate endpoint.
func DecodeMethodBodyPrimitiveStringValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			body string
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
		if !(body == "val") {
			err = loom.MergeErrors(err, loom.InvalidEnumValueError("body", body, []any{"val"}))
		}
		if err != nil {
			return nil, err
		}
		payload := body

		return payload, nil
	}
}
`


