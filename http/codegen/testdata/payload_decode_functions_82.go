package testdata


var PayloadBodyUserValidateDecodeCode = `// DecodeMethodBodyUserValidateRequest returns a decoder for requests sent to
// the ServiceBodyUserValidate MethodBodyUserValidate endpoint.
func DecodeMethodBodyUserValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
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
		err = loom.MergeErrors(err, loom.ValidatePattern("body", body, "apattern"))
		if err != nil {
			return nil, err
		}
		payload := NewMethodBodyUserValidatePayloadType(body)

		return payload, nil
	}
}
`


