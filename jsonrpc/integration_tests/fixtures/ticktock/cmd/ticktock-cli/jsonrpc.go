package main

import (
	"net/http"
	"time"

	cli "example.com/ticktock/gen/jsonrpc/cli/ticktock"
	goahttp "github.com/CaliLuke/loom/http"
	goa "github.com/CaliLuke/loom/pkg"
)

func doJSONRPC(scheme, host string, timeout int, debug bool) (goa.Endpoint, any, error) {
	var (
		doer goahttp.Doer
	)
	{
		doer = &http.Client{Timeout: time.Duration(timeout) * time.Second}
		if debug {
			doer = goahttp.NewDebugDoer(doer)
		}
	}

	return cli.ParseEndpoint(
		scheme,
		host,
		doer,
		goahttp.RequestEncoder,
		goahttp.ResponseDecoder,
		debug,
	)
}

func jsonrpcUsageCommands() []string {
	return cli.UsageCommands()
}

func jsonrpcUsageExamples() string {
	return cli.UsageExamples()
}
