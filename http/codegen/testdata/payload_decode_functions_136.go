package testdata


var PathIntAliasDecodeCode = `// DecodeMethodARequest returns a decoder for requests sent to the
// ServicePathIntAlias MethodA endpoint.
func DecodeMethodARequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			int_   int
			int32_ int32
			int64_ int64
			err    error

			params = mux.Vars(r)
		)
		{
			int_Raw := params["int"]
			v, err2 := strconv.ParseInt(int_Raw, 10, strconv.IntSize)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError("int", int_Raw, "integer"))
			}
			int_ = int(v)
		}
		{
			int32_Raw := params["int32"]
			v, err2 := strconv.ParseInt(int32_Raw, 10, 32)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError("int32", int32_Raw, "integer"))
			}
			int32_ = int32(v)
		}
		{
			int64_Raw := params["int64"]
			v, err2 := strconv.ParseInt(int64_Raw, 10, 64)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError("int64", int64_Raw, "integer"))
			}
			int64_ = v
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodAPayload(int_, int32_, int64_)

		return payload, nil
	}
}
`


