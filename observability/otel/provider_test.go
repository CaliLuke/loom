package otel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTraceExporterOptions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		cfg  TraceConfig
		want int
	}{
		{
			name: "empty config",
			cfg:  TraceConfig{},
			want: 0,
		},
		{
			name: "all settings set",
			cfg: TraceConfig{
				Endpoint: "localhost:4318",
				Insecure: true,
				Headers:  map[string]string{"x-api-key": "secret"},
			},
			want: 3,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Len(t, traceExporterOptions(tc.cfg), tc.want)
		})
	}
}

func TestMetricExporterOptions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		cfg  MetricConfig
		want int
	}{
		{
			name: "empty config",
			cfg:  MetricConfig{},
			want: 0,
		},
		{
			name: "all settings set",
			cfg: MetricConfig{
				Endpoint: "localhost:4318",
				Insecure: true,
				Headers:  map[string]string{"x-api-key": "secret"},
			},
			want: 3,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Len(t, metricExporterOptions(tc.cfg), tc.want)
		})
	}
}

func TestLogExporterOptions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		cfg  LogConfig
		want int
	}{
		{
			name: "empty config",
			cfg:  LogConfig{},
			want: 0,
		},
		{
			name: "all settings set",
			cfg: LogConfig{
				Endpoint: "localhost:4318",
				Insecure: true,
				Headers:  map[string]string{"x-api-key": "secret"},
			},
			want: 3,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Len(t, logExporterOptions(tc.cfg), tc.want)
		})
	}
}
