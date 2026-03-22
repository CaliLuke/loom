package main

import (
	"context"
	"net/http"
	"net/url"
	"sync"
	"time"

	clock "example.com/mixedtick/gen/clock"
	clockjssvr "example.com/mixedtick/gen/jsonrpc/clock/server"
	loomhttp "github.com/CaliLuke/loom/http"
	"goa.design/clue/debug"
	"goa.design/clue/log"
)

func handleHTTPServer(ctx context.Context, u *url.URL, clockEndpoints *clock.Endpoints, wg *sync.WaitGroup, errc chan error, dbg bool) {
	var (
		dec = loomhttp.RequestDecoder
		enc = loomhttp.ResponseEncoder
	)

	var mux loomhttp.Muxer
	{
		mux = loomhttp.NewMuxer()
		if dbg {
			debug.MountPprofHandlers(debug.Adapt(mux))
			debug.MountDebugLogEnabler(debug.Adapt(mux))
		}
	}

	eh := errorHandler(ctx)
	clockJSONRPCServer := clockjssvr.New(clockEndpoints, mux, dec, enc, eh)
	clockjssvr.Mount(mux, clockJSONRPCServer)

	var handler http.Handler = mux
	if dbg {
		handler = debug.HTTP()(handler)
	}
	handler = log.HTTP(ctx)(handler)

	srv := &http.Server{Addr: u.Host, Handler: handler, ReadHeaderTimeout: 60 * time.Second}
	for _, m := range clockJSONRPCServer.Methods {
		log.Printf(ctx, "JSON-RPC method %q mounted on POST /rpc", m)
	}

	(*wg).Add(1)
	go func() {
		defer (*wg).Done()

		go func() {
			log.Printf(ctx, "HTTP server listening on %q", u.Host)
			errc <- srv.ListenAndServe()
		}()

		<-ctx.Done()
		log.Printf(ctx, "shutting down HTTP server at %q", u.Host)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf(ctx, "failed to shutdown: %v", err)
		}
	}()
}

func errorHandler(logCtx context.Context) func(context.Context, http.ResponseWriter, error) {
	return func(ctx context.Context, w http.ResponseWriter, err error) {
		log.Printf(logCtx, "ERROR: %s", err.Error())
	}
}
