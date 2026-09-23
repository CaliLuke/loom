package pool

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"
)

// errMalformedMessage is the panic value cause for pool stream payloads that do
// not match the length-prefixed wire format.
var errMalformedMessage = errors.New("malformed pool message")

// marshalJob marshals a job into a byte slice.
func marshalJob(job *Job) []byte {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, int32(len(job.Key))); err != nil {
		panic(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, []byte(job.Key)); err != nil {
		panic(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, int32(len(job.NodeID))); err != nil {
		panic(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, []byte(job.NodeID)); err != nil {
		panic(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, int32(len(job.Payload))); err != nil {
		panic(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, job.Payload); err != nil {
		panic(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, job.CreatedAt.UnixNano()); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// unmarshalJob unmarshals a job from a byte slice created by marshalJob.
func unmarshalJob(data []byte) *Job {
	reader := bytes.NewReader(data)
	key := readField(reader, "job key")
	nodeID := readField(reader, "job node ID")
	payload := readField(reader, "job payload")
	if len(payload) == 0 {
		payload = nil
	}
	var createdAtTimestamp int64
	if err := binary.Read(reader, binary.LittleEndian, &createdAtTimestamp); err != nil {
		panic(fmt.Errorf("%w: read job creation time: %w", errMalformedMessage, err))
	}
	return &Job{
		Key:       string(key),
		Payload:   payload,
		CreatedAt: time.Unix(0, createdAtTimestamp).UTC(),
		NodeID:    string(nodeID),
	}
}

// marshalJobKey marshals a job key into a byte slice.
func marshalJobKey(key string) []byte {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, int32(len(key))); err != nil {
		panic(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, []byte(key)); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// unmarshalJobKey reads the leading job key of a job, job key, or
// notification payload.
func unmarshalJobKey(data []byte) string {
	return string(readField(bytes.NewReader(data), "job key"))
}

// unmarshalJobKeyAndNodeID reads the leading job key and node ID of a job
// payload.
func unmarshalJobKeyAndNodeID(data []byte) (string, string) {
	reader := bytes.NewReader(data)
	key := readField(reader, "job key")
	nodeID := readField(reader, "job node ID")
	return string(key), string(nodeID)
}

func marshalNotification(key string, payload []byte) []byte {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, int32(len(key))); err != nil {
		panic(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, []byte(key)); err != nil {
		panic(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, int32(len(payload))); err != nil {
		panic(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, payload); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// unmarshalNotification unmarshals a notification created by
// marshalNotification.
func unmarshalNotification(data []byte) (string, []byte) {
	reader := bytes.NewReader(data)
	key := readField(reader, "notification key")
	payload := readField(reader, "notification payload")
	return string(key), payload
}

// Envelope used to identify event sender.
func marshalEnvelope(sender string, payload []byte) []byte {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, int32(len(sender))); err != nil {
		panic(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, []byte(sender)); err != nil {
		panic(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, int32(len(payload))); err != nil {
		panic(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, payload); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// unmarshalEnvelope unmarshals an envelope from a byte slice created by marshalEnvelope.
func unmarshalEnvelope(data []byte) (string, []byte) {
	reader := bytes.NewReader(data)
	sender := readField(reader, "envelope sender")
	payload := readField(reader, "envelope payload")
	if len(payload) == 0 {
		payload = nil
	}
	return string(sender), payload
}

// marshalAck marshals an ack into a byte slice.
func marshalAck(ak *ack) []byte {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, int32(len(ak.EventID))); err != nil {
		panic(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, []byte(ak.EventID)); err != nil {
		panic(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, int32(len(ak.Error))); err != nil {
		panic(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, []byte(ak.Error)); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// unmarshalAck unmarshals an ack from a byte slice created by marshalAck.
func unmarshalAck(data []byte) *ack {
	reader := bytes.NewReader(data)
	eventID := readField(reader, "ack event ID")
	errorText := readField(reader, "ack error")
	return &ack{
		EventID: string(eventID),
		Error:   string(errorText),
	}
}

// readField reads one little-endian int32 length-prefixed field. It panics
// with an error wrapping errMalformedMessage when the prefix is truncated,
// negative, or larger than the remaining input, so a corrupt stream entry can
// never trigger an allocation beyond the message size.
func readField(reader *bytes.Reader, field string) []byte {
	var length int32
	if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
		panic(fmt.Errorf("%w: read %s length: %w", errMalformedMessage, field, err))
	}
	if length < 0 || int64(length) > int64(reader.Len()) {
		panic(fmt.Errorf("%w: %s length %d exceeds %d remaining bytes", errMalformedMessage, field, length, reader.Len()))
	}
	value := make([]byte, length)
	if _, err := io.ReadFull(reader, value); err != nil {
		panic(fmt.Errorf("%w: read %s: %w", errMalformedMessage, field, err))
	}
	return value
}
