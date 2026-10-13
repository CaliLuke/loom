package log

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"strconv"
	"time"
	"unicode/utf8"
)

// reset escape sequence for color unset
const reset = "\033[0m"

// TimestampFormatLayout is used to set the layout for our TimestampKey
// (default "time") values.  Default format is time.RFC3339.
var TimestampFormatLayout = time.RFC3339

// epoch used to compute relative timestamps
var epoch time.Time

// FormatText is the default log formatter when not running in a terminal, it
// prints entries using the logfmt format:
//
//	time=TIME level=SEVERITY KEY=VAL KEY=VAL ...
//
// Where TIME is the UTC timestamp in RFC3339 format, SEVERITY is one of
// "debug", "info" or "error", and KEY=VAL are the entry key/value pairs.
// Names and presented values are quoted and escaped, including complex values.
// Embedded newlines remain escaped data on the same physical output line.
//
// Output can be customised with log.TimestampKey, log.TimestampFormatLayout,
// and log.SeverityKey.
func FormatText(e *Entry) []byte {
	b := make([]byte, 0, 256)

	b = appendKeyValue(b, TimestampKey, e.Time.Format(TimestampFormatLayout))
	b = append(b, ' ')
	b = appendKeyValue(b, SeverityKey, e.Severity)

	for _, kv := range e.KeyVals {
		b = append(b, ' ')
		b = appendKeyValue(b, kv.K, kv.V)
	}

	b = append(b, '\n')
	return b
}

// FormatJSON is a log formatter that prints entries using JSON. Entries are
// formatted as follows:
//
//	{
//	  "time": "TIMESTAMP", // UTC timestamp in RFC3339 format
//	  "level": "SEVERITY", // one of DEBUG, INFO or ERROR
//	  "key1": "val1",      // entry key/value pairs
//	  "key2": "val2",
//	  ...
//	}
//
// Field order and duplicate names are preserved. Nonfinite scalar floats are
// encoded as "NaN", "Infinity" or "-Infinity". Unsupported complex values are
// represented by diagnostic strings; invalid UTF-8 in names and strings is
// replaced with U+FFFD.
//
// Output can be customised with log.TimestampKey, log.TimestampFormatLayout,
// and log.SeverityKey.
func FormatJSON(e *Entry) []byte {
	b, err := json.Marshal((*jsonEntry)(e), jsontext.AllowInvalidUTF8(true), jsontext.AllowDuplicateNames(true))
	if err != nil {
		// The entry encoder contains unsupported values as diagnostic strings.
		panic(fmt.Errorf("encode log entry: %w", err))
	}
	return append(b, '\n')
}

// FormatTerminal is a log formatter that prints entries suitable for terminal
// that supports colors. It prints entries in the following format:
//
//	SEVERITY[seconds] key=val key=val ...
//
// Where SEVERITY is one of DEBG, INFO or ERRO, seconds is the number of seconds
// since the application started, message is the log message, and key=val are
// the entry key/value pairs. The severity and keys are colored according to the
// severity (gray for debug entries, blue for info entries and red for errors).
// Names and values use the same escaping as FormatText.
func FormatTerminal(e *Entry) []byte {
	b := make([]byte, 0, 256)
	b = append(b, e.Severity.Color()...)
	b = append(b, e.Severity.Code()...)
	b = append(b, reset...)
	b = fmt.Appendf(b, "[%04d]", int(e.Time.Sub(epoch)/time.Second))
	if len(e.KeyVals) > 0 {
		b = append(b, ' ')
		for i, kv := range e.KeyVals {
			b = append(b, e.Severity.Color()...)
			b = appendEscapedString(b, kv.K)
			b = append(b, reset...)
			b = append(b, '=')
			b = appendTextValue(b, kv.V)
			if i < len(e.KeyVals)-1 {
				b = append(b, ' ')
			}
		}
	}
	b = append(b, '\n')
	return b
}

func init() {
	epoch = time.Now()
}

func needsQuoting(s string) bool {
	if len(s) == 0 || !utf8.ValidString(s) {
		return true
	}
	for i := 0; i < len(s); i++ {
		if s[i] <= ' ' || s[i] == '=' || s[i] == '"' || s[i] == '\\' {
			return true
		}
	}
	return false
}

func appendKeyValue(b []byte, key string, value any) []byte {
	b = appendEscapedString(b, key)
	b = append(b, '=')
	return appendTextValue(b, value)
}

func appendTextValue(b []byte, value any) []byte {
	switch v := value.(type) {
	case string:
		return appendEscapedString(b, v)
	case int:
		return strconv.AppendInt(b, int64(v), 10)
	case int32:
		return strconv.AppendInt(b, int64(v), 10)
	case int64:
		return strconv.AppendInt(b, v, 10)
	case uint:
		return strconv.AppendUint(b, uint64(v), 10)
	case uint32:
		return strconv.AppendUint(b, uint64(v), 10)
	case uint64:
		return strconv.AppendUint(b, v, 10)
	case float32:
		return appendTextFloat(b, jsontext.Float32(v))
	case float64:
		return appendTextFloat(b, jsontext.Float(v))
	case bool:
		return strconv.AppendBool(b, v)
	case []any:
		return appendTextArray(b, v)
	default:
		// Fallback to fmt.Sprintf for complex types
		return appendEscapedString(b, fmt.Sprintf("%v", v))
	}
}

func appendEscapedString(b []byte, s string) []byte {
	if needsQuoting(s) {
		return appendJSONString(b, s)
	}
	return append(b, s...)
}

func appendTextArray(b []byte, arr []any) []byte {
	b = append(b, '[')
	for i, v := range arr {
		if i > 0 {
			b = append(b, ' ')
		}
		b = appendTextValue(b, v)
	}
	return append(b, ']')
}

func appendJSONString(b []byte, s string) []byte {
	quoted, err := json.Marshal(s, jsontext.AllowInvalidUTF8(true))
	if err != nil {
		// Strings have no custom codec; permissive UTF-8 encoding cannot fail.
		panic(fmt.Errorf("encode log string: %w", err))
	}
	return append(b, quoted...)
}

func appendTextFloat(b []byte, token jsontext.Token) []byte {
	if token.Kind() == '"' {
		return appendJSONString(b, token.String())
	}
	return append(b, token.String()...)
}
