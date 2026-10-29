package http

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/CaliLuke/loom/internal/httpcodec"
	loom "github.com/CaliLuke/loom/pkg"
)

type mediaContractKey struct{}
type mediaPartsKey struct{}

// SelectMediaType returns the most specific matching media contract index.
// Missing, malformed, wildcard and unmatched Content-Type values return a
// service error suitable for an HTTP 415 response. Contract parameters must
// match; additional incoming parameters are allowed.
func SelectMediaType(contentType string, contracts []string) (int, error) {
	index, err := httpcodec.SelectMediaContract(contracts, contentType)
	if err != nil {
		return -1, loom.UnsupportedMediaTypeError(contentType)
	}
	return index, nil
}

// SelectRequestMediaType selects the request contract. An optional absent body
// needs no Content-Type; a nonempty body must always identify its representation.
func SelectRequestMediaType(request *http.Request, contracts []string, optional bool) (int, error) {
	if optional && request.Header.Get("Content-Type") == "" {
		if request.Body == nil {
			return -1, io.EOF
		}
		var first [1]byte
		_, err := io.ReadFull(request.Body, first[:])
		if err == io.EOF {
			return -1, io.EOF
		}
		if err != nil {
			return -1, err
		}
	}
	return SelectMediaType(request.Header.Get("Content-Type"), contracts)
}

// EncodeMediaRequest publishes one selected typed representation through the
// supplied factory. A mapped Content-Type may refine the selection but must
// resolve to the same branch. Ranges require that concrete mapped value.
func EncodeMediaRequest(request *http.Request, index int, contracts []string, value any, parts map[string]string, factory func(*http.Request) Encoder) error {
	media, err := outgoingMediaType(request.Header.Get("Content-Type"), index, contracts)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", media)
	// Preserve the request identity: encoder factories install its Body/GetBody.
	ctx := context.WithValue(request.Context(), mediaContractKey{}, true)
	ctx = context.WithValue(ctx, mediaPartsKey{}, parts)
	*request = *request.WithContext(ctx)
	return factory(request).Encode(value)
}

// DecodeMediaRequest invokes the supplied factory for an already selected typed
// branch. The branch's null contract replaces the enclosing union's null policy.
func DecodeMediaRequest(request *http.Request, target any, allowsNull bool, parts map[string]string, factory func(*http.Request) Decoder) error {
	ctx := context.WithValue(request.Context(), mediaContractKey{}, true)
	ctx = context.WithValue(ctx, nonNullableBodyKey{}, !allowsNull)
	ctx = context.WithValue(ctx, mediaPartsKey{}, parts)
	return factory(request.WithContext(ctx)).Decode(target)
}

// MediaResponseEncoder selects a response representation before status commitment.
// Its returned encoder writes the selected value and ignores its Encode argument.
// The service chooses the variant; this function does not negotiate Accept.
func MediaResponseEncoder(ctx context.Context, writer http.ResponseWriter, index int, contracts []string, value any, parts map[string]string, factory func(context.Context, http.ResponseWriter) Encoder) (Encoder, error) {
	media, err := outgoingMediaType(writer.Header().Get("Content-Type"), index, contracts)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, ContentTypeKey, media)
	ctx = context.WithValue(ctx, mediaContractKey{}, true)
	ctx = context.WithValue(ctx, mediaPartsKey{}, parts)
	writer.Header().Set("Content-Type", media)
	encoder := factory(ctx, writer)
	if failure, ok := encoder.(*mediaCodecError); ok {
		return nil, failure.err
	}
	return EncodingFunc(func(any) error {
		return encoder.Encode(value)
	}), nil
}

// DecodeMediaResponse invokes the supplied factory for a selected response
// branch. It preserves the response body and metadata and retains request context.
func DecodeMediaResponse(response *http.Response, target any, allowsNull bool, parts map[string]string, factory func(*http.Response) Decoder) error {
	copy := *response
	request := response.Request
	if request == nil {
		request = new(http.Request)
	}
	ctx := context.WithValue(request.Context(), mediaContractKey{}, true)
	ctx = context.WithValue(ctx, nonNullableBodyKey{}, !allowsNull)
	ctx = context.WithValue(ctx, mediaPartsKey{}, parts)
	copy.Request = request.WithContext(ctx)
	return factory(&copy).Decode(target)
}

func outgoingMediaType(media string, index int, contracts []string) (string, error) {
	if index < 0 || index >= len(contracts) {
		return "", fmt.Errorf("HTTP media body has no selected variant")
	}
	if media == "" {
		media = contracts[index]
	}
	selected, err := httpcodec.SelectMediaContract(contracts, media)
	if err != nil {
		return "", err
	}
	if selected != index {
		return "", fmt.Errorf("Content-Type %q selects a different body variant", media)
	}
	return media, nil
}

func mediaContractContext(ctx context.Context) bool {
	selected, _ := ctx.Value(mediaContractKey{}).(bool)
	return selected
}

func mediaPartTypes(ctx context.Context) map[string]string {
	parts, _ := ctx.Value(mediaPartsKey{}).(map[string]string)
	return parts
}
