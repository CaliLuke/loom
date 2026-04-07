package testdata


var QueryArrayAliasDecodeCode = `// DecodeMethodARequest returns a decoder for requests sent to the
// ServiceQueryArrayAlias MethodA endpoint.
func DecodeMethodARequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			array []uint
			err   error
		)
		{
			arrayRaw := r.URL.Query()["array"]
			if arrayRaw != nil {
				array = make([]uint, len(arrayRaw))
				for i, rv := range arrayRaw {
					v, err2 := strconv.ParseUint(rv, 10, strconv.IntSize)
					if err2 != nil {
						err = loom.MergeErrors(err, loom.InvalidFieldTypeError("array", arrayRaw, "array of unsigned integers"))
					}
					array[i] = uint(v)
				}
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodAPayload(array)

		return payload, nil
	}
}
`


