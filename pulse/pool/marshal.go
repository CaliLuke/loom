package pool

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"
)

// errMalformedMessage is wrapped by every decoding error for pool stream
// payloads that do not match the length-prefixed wire format.
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
func unmarshalJob(data []byte) (*Job, error) {
	reader := bytes.NewReader(data)
	key, err := readField(reader, "job key")
	if err != nil {
		return nil, err
	}
	nodeID, err := readField(reader, "job node ID")
	if err != nil {
		return nil, err
	}
	payload, err := readField(reader, "job payload")
	if err != nil {
		return nil, err
	}
	if len(payload) == 0 {
		payload = nil
	}
	var createdAtTimestamp int64
	if err := binary.Read(reader, binary.LittleEndian, &createdAtTimestamp); err != nil {
		return nil, fmt.Errorf("%w: read job creation time: %w", errMalformedMessage, err)
	}
	return &Job{
		Key:       string(key),
		Payload:   payload,
		CreatedAt: time.Unix(0, createdAtTimestamp).UTC(),
		NodeID:    string(nodeID),
	}, nil
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
func unmarshalJobKey(data []byte) (string, error) {
	key, err := readField(bytes.NewReader(data), "job key")
	if err != nil {
		return "", err
	}
	return string(key), nil
}

// unmarshalJobKeyAndNodeID reads the leading job key and node ID of a job
// payload.
func unmarshalJobKeyAndNodeID(data []byte) (string, string, error) {
	reader := bytes.NewReader(data)
	key, err := readField(reader, "job key")
	if err != nil {
		return "", "", err
	}
	nodeID, err := readField(reader, "job node ID")
	if err != nil {
		return "", "", err
	}
	return string(key), string(nodeID), nil
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
func unmarshalNotification(data []byte) (string, []byte, error) {
	reader := bytes.NewReader(data)
	key, err := readField(reader, "notification key")
	if err != nil {
		return "", nil, err
	}
	payload, err := readField(reader, "notification payload")
	if err != nil {
		return "", nil, err
	}
	return string(key), payload, nil
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
func unmarshalEnvelope(data []byte) (string, []byte, error) {
	reader := bytes.NewReader(data)
	sender, err := readField(reader, "envelope sender")
	if err != nil {
		return "", nil, err
	}
	payload, err := readField(reader, "envelope payload")
	if err != nil {
		return "", nil, err
	}
	if len(payload) == 0 {
		payload = nil
	}
	return string(sender), payload, nil
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
func unmarshalAck(data []byte) (*ack, error) {
	reader := bytes.NewReader(data)
	eventID, err := readField(reader, "ack event ID")
	if err != nil {
		return nil, err
	}
	errorText, err := readField(reader, "ack error")
	if err != nil {
		return nil, err
	}
	return &ack{
		EventID: string(eventID),
		Error:   string(errorText),
	}, nil
}

// readField reads one little-endian int32 length-prefixed field. It returns
// an error wrapping errMalformedMessage when the prefix is truncated,
// negative, or larger than the remaining input, so a corrupt stream entry can
// never trigger an allocation beyond the message size.
func readField(reader *bytes.Reader, field string) ([]byte, error) {
	var length int32
	if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
		return nil, fmt.Errorf("%w: read %s length: %w", errMalformedMessage, field, err)
	}
	if length < 0 || int64(length) > int64(reader.Len()) {
		return nil, fmt.Errorf("%w: %s length %d exceeds %d remaining bytes", errMalformedMessage, field, length, reader.Len())
	}
	value := make([]byte, length)
	if _, err := io.ReadFull(reader, value); err != nil {
		return nil, fmt.Errorf("%w: read %s: %w", errMalformedMessage, field, err)
	}
	return value, nil
}
