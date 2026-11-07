package http

import "context"

type (
	// MediaMultipartCodec binds one captured multipart occurrence to its concrete
	// generated carrier. Callbacks belong to one exchange and one direction.
	MediaMultipartCodec struct {
		// Encode returns the complete validated part set before publication.
		Encode func() (*MultipartForm, error)
		// Decode consumes an immutable once-parsed part set within the body limit.
		Decode func(*MultipartForm) error
	}
	mediaMultipartKey struct{}
)

// WithMediaMultipartCodec supplies captured part codecs to the default media
// factories. A custom factory retains ownership of its selected representation.
func WithMediaMultipartCodec(ctx context.Context, codec MediaMultipartCodec) context.Context {
	return context.WithValue(ctx, mediaMultipartKey{}, codec)
}

func mediaMultipartCodec(ctx context.Context) MediaMultipartCodec {
	codec, _ := ctx.Value(mediaMultipartKey{}).(MediaMultipartCodec)
	return codec
}
