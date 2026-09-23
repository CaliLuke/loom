package streaming

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	redis "github.com/redis/go-redis/v9"
)

// FuzzEventFromMessage checks that arbitrary Redis stream entries decode to
// the fields Stream.Add wrote or fail with an error, and that CreatedAt never
// panics on arbitrary IDs.
func FuzzEventFromMessage(f *testing.F) {
	f.Add("1700000000123-0", "created", "payload", "topic", byte(0b111), byte(0))
	f.Add("", "", "", "", byte(0), byte(0))
	f.Add("-", "n", "p", "", byte(0b011), byte(0b100))
	f.Add("9223372036854775807-18446744073709551615", "n", "\xff", "t", byte(0b111), byte(0b001))
	f.Add("-9223372036854775808-1", "n", "p", "t", byte(0b111), byte(0b010))
	f.Add("123", "n", "p", "t", byte(0b110), byte(0))
	f.Fuzz(func(t *testing.T, id, name, payload, topic string, present, nonString byte) {
		values := map[string]any{}
		fields := []struct {
			key   string
			value string
		}{{nameKey, name}, {payloadKey, payload}, {topicKey, topic}}
		for i, field := range fields {
			if present&(1<<i) == 0 {
				continue
			}
			if nonString&(1<<i) != 0 {
				values[field.key] = []byte(field.value)
				continue
			}
			values[field.key] = field.value
		}
		stringField := func(i int) bool {
			return present&(1<<i) != 0 && nonString&(1<<i) == 0
		}
		topicOK := present&0b100 == 0 || stringField(2)

		event, err := eventFromMessage("events", streamKeyPrefix+"events", "sink", redis.XMessage{ID: id, Values: values}, nil)

		if stringField(0) && stringField(1) && topicOK {
			if err != nil {
				t.Errorf("valid entry %v returned %v", values, err)
				return
			}
			wantTopic := ""
			if stringField(2) {
				wantTopic = topic
			}
			if event.ID != id || event.EventName != name || !bytes.Equal(event.Payload, []byte(payload)) || event.Topic != wantTopic {
				t.Errorf("entry %v decoded as %+v", values, event)
			}
		} else if err == nil || event != nil {
			t.Errorf("malformed entry %v decoded as %+v, %v", values, event, err)
		}

		createdAt := (&Event{ID: id}).CreatedAt()
		millis, _, _ := strings.Cut(id, "-")
		if ms, parseErr := strconv.ParseInt(millis, 10, 64); parseErr == nil {
			if createdAt.UnixMilli() != ms {
				t.Errorf("CreatedAt(%q) = %v, want %d ms", id, createdAt, ms)
			}
		} else if createdAt.UnixMilli() != 0 {
			t.Errorf("CreatedAt(%q) = %v, want epoch", id, createdAt)
		}
	})
}
