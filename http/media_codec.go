package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"maps"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"reflect"
	"slices"

	"github.com/CaliLuke/loom/internal/httpcodec"
	loom "github.com/CaliLuke/loom/pkg"
)

type mediaCodecError struct{ err error }

func (e *mediaCodecError) Encode(any) error {
	return e.err
}
func (e *mediaCodecError) Decode(any) error {
	return e.err
}

func mediaRequestDecoder(request *http.Request) Decoder {
	rejectNull, _ := request.Context().Value(nonNullableBodyKey{}).(bool)
	return mediaDecoder(request.Body, request.Header.Get("Content-Type"), decodeRequest, RequestBodyLimit(request.Context()), !rejectNull, mediaFormCodec(request.Context()), mediaMultipartCodec(request.Context()))
}

func mediaDecoder(reader io.Reader, media string, mode decodeMode, limit int64, allowsNull bool, form MediaFormCodec, multipartCodec MediaMultipartCodec) Decoder {
	base, _, err := mime.ParseMediaType(media)
	if err != nil {
		return newUnsupportedDecoder(media)
	}
	switch base {
	case "application/octet-stream":
		return mediaRawDecoder(reader, media, mode, limit, false)
	case "application/x-www-form-urlencoded":
		return newLimitedDecoder(reader, func(data []byte, target any) error {
			values, err := url.ParseQuery(string(data))
			if err != nil {
				return err
			}
			if form.Decode != nil {
				pairs, err := ParseRawPairs(string(data))
				if err != nil {
					return err
				}
				return form.Decode(pairs, values)
			}
			return decodeMediaFormValues(values, target)
		}, mode, limit)
	case "multipart/form-data":
		return EncodingFunc(func(target any) error {
			request := &http.Request{Header: http.Header{"Content-Type": {media}}, Body: io.NopCloser(reader)}
			form, err := ReadMultipartFormWithLimit(request, limit)
			if err != nil {
				return requestBodyDecodeError(err, mode)
			}
			if multipartCodec.Decode != nil {
				return multipartCodec.Decode(form)
			}
			values := make(url.Values)
			files := make(map[string]bool)
			for _, part := range form.Parts {
				if files[part.Name] || part.Filename != "" && len(values[part.Name]) > 0 {
					return fmt.Errorf("duplicate multipart field %s", part.Name)
				}
				files[part.Name] = part.Filename != ""
				values.Add(part.Name, string(part.Data))
			}
			return decodeMediaFormValues(values, target)
		})
	}
	switch httpcodec.RequestContentType(media).Kind {
	case httpcodec.JSON:
		decode := decodeJSON
		if !allowsNull {
			decode = decodeNonNullableJSON
		}
		return newLimitedDecoder(reader, decode, mode, limit)
	case httpcodec.Text:
		return mediaRawDecoder(reader, media, mode, limit, true)
	default:
		return newUnsupportedDecoder(media)
	}
}

func mediaRawDecoder(reader io.Reader, media string, mode decodeMode, limit int64, text bool) Decoder {
	return EncodingFunc(func(target any) error {
		value := reflect.ValueOf(target)
		if value.Kind() != reflect.Pointer || value.IsNil() {
			return fmt.Errorf("cannot decode %s into %T", media, target)
		}
		value = value.Elem()
		data, err := readAllLimited(reader, limit)
		if err != nil {
			return requestBodyDecodeError(err, mode)
		}
		switch {
		case text && value.Kind() == reflect.String:
			value.SetString(string(data))
		case value.Kind() == reflect.Slice && value.Type().Elem().Kind() == reflect.Uint8:
			value.SetBytes(data)
		default:
			return fmt.Errorf("cannot decode %s into %T", media, target)
		}
		return nil
	})
}

func mediaRequestEncoder(request *http.Request) Encoder {
	return EncodingFunc(func(value any) error {
		var buffer bytes.Buffer
		encoder, media, err := mediaEncoder(&buffer, request.Header.Get("Content-Type"), mediaPartTypes(request.Context()), mediaFormCodec(request.Context()), mediaMultipartCodec(request.Context()))
		if err != nil {
			return err
		}
		if err = encoder.Encode(value); err != nil {
			return err
		}
		data := buffer.Bytes()
		request.Header.Set("Content-Type", media)
		request.Body = io.NopCloser(bytes.NewReader(data))
		request.ContentLength = int64(len(data))
		request.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(data)), nil
		}
		return nil
	})
}

func mediaResponseEncoder(ctx context.Context, writer http.ResponseWriter) Encoder {
	encoder, media, err := mediaEncoder(writer, stringContextValue(ctx, ContentTypeKey), mediaPartTypes(ctx), mediaFormCodec(ctx), mediaMultipartCodec(ctx))
	if err != nil {
		return &mediaCodecError{err: err}
	}
	writer.Header().Set("Content-Type", media)
	return encoder
}

func mediaEncoder(writer io.Writer, media string, parts map[string]string, form MediaFormCodec, multipartCodec MediaMultipartCodec) (Encoder, string, error) {
	base, params, err := mime.ParseMediaType(media)
	if err != nil {
		return nil, "", loom.UnsupportedMediaTypeError(media)
	}
	switch base {
	case "application/octet-stream":
		return mediaRawEncoder(writer, media, false), media, nil
	case "application/x-www-form-urlencoded":
		if form.Encode != nil {
			encoded, err := form.Encode()
			if err != nil {
				return nil, "", err
			}
			return EncodingFunc(func(any) error {
				return writeMediaBytes(writer, []byte(encoded))
			}), media, nil
		}
		return EncodingFunc(func(value any) error {
			values, err := EncodeFormValues(value)
			if err != nil {
				return err
			}
			return writeMediaBytes(writer, []byte(values.Encode()))
		}), media, nil
	case "multipart/form-data":
		multipartWriter := multipart.NewWriter(writer)
		if boundary := params["boundary"]; boundary != "" {
			if err := multipartWriter.SetBoundary(boundary); err != nil {
				return nil, "", err
			}
		}
		params["boundary"] = multipartWriter.Boundary()
		if multipartCodec.Encode != nil {
			captured, err := multipartCodec.Encode()
			if err != nil {
				return nil, "", err
			}
			return EncodingFunc(func(any) error {
				return WriteMultipartForm(multipartWriter, captured)
			}), mime.FormatMediaType(base, params), nil
		}
		return EncodingFunc(func(value any) error {
			return encodeMultipartMedia(multipartWriter, parts, value)
		}), mime.FormatMediaType(base, params), nil
	}
	switch httpcodec.RequestContentType(media).Kind {
	case httpcodec.JSON:
		return newJSONResponseEncoder(writer), media, nil
	case httpcodec.Text:
		return mediaRawEncoder(writer, media, true), media, nil
	default:
		return nil, "", loom.UnsupportedMediaTypeError(media)
	}
}

func mediaRawEncoder(writer io.Writer, media string, text bool) Encoder {
	return EncodingFunc(func(input any) error {
		value := reflect.ValueOf(input)
		for value.Kind() == reflect.Pointer {
			if value.IsNil() {
				return fmt.Errorf("cannot encode nil as %s", media)
			}
			value = value.Elem()
		}
		switch {
		case text && value.Kind() == reflect.String:
			return writeMediaBytes(writer, []byte(value.String()))
		case value.Kind() == reflect.Slice && value.Type().Elem().Kind() == reflect.Uint8:
			return writeMediaBytes(writer, value.Bytes())
		default:
			return fmt.Errorf("cannot encode %T as %s", input, media)
		}
	})
}

func writeMediaBytes(writer io.Writer, data []byte) error {
	written, err := writer.Write(data)
	if err != nil {
		return err
	}
	if written != len(data) {
		return io.ErrShortWrite
	}
	return nil
}

// A selected form representation supplies an object even when all of its
// optional fields are absent. Keep this allocation separate from ordinary
// form decoding, where an absent root may still be represented by nil.
func decodeMediaFormValues(values url.Values, target any) error {
	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return fmt.Errorf("media form decode target must be a non-nil pointer")
	}
	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			value.Set(reflect.New(value.Type().Elem()))
		}
		value = value.Elem()
	}
	return DecodeFormValues(values, target)
}

func encodeMultipartMedia(multipartWriter *multipart.Writer, parts map[string]string, value any) error {
	values, err := EncodeFormValues(value)
	if err != nil {
		return err
	}
	for _, name := range slices.Sorted(maps.Keys(values)) {
		for _, field := range values[name] {
			partType := parts[name]
			if partType == "" {
				partType = "text/plain"
			}
			header := textproto.MIMEHeader{}
			header.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{"name": name}))
			header.Set("Content-Type", partType)
			part, err := multipartWriter.CreatePart(header)
			if err != nil {
				return err
			}
			if err := writeMediaBytes(part, []byte(field)); err != nil {
				return err
			}
		}
	}
	return multipartWriter.Close()
}
