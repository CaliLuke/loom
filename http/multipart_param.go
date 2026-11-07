package http

// DecodeMultipartParam reads a captured style grammar from literal part values.
// Unlike URL form fields, multipart names and values undergo no URI decoding.
func DecodeMultipartParam(form *MultipartForm, name string, codec ParamCodec, kind ParamKind, siblings ParamNamespace) (ParamValue, error) {
	return decodePairsParam(multipartPairs(form), name, codec, kind, siblings, paramNoUnescape)
}

// AddParam appends one styled binding atomically. Data that collides with a
// literal collection delimiter is unspellable and rejected before publication.
func (f *MultipartForm) AddParam(name, media string, codec ParamCodec, value ParamValue, siblings ParamNamespace) error {
	pairs, err := encodeParamPairs(name, codec, value, siblings, true)
	if err != nil {
		return err
	}
	for _, pair := range pairs {
		f.Parts = append(f.Parts, MultipartPart{Name: pair.name, ContentType: media, Data: []byte(pair.value)})
	}
	return nil
}

// MultipartFieldPresent reports whether any physical part belongs to the
// captured binding, excluding sibling ownership before applying residual claims.
func MultipartFieldPresent(form *MultipartForm, keys FormKeySet, siblings ParamNamespace) bool {
	if form == nil {
		return false
	}
	for _, part := range form.Parts {
		if paramSiblingOwns(siblings, part.Name) {
			continue
		}
		if keys.Residual || paramSiblingOwns(ParamNamespace{Literals: keys.Literals, Prefixes: keys.Prefixes}, part.Name) {
			return true
		}
	}
	return false
}

func multipartPairs(form *MultipartForm) RawPairs {
	pairs := RawPairs{hasInput: form != nil}
	if form != nil {
		for _, part := range form.Parts {
			pairs.pairs = append(pairs.pairs, rawParamPair{name: part.Name, value: string(part.Data)})
		}
	}
	return pairs
}
