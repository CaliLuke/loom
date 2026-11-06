package http

import "context"

type (
	// MediaFormCodec binds generated typed form fields to one selected media
	// occurrence. Encode and Decode capture that occurrence's concrete carrier;
	// only the callback for the current direction needs to be supplied.
	MediaFormCodec struct {
		// Encode returns the complete form bytes, preserving structural delimiters.
		Encode func() (string, error)
		// Decode consumes raw pairs and ordinary decoded values from the same body.
		Decode func(RawPairs, map[string][]string) error
	}
	mediaFormKey struct{}
)

// WithMediaFormCodec supplies field codecs to the default HTTP media factories.
// Custom factories retain ownership of their codec choice. The callbacks belong
// to one encoding or decoding call and must not be reused for another carrier.
func WithMediaFormCodec(ctx context.Context, codec MediaFormCodec) context.Context {
	return context.WithValue(ctx, mediaFormKey{}, codec)
}

func mediaFormCodec(ctx context.Context) MediaFormCodec {
	codec, _ := ctx.Value(mediaFormKey{}).(MediaFormCodec)
	return codec
}
