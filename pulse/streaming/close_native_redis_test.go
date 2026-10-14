package streaming

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/streaming/options"
)

// This readiness probe requires the disposable real Redis tier. CLIENT LIST
// establishes that Redis is executing a blocking command before Close starts.
func TestFiniteNativePollClose(t *testing.T) {
	for _, kind := range []string{"reader", "sink"} {
		for _, enabled := range []bool{false, true} {
			for _, block := range []time.Duration{0, time.Hour} {
				t.Run(fmt.Sprintf("%s/context=%t/block=%s", kind, enabled, block), func(t *testing.T) {
					srv := startTestServer(t)
					if !srv.Real() {
						t.Skip("requires disposable real Redis")
					}
					opts := *srv.Client.Options()
					opts.ContextTimeoutEnabled = enabled
					opts.ReadTimeout = -1
					opts.WriteTimeout = -1
					rdb := redis.NewClient(&opts)
					t.Cleanup(func() {
						require.NoError(t, rdb.Close())
					})
					stream := newTestStream(t, rdb, "real-close")
					var stop func()
					var command string
					if kind == "reader" {
						reader, err := stream.NewReader(t.Context(), options.WithReaderBlockDuration(block))
						require.NoError(t, err)
						reader.Subscribe()
						stop = reader.Close
						command = "xread"
					} else {
						sink, err := stream.NewSink(t.Context(), "readiness", options.WithSinkBlockDuration(block))
						require.NoError(t, err)
						sink.Subscribe()
						stop = func() {
							sink.Close(t.Context())
						}
						command = "xreadgroup"
					}
					require.Eventually(t, func() bool {
						clients, err := srv.Client.ClientList(t.Context()).Result()
						require.NoError(t, err)
						for _, line := range strings.Split(clients, "\n") {
							if strings.Contains(line, "cmd="+command+" ") && strings.Contains(line, "flags=b ") {
								return true
							}
						}
						return false
					}, time.Second, time.Millisecond)
					done := make(chan struct{})
					go func() {
						stop()
						close(done)
					}()
					second := make(chan struct{})
					go func() {
						stop()
						close(second)
					}()
					select {
					case <-done:
					case <-time.After(7 * time.Second):
						// Release parent failures so a regression never strands a worker.
						_, err := stream.Add(t.Context(), "release", []byte("payload"))
						require.NoError(t, err)
						<-done
						t.Error("Close did not join within the finite server poll allowance")
					}
					select {
					case <-second:
					case <-time.After(time.Second):
						t.Fatal("second Close did not join completed shutdown")
					}
					require.NoError(t, rdb.Ping(t.Context()).Err(), "caller client stays usable")
				})
			}
		}
	}
}
