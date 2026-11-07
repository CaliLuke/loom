package http

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
)

// MultipartMetadataJSON supplies a captured semantic string default from the
// sole uploaded file part. An explicit target part, even empty, wins. It never
// adds a physical part or changes closure claims. header is "filename" or
// "Content-Type"; duplicate source parts and absent/empty headers supply no value.
func MultipartMetadataJSON(form *MultipartForm, name, file, header string) (jsontext.Value, bool, error) {
	if header != "filename" && header != "Content-Type" {
		return nil, false, fmt.Errorf("invalid multipart metadata source %q", header)
	}
	if form == nil {
		return nil, false, nil
	}
	var source *MultipartPart
	for index := range form.Parts {
		part := &form.Parts[index]
		if part.Name == name {
			return nil, false, nil
		}
		if part.Name == file {
			if source != nil {
				return nil, false, nil
			}
			source = part
		}
	}
	if source == nil || source.Filename == "" {
		return nil, false, nil
	}
	value := source.Filename
	if header == "Content-Type" {
		value = source.ContentType
	}
	if value == "" {
		return nil, false, nil
	}
	raw, err := json.Marshal(value)
	return raw, err == nil, err
}
