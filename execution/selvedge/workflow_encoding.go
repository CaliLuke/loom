package selvedge

import (
	"bytes"
	nativejson "encoding/json"
	"fmt"
	"unicode/utf8"
)

// nativeWorkflowInput preserves Selvedge's native JSON codec boundary. This
// exact file is exempt from the JSON v2 import lint: changing its codec would
// change durable input identity, marshaler dispatch and nil/field semantics.
// Other Loom-owned JSON remains on JSON v2; callers pass frozen bytes here.
type nativeWorkflowInput = nativejson.RawMessage

func encodeWorkflowInput(input any) ([]byte, error) {
	encoded, err := nativejson.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInputEncoding, err)
	}
	frozen, err := nativejson.Marshal(nativeWorkflowInput(encoded))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInputEncoding, err)
	}
	again, err := nativejson.Marshal(nativeWorkflowInput(frozen))
	if err != nil || !bytes.Equal(again, frozen) || !utf8.Valid(frozen) {
		return nil, fmt.Errorf("%w: unstable or non-UTF-8 encoding", ErrInputEncoding)
	}
	return frozen, nil
}
