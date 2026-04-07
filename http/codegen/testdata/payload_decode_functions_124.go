package testdata


var PayloadMultipartUserTypeDecodeCode = `// DecodeMethodMultipartUserTypeRequest returns a decoder for requests sent to
// the ServiceMultipartUserType MethodMultipartUserType endpoint.
func DecodeMethodMultipartUserTypeRequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var payload *servicemultipartusertype.MethodMultipartUserTypePayload
		if err := decoder(r).Decode(&payload); err != nil {
			var gerr *loom.ServiceError
			if errors.As(err, &gerr) {
				return nil, gerr
			}
			return nil, loom.DecodePayloadError(err.Error())
		}

		return payload, nil
	}
}
`


