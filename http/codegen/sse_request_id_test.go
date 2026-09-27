package codegen

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/stretchr/testify/require"
)

// TestSSERequestIDGeneratedField resolves payload field names and pointer semantics.
func TestSSERequestIDGeneratedField(t *testing.T) {
	for _, tc := range []struct {
		name, field, want string
		required          bool
	}{
		{"snake_case", "", "LastEventID", false},
		{"custom name", "ResumeToken", "ResumeToken", false},
		{"required custom name", "ResumeToken", "ResumeToken", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := RunHTTPDSL(t, func() {
				Service("events", func() {
					Method("watch", func() {
						Payload(func() {
							Attribute("last_event_id", String, func() {
								if tc.field != "" {
									Meta("struct:field:name", tc.field)
								}
							})
							if tc.required {
								Required("last_event_id")
							}
						})
						StreamingResult(String)
						HTTP(func() {
							GET("/events")
							ServerSentEvents(func() {
								SSERequestID("last_event_id")
							})
						})
					})
				})
			})
			data := CreateHTTPServices(root).Get("events").Endpoint("watch").SSE
			require.Equal(t, tc.want, data.RequestIDField)
			require.Equal(t, !tc.required, data.RequestIDPointer)
		})
	}
}
