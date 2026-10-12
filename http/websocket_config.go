package http

import (
	"fmt"
	"math"
)

type (
	// WebSocketLimits bounds inbound raw messages retained by one connection.
	// Zero fields select defaults. These limits exclude decoded Go objects,
	// allocator overhead and values retained by ReadMessage callers.
	WebSocketLimits struct {
		// MaxMessageBytes bounds one expanded message; zero means 32 MiB.
		MaxMessageBytes int64
		// MaxBufferedBytes includes queued/current-decode payload bytes and one
		// maximum in-flight message; zero means twice MaxMessageBytes.
		MaxBufferedBytes int64
		// MaxBufferedMessages bounds queued/current-decode records, including
		// empty messages; zero means 256. One additional message may be in flight.
		MaxBufferedMessages int
	}

	// WebSocketConfig configures one physical WebSocket connection. Its zero
	// value selects default limits and no hook. Expanded limits apply even if
	// the hook changes Gorilla's framing limit.
	WebSocketConfig struct {
		limits    WebSocketLimits
		configure ConnConfigureFunc
	}
)

// NewWebSocketConfig validates limits and configures the attachment hook.
// Negative limits reject. MaxMessageBytes must not exceed MaxInt64/2, and the
// total byte budget must be at least twice the resolved message maximum.
func NewWebSocketConfig(limits WebSocketLimits, configure ConnConfigureFunc) (WebSocketConfig, error) {
	if limits.MaxMessageBytes < 0 || limits.MaxMessageBytes > math.MaxInt64/2 {
		return WebSocketConfig{}, fmt.Errorf("loom http websocket message limit must be between zero and MaxInt64/2")
	}
	if limits.MaxBufferedBytes < 0 || limits.MaxBufferedMessages < 0 {
		return WebSocketConfig{}, fmt.Errorf("loom http websocket buffer limits must not be negative")
	}
	config := WebSocketConfig{limits: limits, configure: configure}
	if config.MaxBufferedBytes()-config.MaxMessageBytes() < config.MaxMessageBytes() {
		return WebSocketConfig{}, fmt.Errorf("loom http websocket byte budget must hold two maximum messages")
	}
	return config, nil
}

// MaxMessageBytes returns the maximum expanded size of one inbound message.
func (c WebSocketConfig) MaxMessageBytes() int64 {
	if c.limits.MaxMessageBytes == 0 {
		return DefaultMaxRequestBodyBytes
	}
	return c.limits.MaxMessageBytes
}

// MaxBufferedBytes returns the total raw message budget, including one flight.
func (c WebSocketConfig) MaxBufferedBytes() int64 {
	if c.limits.MaxBufferedBytes == 0 {
		return 2 * c.MaxMessageBytes()
	}
	return c.limits.MaxBufferedBytes
}

// MaxBufferedMessages returns the queued/current-decode message-count limit.
func (c WebSocketConfig) MaxBufferedMessages() int {
	if c.limits.MaxBufferedMessages == 0 {
		return 256
	}
	return c.limits.MaxBufferedMessages
}
