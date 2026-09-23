package pool

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUnmarshalMalformedPoolMessagesReturnsBoundedError(t *testing.T) {
	type decoder struct {
		name   string
		decode func([]byte) error
		fields int
	}
	decoders := []decoder{
		{name: "job", decode: decodeJob, fields: 4},
		{name: "job key", decode: decodeJobKey, fields: 1},
		{name: "job key and node", decode: decodeJobKeyAndNodeID, fields: 2},
		{name: "notification", decode: decodeNotification, fields: 2},
		{name: "envelope", decode: decodeEnvelope, fields: 2},
		{name: "ack", decode: decodeAck, fields: 2},
	}
	inputs := []struct {
		name      string
		data      []byte
		minFields int
		bounded   bool
	}{
		{name: "empty", data: nil, minFields: 1},
		{name: "short length", data: []byte{1, 0}, minFields: 1},
		{name: "negative length", data: []byte{0xff, 0xff, 0xff, 0xff}, minFields: 1},
		{name: "min int32 length", data: []byte{0x00, 0x00, 0x00, 0x80}, minFields: 1},
		{name: "length exceeds data", data: []byte{0xff, 0xff, 0xff, 0x7f, 'a'}, minFields: 1},
		{name: "truncated field", data: []byte{5, 0, 0, 0, 'a', 'b'}, minFields: 1},
		{name: "missing second length", data: []byte{1, 0, 0, 0, 'a'}, minFields: 2},
		{name: "negative second length", data: []byte{1, 0, 0, 0, 'a', 0xfe, 0xff, 0xff, 0xff}, minFields: 2},
		{name: "huge second length", data: []byte{0, 0, 0, 0, 0x00, 0x00, 0x00, 0x70}, minFields: 2, bounded: true},
		{name: "huge payload length", data: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0x00, 0x00, 0x00, 0x70}, minFields: 3, bounded: true},
		{name: "missing created at", data: marshalJob(&Job{Key: "k", NodeID: "n", CreatedAt: time.Unix(0, 1)})[:15], minFields: 4},
	}
	for _, decoder := range decoders {
		for _, input := range inputs {
			if decoder.fields < input.minFields {
				continue
			}
			t.Run(decoder.name+"/"+input.name, func(t *testing.T) {
				var before, after runtime.MemStats
				runtime.ReadMemStats(&before)
				err := decoder.decode(input.data)
				runtime.ReadMemStats(&after)
				require.ErrorIs(t, err, errMalformedMessage)
				if input.bounded {
					require.Less(t, after.TotalAlloc-before.TotalAlloc, uint64(1<<20))
				}
			})
		}
	}
}

func decodeJob(data []byte) error {
	_, err := unmarshalJob(data)
	return err
}

func decodeJobKey(data []byte) error {
	_, err := unmarshalJobKey(data)
	return err
}

func decodeJobKeyAndNodeID(data []byte) error {
	_, _, err := unmarshalJobKeyAndNodeID(data)
	return err
}

func decodeNotification(data []byte) error {
	_, _, err := unmarshalNotification(data)
	return err
}

func decodeEnvelope(data []byte) error {
	_, _, err := unmarshalEnvelope(data)
	return err
}

func decodeAck(data []byte) error {
	_, err := unmarshalAck(data)
	return err
}
