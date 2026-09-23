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
		job := unmarshalJob(marshalJob(&Job{
			Key:       key,
			NodeID:    nodeID,
			Payload:   payload,
			CreatedAt: time.Unix(0, createdAt),
		}))
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
		if got := unmarshalJobKey(encodedJob); got != key {
			t.Errorf("job key prefix = %q, want %q", got, key)
		}
		if gotKey, gotNode := unmarshalJobKeyAndNodeID(encodedJob); gotKey != key || gotNode != nodeID {
			t.Errorf("job key/node prefix = %q/%q, want %q/%q", gotKey, gotNode, key, nodeID)
		}
		if got := unmarshalJobKey(marshalJobKey(key)); got != key {
			t.Errorf("job key round trip = %q, want %q", got, key)
		}
		notificationKey, notificationPayload := unmarshalNotification(marshalNotification(key, payload))
		if notificationKey != key || !bytes.Equal(notificationPayload, payload) || notificationPayload == nil {
			t.Errorf("notification round trip = %q/%#v, want %q/%q", notificationKey, notificationPayload, key, payload)
		}
		if got := unmarshalJobKey(marshalNotification(key, payload)); got != key {
			t.Errorf("notification key prefix = %q, want %q", got, key)
		}
		sender, envelopePayload := unmarshalEnvelope(marshalEnvelope(nodeID, payload))
		if sender != nodeID || !bytes.Equal(envelopePayload, payload) || len(payload) == 0 && envelopePayload != nil {
			t.Errorf("envelope round trip = %q/%#v, want %q/%q", sender, envelopePayload, nodeID, payload)
		}
		decodedAck := unmarshalAck(marshalAck(&ack{EventID: key, Error: nodeID}))
		if decodedAck.EventID != key || decodedAck.Error != nodeID {
			t.Errorf("ack round trip = %+v, want %q/%q", decodedAck, key, nodeID)
		}
	})
}

// FuzzUnmarshalPoolMessage checks that arbitrary stream payloads either decode
// or panic with errMalformedMessage, and never allocate beyond their size.
func FuzzUnmarshalPoolMessage(f *testing.F) {
	f.Add(marshalJob(&Job{Key: "k", NodeID: "n", Payload: []byte("p"), CreatedAt: time.Unix(0, 1)}))
	f.Add(marshalEnvelope("worker-1", marshalAck(&ack{EventID: "1-0", Error: "failed"})))
	f.Add(marshalNotification("job", nil))
	f.Add([]byte{})
	f.Add([]byte{0xff, 0xff, 0xff, 0xff})
	f.Add([]byte{0xff, 0xff, 0xff, 0x7f, 'a'})
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 0x70})
	decoders := map[string]func([]byte){
		"job":              func(data []byte) { unmarshalJob(data) },
		"job key":          func(data []byte) { unmarshalJobKey(data) },
		"job key and node": func(data []byte) { unmarshalJobKeyAndNodeID(data) },
		"notification":     func(data []byte) { unmarshalNotification(data) },
		"envelope":         func(data []byte) { unmarshalEnvelope(data) },
		"ack":              func(data []byte) { unmarshalAck(data) },
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		for name, decode := range decoders {
			if recovered := recoverDecodePanic(func() { decode(data) }); recovered != nil {
				err, ok := recovered.(error)
				if !ok || !errors.Is(err, errMalformedMessage) {
					t.Errorf("%s decoder panicked with %v for %x", name, recovered, data)
				}
			}
		}
	})
}

func recoverDecodePanic(decode func()) (recovered any) {
	defer func() {
		recovered = recover()
	}()
	decode()
	return nil
}
