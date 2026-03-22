package main

import (
	"net/http"
	"time"

	cli "example.com/ticktock/gen/jsonrpc/cli/ticktock"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

func doJSONRPC(scheme, host string, timeout int, debug bool) (loom.Endpoint, any, error) {
	var (
		doer loomhttp.Doer
	)
	{
		doer = &http.Client{Timeout: time.Duration(timeout) * time.Second}
		if debug {
			doer = loomhttp.NewDebugDoer(doer)
		}
	}

	return cli.ParseEndpoint(
		scheme,
		host,
		doer,
		loomhttp.RequestEncoder,
		loomhttp.ResponseDecoder,
		debug,
	)
}

func jsonrpcUsageCommands() []string {
	return cli.UsageCommands()
}

func jsonrpcUsageExamples() string {
	return cli.UsageExamples()
}
