package pool

import (
	"encoding/hex"
	"hash/crc64"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	jobWireHex = "0b00000074656e616e743a73796e63060000006e6f64652d31" +
		"0e0000007b22666f726365223a747275657dc80f5fa31c000000"
	notificationWireHex = "050000006a6f622d31070000007061796c6f6164"
	envelopeWireHex     = "08000000776f726b65722d31070000007061796c6f6164"
	ackWireHex          = "03000000312d30060000006661696c6564"
	jobKeyWireHex       = "050000006a6f622d31"
)

func TestJobMarshalRoundTrip(t *testing.T) {
	createdAt := time.Unix(123, 456).UTC()
	job := &Job{
		Key:       "tenant:sync",
		Payload:   []byte(`{"force":true}`),
		CreatedAt: createdAt,
		NodeID:    "node-1",
	}

	encoded := marshalJob(job)
	require.Equal(t, mustDecodeHex(t, jobWireHex), encoded)
	decoded := unmarshalJob(encoded)

	require.Equal(t, job.Key, decoded.Key)
	require.Equal(t, job.Payload, decoded.Payload)
	require.Equal(t, job.CreatedAt, decoded.CreatedAt)
	require.Equal(t, job.NodeID, decoded.NodeID)
	require.Nil(t, decoded.Worker)
}

func TestJobMarshalRoundTripWithEmptyPayload(t *testing.T) {
	job := &Job{
		Key:       "empty",
		CreatedAt: time.Unix(0, 1).UTC(),
		NodeID:    "node-1",
	}

	decoded := unmarshalJob(marshalJob(job))

	require.Equal(t, job.Key, decoded.Key)
	require.Nil(t, decoded.Payload)
	require.Equal(t, job.CreatedAt, decoded.CreatedAt)
	require.Equal(t, job.NodeID, decoded.NodeID)
}

func TestJobKeyMarshalHelpers(t *testing.T) {
	encoded := marshalJobKey("job-1")
	require.Equal(t, mustDecodeHex(t, jobKeyWireHex), encoded)
	require.Equal(t, "job-1", unmarshalJobKey(encoded))

	key, nodeID := unmarshalJobKeyAndNodeID(marshalJob(&Job{
		Key:       "job-2",
		NodeID:    "node-2",
		CreatedAt: time.Unix(0, 2).UTC(),
	}))
	require.Equal(t, "job-2", key)
	require.Equal(t, "node-2", nodeID)
}

func TestNotificationMarshalRoundTrip(t *testing.T) {
	encoded := marshalNotification("job-1", []byte("payload"))
	require.Equal(t, mustDecodeHex(t, notificationWireHex), encoded)
	key, payload := unmarshalNotification(encoded)
	require.Equal(t, "job-1", key)
	require.Equal(t, []byte("payload"), payload)

	key, payload = unmarshalNotification(marshalNotification("job-1", nil))
	require.Equal(t, "job-1", key)
	require.Empty(t, payload)
}

func TestEnvelopeMarshalRoundTrip(t *testing.T) {
	encoded := marshalEnvelope("worker-1", []byte("payload"))
	require.Equal(t, mustDecodeHex(t, envelopeWireHex), encoded)
	sender, payload := unmarshalEnvelope(encoded)
	require.Equal(t, "worker-1", sender)
	require.Equal(t, []byte("payload"), payload)

	sender, payload = unmarshalEnvelope(marshalEnvelope("worker-1", nil))
	require.Equal(t, "worker-1", sender)
	require.Nil(t, payload)
}

func TestAckMarshalRoundTrip(t *testing.T) {
	encoded := marshalAck(&ack{
		EventID: "1-0",
		Error:   "failed",
	})
	require.Equal(t, mustDecodeHex(t, ackWireHex), encoded)
	decoded := unmarshalAck(encoded)

	require.Equal(t, "1-0", decoded.EventID)
	require.Equal(t, "failed", decoded.Error)
}

func TestParseDispatchClaim(t *testing.T) {
	status, value, err := parseDispatchClaim([]any{dispatchClaimed, "123"})
	require.NoError(t, err)
	require.Equal(t, dispatchClaimed, status)
	require.Equal(t, "123", value)

	_, _, err = parseDispatchClaim([]any{"bad", "123"})
	require.ErrorContains(t, err, "invalid claim status")

	_, _, err = parseDispatchClaim([]any{dispatchClaimed, 123})
	require.ErrorContains(t, err, "invalid claim value")

	_, _, err = parseDispatchClaim([]any{dispatchClaimed})
	require.ErrorContains(t, err, "invalid claim result")
}

func TestPoolNameHelpers(t *testing.T) {
	require.Equal(t, "pool:node-keepalive", nodeKeepAliveMapName("pool"))
	require.Equal(t, "pool:shutdown", nodeShutdownMapName("pool"))
	require.Equal(t, "pool:workers", workerMapName("pool"))
	require.Equal(t, "pool:worker-keepalive", workerKeepAliveMapName("pool"))
	require.Equal(t, "pool:cleanup", workerCleanupMapName("pool"))
	require.Equal(t, "pool:jobs", jobMapName("pool"))
	require.Equal(t, "pool:pending-jobs", jobPendingMapName("pool"))
	require.Equal(t, "pool:job-payloads", jobPayloadMapName("pool"))
	require.Equal(t, "pool:tickers", tickerMapName("pool"))
	require.Equal(t, "pool:pool", poolStreamName("pool"))
	require.Equal(t, "pool:node:node-1", nodeStreamName("pool", "node-1"))
	require.Equal(t, "worker:worker-1", workerStreamName("worker-1"))
	require.Equal(t, "map:pool:jobs:content", rmapContentKey("pool:jobs"))
	require.Equal(t, "map:pool:jobs:updates", rmapUpdateChannel("pool:jobs"))
	require.Equal(t, "worker-1:1-0", pendingEventKey("worker-1", "1-0"))
}

func TestJumpHashIsStableAndInRange(t *testing.T) {
	jh := &jumpHash{h: crc64.New(crc64.MakeTable(crc64.ISO))}

	first := jh.Hash("tenant:sync", 17)
	second := jh.Hash("tenant:sync", 17)

	require.Equal(t, first, second)
	require.GreaterOrEqual(t, first, int64(0))
	require.Less(t, first, int64(17))
}

func mustDecodeHex(t *testing.T, value string) []byte {
	t.Helper()
	decoded, err := hex.DecodeString(value)
	require.NoError(t, err)
	return decoded
}
