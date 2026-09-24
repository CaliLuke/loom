package redistest

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
		{name: "6.2", info: "# Server\r\nredis_version:6.2.24\r\nredis_mode:standalone\r\n", want: 6},
		{name: "7.4", info: "# Server\r\nredis_version:7.4.11\r\n", want: 7},
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
