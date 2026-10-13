package log

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
)

type (
	// jsonEntry keeps ordered fields while the standard encoder owns JSON syntax.
	jsonEntry Entry

	// jsonEntryWriter records the first encoder error across recursive values.
	jsonEntryWriter struct {
		encoder *jsontext.Encoder
		err     error
	}
)

// MarshalJSONTo writes ordered fields, retaining duplicate names and containing
// unsupported complex values as diagnostic strings instead of losing the entry.
func (e *jsonEntry) MarshalJSONTo(encoder *jsontext.Encoder) error {
	w := jsonEntryWriter{encoder: encoder}
	w.token(jsontext.BeginObject)
	w.field(TimestampKey, e.Time.Format(TimestampFormatLayout))
	w.field(SeverityKey, e.Severity.String())
	for _, kv := range e.KeyVals {
		w.field(kv.K, kv.V)
	}
	w.token(jsontext.EndObject)
	return w.err
}

func (w *jsonEntryWriter) field(key string, value any) {
	w.token(jsontext.String(key))
	w.value(value)
}

func (w *jsonEntryWriter) token(token jsontext.Token) {
	if w.err == nil {
		w.err = w.encoder.WriteToken(token)
	}
}

func (w *jsonEntryWriter) value(value any) {
	if w.err != nil {
		return
	}
	switch v := value.(type) {
	case string:
		w.token(jsontext.String(v))
	case int:
		w.token(jsontext.Int(int64(v)))
	case int32:
		w.token(jsontext.Int(int64(v)))
	case int64:
		w.token(jsontext.Int(v))
	case uint:
		w.token(jsontext.Uint(uint64(v)))
	case uint32:
		w.token(jsontext.Uint(uint64(v)))
	case uint64:
		w.token(jsontext.Uint(v))
	case float32:
		w.token(jsontext.Float32(v))
	case float64:
		w.token(jsontext.Float(v))
	case bool:
		w.token(jsontext.Bool(v))
	case []any:
		if err := w.encoder.WriteToken(jsontext.BeginArray); err != nil {
			w.invalid(err)
			return
		}
		for _, element := range v {
			w.value(element)
		}
		w.token(jsontext.EndArray)
	case time.Duration:
		w.token(jsontext.String(v.String()))
	case codes.Code:
		w.token(jsontext.String(v.String()))
	default:
		encoded, err := json.Marshal(v)
		if err != nil {
			w.invalid(err)
			return
		}
		if err := w.encoder.WriteValue(encoded); err != nil {
			w.invalid(err)
		}
	}
}

func (w *jsonEntryWriter) invalid(err error) {
	// Rejected tokens/raw values leave the standard encoder state unchanged.
	// A diagnostic scalar fits at the current depth, including inside a full array.
	w.token(jsontext.String(fmt.Sprintf("<invalid: %s>", err)))
}
