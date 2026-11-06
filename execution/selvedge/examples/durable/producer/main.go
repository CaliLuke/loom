// Command producer submits or inspects an object-metadata job without running it.
package main

import (
	"context"
	"encoding/json/v2"
	"errors"
	"flag"
	"fmt"
	"log"
	"time"

	sv "github.com/CaliLuke/selvedge"

	adapter "github.com/CaliLuke/loom/execution/selvedge"
	"github.com/CaliLuke/loom/execution/selvedge/examples/durable/contract"
)

type jobView struct {
	// ID identifies the durable job.
	ID string
	// State reports the native lifecycle status.
	State sv.WorkflowStatus
	// Output contains completed metadata.
	Output contract.Metadata
	// Error contains a recorded job failure, when present.
	Error string
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() (err error) {
	database := flag.String("database", "metadata.db", "SQLite path or PostgreSQL URL")
	action := flag.String("action", "submit", "submit or lookup")
	id := flag.String("id", "", "stable job ID")
	url := flag.String("url", "", "object URL for submit")
	flag.Parse()
	p, err := adapter.NewProducer(sv.WithDatabase(*database))
	if err != nil {
		return err
	}
	if err := adapter.Declare(p, contract.Inspect); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := p.Start(ctx); err != nil {
		return err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err = errors.Join(err, p.Shutdown(cleanup))
	}()
	var value any
	switch *action {
	case "submit":
		value, err = adapter.Submit(ctx, p, contract.Inspect, *id, contract.Input{URL: *url})
	case "lookup":
		var job adapter.Job[contract.Metadata]
		job, err = adapter.Lookup(ctx, p, contract.Inspect, *id)
		view := jobView{ID: job.ID, State: job.State, Output: job.Output}
		if job.Err != nil {
			view.Error = job.Err.Error()
		}
		value = view
	default:
		return fmt.Errorf("unknown action %q", *action)
	}
	if err != nil {
		return err
	}
	data, err := json.Marshal(value, json.Deterministic(true))
	if err != nil {
		return err
	}
	_, err = fmt.Println(string(data))
	return err
}
