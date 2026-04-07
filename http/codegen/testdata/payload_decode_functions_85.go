package testdata


var PayloadBodyObjectValidateDecodeCode = `// DecodeMethodBodyObjectValidateRequest returns a decoder for requests sent to
// the ServiceBodyObjectValidate MethodBodyObjectValidate endpoint.
func DecodeMethodBodyObjectValidateRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			body struct {
				B *string ` + "`" + `form:"b" json:"b" xml:"b"` + "`" + `
			}
			err error
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
		if body.B != nil {
			err = loom.MergeErrors(err, loom.ValidatePattern("body.b", *body.B, "pattern"))
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodBodyObjectValidatePayload(body)

		return payload, nil
	}
}
`


