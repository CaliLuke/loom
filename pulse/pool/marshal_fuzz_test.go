package pool

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

// FuzzPoolMessageRoundTrip checks that every pool wire message decodes back to
// the values it was encoded from.
func FuzzPoolMessageRoundTrip(f *testing.F) {
	f.Add("tenant:sync", "node-1", []byte(`{"force":true}`), int64(123000000456))
	f.Add("", "", []byte(nil), int64(0))
	f.Add("k\x00\xff", "n\n", []byte{0, 0, 0, 0}, int64(-1))
	f.Add("job", "node", []byte{0xff, 0xff, 0xff, 0xff}, int64(1<<62))
	f.Fuzz(func(t *testing.T, key, nodeID string, payload []byte, createdAt int64) {
		job, err := unmarshalJob(marshalJob(&Job{
			Key:       key,
			NodeID:    nodeID,
			Payload:   payload,
			CreatedAt: time.Unix(0, createdAt),
		}))
		if err != nil {
			t.Fatalf("job round trip: %v", err)
		}
		if job.Key != key || job.NodeID != nodeID || !bytes.Equal(job.Payload, payload) {
			t.Errorf("job round trip = %q/%q/%q, want %q/%q/%q", job.Key, job.NodeID, job.Payload, key, nodeID, payload)
		}
		if len(payload) == 0 && job.Payload != nil {
			t.Errorf("empty job payload decoded as %#v, want nil", job.Payload)
		}
		if job.CreatedAt.UnixNano() != createdAt || job.CreatedAt.Location() != time.UTC {
			t.Errorf("job created at = %v, want %d UTC", job.CreatedAt, createdAt)
		}
		encodedJob := marshalJob(&Job{Key: key, NodeID: nodeID, Payload: payload})
		if got, err := unmarshalJobKey(encodedJob); err != nil || got != key {
			t.Errorf("job key prefix = %q/%v, want %q", got, err, key)
		}
		if gotKey, gotNode, err := unmarshalJobKeyAndNodeID(encodedJob); err != nil || gotKey != key || gotNode != nodeID {
			t.Errorf("job key/node prefix = %q/%q/%v, want %q/%q", gotKey, gotNode, err, key, nodeID)
		}
		if got, err := unmarshalJobKey(marshalJobKey(key)); err != nil || got != key {
			t.Errorf("job key round trip = %q/%v, want %q", got, err, key)
		}
		notificationKey, notificationPayload, err := unmarshalNotification(marshalNotification(key, payload))
		if err != nil || notificationKey != key || !bytes.Equal(notificationPayload, payload) || notificationPayload == nil {
			t.Errorf("notification round trip = %q/%#v/%v, want %q/%q", notificationKey, notificationPayload, err, key, payload)
		}
		if got, err := unmarshalJobKey(marshalNotification(key, payload)); err != nil || got != key {
			t.Errorf("notification key prefix = %q/%v, want %q", got, err, key)
		}
		sender, envelopePayload, err := unmarshalEnvelope(marshalEnvelope(nodeID, payload))
		if err != nil || sender != nodeID || !bytes.Equal(envelopePayload, payload) || len(payload) == 0 && envelopePayload != nil {
			t.Errorf("envelope round trip = %q/%#v/%v, want %q/%q", sender, envelopePayload, err, nodeID, payload)
		}
		decodedAck, err := unmarshalAck(marshalAck(&ack{EventID: key, Error: nodeID}))
		if err != nil || decodedAck.EventID != key || decodedAck.Error != nodeID {
			t.Errorf("ack round trip = %+v/%v, want %q/%q", decodedAck, err, key, nodeID)
		}
	})
}

// FuzzUnmarshalPoolMessage checks that arbitrary stream payloads either decode
// or return an error wrapping errMalformedMessage, and never panic.
func FuzzUnmarshalPoolMessage(f *testing.F) {
	f.Add(marshalJob(&Job{Key: "k", NodeID: "n", Payload: []byte("p"), CreatedAt: time.Unix(0, 1)}))
	f.Add(marshalEnvelope("worker-1", marshalAck(&ack{EventID: "1-0", Error: "failed"})))
	f.Add(marshalNotification("job", nil))
	f.Add([]byte{})
	f.Add([]byte{0xff, 0xff, 0xff, 0xff})
	f.Add([]byte{0xff, 0xff, 0xff, 0x7f, 'a'})
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 0x70})
	decoders := map[string]func([]byte) error{
		"job":              decodeJob,
		"job key":          decodeJobKey,
		"job key and node": decodeJobKeyAndNodeID,
		"notification":     decodeNotification,
		"envelope":         decodeEnvelope,
		"ack":              decodeAck,
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		for name, decode := range decoders {
			if err := decode(data); err != nil && !errors.Is(err, errMalformedMessage) {
				t.Errorf("%s decoder returned %v for %x", name, err, data)
			}
		}
	})
}
