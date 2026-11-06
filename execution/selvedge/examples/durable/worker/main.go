// Command worker executes the application's object-metadata jobs.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	sv "github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"

	"github.com/CaliLuke/loom/execution"
	adapter "github.com/CaliLuke/loom/execution/selvedge"
	"github.com/CaliLuke/loom/execution/selvedge/examples/durable/contract"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() (err error) {
	database := flag.String("database", "metadata.db", "SQLite path or PostgreSQL URL")
	lease := flag.Duration("lease", 30*time.Second, "execution lease duration")
	poll := flag.Duration("poll", time.Second, "maximum fallback polling interval")
	flag.Parse()
	w, err := adapter.NewWorker(sv.WithDatabase(*database), sv.WithLease(*lease, *lease/3, *lease/6, *lease/12), sv.WithMaxPollInterval(*poll), sv.WithActivityPollInterval(*poll), sv.WithWorkflowResumptionInterval(*poll))
	if err != nil {
		return err
	}
	if err := adapter.Register(w.Activities(), contract.Read, readMetadata, sv.ActivityOptions{StartToCloseTimeout: 30 * time.Second, RetryPolicy: retry.Fixed(3, 250*time.Millisecond)}); err != nil {
		return err
	}
	if err := adapter.RegisterWorkflow(w, contract.Inspect, func(ctx *sv.WorkflowContext, in contract.Input) (contract.Metadata, error) {
		return contract.Read.Execute(ctx.Context(), w.Activities().Scope(ctx), in, execution.ActivityOptions{})
	}); err != nil {
		return err
	}
	signals, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	init, cancel := context.WithTimeout(signals, 10*time.Second)
	err = w.Start(init)
	cancel()
	if err != nil {
		return err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err = errors.Join(err, w.Shutdown(cleanup))
	}()
	if _, err := fmt.Println("ready"); err != nil {
		return err
	}
	<-signals.Done()
	return nil
}

func readMetadata(ctx context.Context, in contract.Input) (contract.Metadata, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodHead, in.URL, nil)
	if err != nil {
		return contract.Metadata{}, err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return contract.Metadata{}, err
	}
	if err := response.Body.Close(); err != nil {
		return contract.Metadata{}, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return contract.Metadata{}, fmt.Errorf("object returned HTTP %d", response.StatusCode)
	}
	return contract.Metadata{Length: response.ContentLength, ContentType: response.Header.Get("Content-Type")}, nil
}
