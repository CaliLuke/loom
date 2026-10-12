package http

import (
	"fmt"
	"math"
)

// WebSocketConfig configures one physical WebSocket connection. Its zero value
// uses DefaultMaxRequestBodyBytes and no connection hook. The expanded message
// limit applies even when the hook changes Gorilla's framing limit.
type WebSocketConfig struct {
	maxMessageBytes int64
	configure       ConnConfigureFunc
}

// NewWebSocketConfig validates maxMessageBytes and configures the hook run by
// WebSocketStream.SetConn. The limit must be positive and less than MaxInt64,
// leaving room for the overflow probe used when reading a message.
func NewWebSocketConfig(maxMessageBytes int64, configure ConnConfigureFunc) (WebSocketConfig, error) {
	if maxMessageBytes <= 0 || maxMessageBytes == math.MaxInt64 {
		return WebSocketConfig{}, fmt.Errorf("loom http websocket message limit must be positive and less than MaxInt64")
	}
	return WebSocketConfig{maxMessageBytes: maxMessageBytes, configure: configure}, nil
}

// MaxMessageBytes returns the maximum expanded size of one inbound message.
// A zero configuration uses DefaultMaxRequestBodyBytes.
func (c WebSocketConfig) MaxMessageBytes() int64 {
	if c.maxMessageBytes == 0 {
		return DefaultMaxRequestBodyBytes
	}
	return c.maxMessageBytes
}
