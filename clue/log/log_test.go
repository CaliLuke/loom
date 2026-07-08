package log

import (
	"context"
	"io"
	"testing"
)

func TestLogTruncatesEntryKeyvals(t *testing.T) {
	var entries []*Entry
	maxsize := len(truncationSuffix)
	ctx := Context(context.Background(),
		WithMaxSize(maxsize),
		WithOutputs(Output{
			Writer: io.Discard,
			Format: func(e *Entry) []byte {
				copied := *e
				copied.KeyVals = append(kvList(nil), e.KeyVals...)
				entries = append(entries, &copied)
				return nil
			},
		}),
	)

	keyvals := make([]Fielder, 0, maxsize+2)
	for i := range maxsize + 2 {
		keyvals = append(keyvals, KV{K: "key", V: i})
	}
	Print(ctx, keyvals...)

	if len(entries) != 1 {
		t.Fatalf("expected one entry, got %d", len(entries))
	}
	if got, want := len(entries[0].KeyVals), maxsize+1; got != want {
		t.Fatalf("expected truncated keyvals length %d, got %d: %#v", want, got, entries[0].KeyVals)
	}
	if got, want := entries[0].KeyVals[maxsize], (KV{K: "log", V: truncationSuffix}); got != want {
		t.Fatalf("expected truncation marker %#v, got %#v", want, got)
	}
}
