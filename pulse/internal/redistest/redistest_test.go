package redistest

import (
	"context"
	"testing"

	"github.com/CaliLuke/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestStructScriptLoadAndExecution(t *testing.T) {
	t.Setenv(AddrEnv, "")
	for _, mode := range []string{"direct", "pipeline", "transaction"} {
		t.Run(mode, func(t *testing.T) {
			client := Start(t, DBRmap).Client
			script := redis.NewScript(`return struct.pack("ic0", 3, "abc")`)
			var loaded *redis.StringCmd
			load := func(pipe redis.Pipeliner) error {
				loaded = script.Load(t.Context(), pipe)
				pipe.Ping(t.Context())
				return nil
			}
			switch mode {
			case "direct":
				loaded = script.Load(t.Context(), client)
			case "pipeline":
				_, err := client.Pipelined(t.Context(), load)
				require.NoError(t, err)
			case "transaction":
				_, err := client.TxPipelined(t.Context(), load)
				require.NoError(t, err)
			}
			require.NoError(t, loaded.Err())
			require.Equal(t, script.Hash(), loaded.Val())
			exists, err := script.Exists(t.Context(), client).Result()
			require.NoError(t, err)
			require.Equal(t, []bool{true}, exists)
			for range 2 {
				result, err := script.EvalSha(t.Context(), client, nil).Text()
				require.NoError(t, err)
				require.Equal(t, "\x03\x00\x00\x00abc", result)
			}
			plain := redis.NewScript(`return "plain"`)
			require.NoError(t, plain.Load(t.Context(), client).Err())
			result, err := plain.Run(t.Context(), client, nil).Text()
			require.NoError(t, err)
			require.Equal(t, "plain", result)
			require.NoError(t, client.ScriptFlush(t.Context()).Err())
			exists, err = script.Exists(t.Context(), client).Result()
			require.NoError(t, err)
			require.Equal(t, []bool{false}, exists)
			require.ErrorContains(t, script.EvalSha(t.Context(), client, nil).Err(), "NOSCRIPT")
			result, err = script.Run(t.Context(), client, nil).Text()
			require.NoError(t, err)
			require.Equal(t, "\x03\x00\x00\x00abc", result)
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			require.ErrorIs(t, script.Load(ctx, client).Err(), context.Canceled)
		})
	}
}

func TestCheckAddr(t *testing.T) {
	cases := []struct {
		name        string
		addr        string
		allowRemote bool
		wantErr     string
	}{
		{name: "localhost", addr: "localhost:6379"},
		{name: "ipv4 loopback", addr: "127.0.0.1:6379"},
		{name: "other ipv4 loopback", addr: "127.1.2.3:6379"},
		{name: "ipv6 loopback", addr: "[::1]:6379"},
		{name: "remote host", addr: "redis.example.com:6379", wantErr: "not a loopback address"},
		{name: "remote ip", addr: "10.0.0.5:6379", wantErr: "not a loopback address"},
		{name: "unspecified ip", addr: "0.0.0.0:6379", wantErr: "not a loopback address"},
		{name: "remote allowed", addr: "redis.example.com:6379", allowRemote: true},
		{name: "no port", addr: "localhost", wantErr: "is not host:port"},
		{name: "no port, remote allowed", addr: "redis.example.com", allowRemote: true, wantErr: "is not host:port"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkAddr(tc.addr, tc.allowRemote)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestMajorVersion(t *testing.T) {
	cases := []struct {
		name    string
		info    string
		want    int
		wantErr bool
	}{
		{name: "stable", info: "# Server\r\nredis_version:8.10.2\r\nredis_mode:standalone\r\n", want: 8},
		{name: "missing", info: "# Server\r\nredis_mode:standalone\r\n", wantErr: true},
		{name: "malformed", info: "redis_version:x.y\r\n", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := majorVersion(tc.info)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
