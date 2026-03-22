package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"

	mixedtick "example.com/mixedtick"
	clock "example.com/mixedtick/gen/clock"
	"goa.design/clue/log"
)

func main() {
	var (
		hostF     = flag.String("host", "localhost", "Server host (valid values: localhost)")
		domainF   = flag.String("domain", "", "Host domain name (overrides host domain specified in service design)")
		httpPortF = flag.String("http-port", "", "HTTP port (overrides host HTTP port specified in service design)")
		secureF   = flag.Bool("secure", false, "Use secure scheme (https or grpcs)")
		dbgF      = flag.Bool("debug", false, "Log request and response bodies")
	)
	flag.Parse()

	format := log.FormatJSON
	if log.IsTerminal() {
		format = log.FormatTerminal
	}
	ctx := log.Context(context.Background(), log.WithFormat(format))
	if *dbgF {
		ctx = log.Context(ctx, log.WithDebug())
		log.Debugf(ctx, "debug logs enabled")
	}
	log.Print(ctx, log.KV{K: "http-port", V: *httpPortF})

	clockSvc := mixedtick.NewClock()
	clockEndpoints := clock.NewEndpoints(clockSvc)

	errc := make(chan error)
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errc <- fmt.Errorf("%s", <-c)
	}()

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(ctx)

	switch *hostF {
	case "localhost":
		addr := "http://localhost:80"
		u, err := url.Parse(addr)
		if err != nil {
			log.Fatalf(ctx, err, "invalid URL %#v\n", addr)
		}
		if *secureF {
			u.Scheme = "https"
		}
		if *domainF != "" {
			u.Host = *domainF
		}
		if *httpPortF != "" {
			h, _, err := net.SplitHostPort(u.Host)
			if err != nil {
				log.Fatalf(ctx, err, "invalid URL %#v\n", u.Host)
			}
			u.Host = net.JoinHostPort(h, *httpPortF)
		} else if u.Port() == "" {
			u.Host = net.JoinHostPort(u.Host, "80")
		}
		handleHTTPServer(ctx, u, clockEndpoints, &wg, errc, *dbgF)
	default:
		log.Fatal(ctx, fmt.Errorf("invalid host argument: %q (valid hosts: localhost)", *hostF))
	}

	log.Printf(ctx, "exiting (%v)", <-errc)
	cancel()
	wg.Wait()
	log.Printf(ctx, "exited")
}
