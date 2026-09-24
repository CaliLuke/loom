package pool

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/streaming"
)

// TestDispatchReturnBeforeRegistration covers a dispatch return that reaches
// the dispatching node before dispatchJob has registered its channel for the
// start event: the result must still reach the dispatcher.
func TestDispatchReturnBeforeRegistration(t *testing.T) {
	cases := []struct {
		name    string
		errText string
	}{
		{name: "success"},
		{name: "start error", errText: "start failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rdb := startTestRedis(t)
			node := addTestNode(t, rdb, "early-return")
			node.returnDispatchStatus(&streaming.Event{
				ID:        "2-0",
				EventName: evDispatchReturn,
				Payload:   marshalAck(&ack{EventID: "1-0", Error: tc.errText}),
			})

			cherr := node.registerDispatch("1-0")
			select {
			case err := <-cherr:
				if tc.errText == "" {
					require.NoError(t, err)
				} else {
					require.EqualError(t, err, tc.errText)
				}
			case <-time.After(time.Second):
				t.Fatal("early dispatch return was lost")
			}
		})
	}
}
