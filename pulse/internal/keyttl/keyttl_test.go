package keyttl

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestArgs(t *testing.T) {
	cases := []struct {
		name        string
		ttl         time.Duration
		sliding     bool
		wantSeconds int64
		wantSliding string
	}{
		{name: "disabled", ttl: 0, wantSeconds: 0, wantSliding: "0"},
		{name: "negative", ttl: -time.Second, sliding: true, wantSeconds: 0, wantSliding: "1"},
		{name: "sub second rounds up", ttl: time.Millisecond, wantSeconds: 1, wantSliding: "0"},
		{name: "truncates to seconds", ttl: 1500 * time.Millisecond, wantSeconds: 1, wantSliding: "0"},
		{name: "fixed", ttl: time.Hour, wantSeconds: 3600, wantSliding: "0"},
		{name: "sliding", ttl: time.Minute, sliding: true, wantSeconds: 60, wantSliding: "1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			seconds, sliding := Args(tc.ttl, tc.sliding)
			assert.Equal(t, tc.wantSeconds, seconds)
			assert.Equal(t, tc.wantSliding, sliding)
		})
	}
}
