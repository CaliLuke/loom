package testdata


var QueryMapAliasDecodeCode = `// DecodeMethodARequest returns a decoder for requests sent to the
// ServiceQueryMapAlias MethodA endpoint.
func DecodeMethodARequest(mux loomhttp.Muxer, decoder func(*http.Request) loomhttp.Decoder) func(*http.Request) (any, error) {
	return func(r *http.Request) (any, error) {
		var (
			map_ map[float32]bool
			err  error
		)
		{
			map_Raw := r.URL.Query()
			if len(map_Raw) != 0 {
				for keyRaw, valRaw := range map_Raw {
					if strings.HasPrefix(keyRaw, "map[") {
						if map_ == nil {
							map_ = make(map[float32]bool)
						}
						var keya float32
						{
							openIdx := strings.IndexRune(keyRaw, '[')
							closeIdx := strings.IndexRune(keyRaw, ']')
							if openIdx == -1 || closeIdx == -1 || closeIdx <= openIdx {
								err = loom.MergeErrors(err, loom.DecodePayloadError("invalid query string: malformed brackets"))
							} else {
								keyaRaw := keyRaw[openIdx+1 : closeIdx]
								v, err2 := strconv.ParseFloat(keyaRaw, 32)
								if err2 != nil {
									err = loom.MergeErrors(err, loom.InvalidFieldTypeError("query", keyaRaw, "float"))
								}
								keya = float32(v)
							}
						}
						var vala bool
						{
							valaRaw := valRaw[0]
							v, err2 := strconv.ParseBool(valaRaw)
							if err2 != nil {
								err = loom.MergeErrors(err, loom.InvalidFieldTypeError("query", valaRaw, "boolean"))
							}
							vala = v
						}
						map_[keya] = vala
					}
				}
			}
		}
		if err != nil {
			return nil, err
		}
		payload := NewMethodAPayload(map_)

		return payload, nil
	}
}
`


