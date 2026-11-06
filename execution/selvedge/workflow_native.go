package selvedge

import (
	"context"

	sv "github.com/CaliLuke/selvedge"
)

type workflowNative struct {
	app      *sv.App
	producer *sv.Producer
	start    func(context.Context) error
	shutdown func(context.Context) error
	submit   func(context.Context, string, string, []byte) error
	describe func(context.Context, string) (*sv.WorkflowDescription, error)
}

func newProducerNative(options []sv.Option) *workflowNative {
	opts := append([]sv.Option(nil), options...)
	n := &workflowNative{}
	n.start = func(ctx context.Context) error {
		p, err := sv.OpenProducer(ctx, opts...)
		if err != nil {
			return err
		}
		n.producer = p
		return nil
	}
	n.shutdown = func(ctx context.Context) error {
		return n.producer.Shutdown(ctx)
	}
	n.submit = func(ctx context.Context, name, id string, input []byte) error {
		_, err := n.producer.StartWorkflowEncoded(ctx, name, input, sv.WithInstanceID(id))
		return err
	}
	n.describe = func(ctx context.Context, id string) (*sv.WorkflowDescription, error) {
		return n.producer.Describe(ctx, id)
	}
	return n
}

func newWorkerNative(options []sv.Option) *workflowNative {
	app := sv.NewApp(options...)
	return &workflowNative{
		app:      app,
		start:    app.Start,
		shutdown: app.Shutdown,
		submit: func(ctx context.Context, name, id string, input []byte) error {
			_, err := sv.StartWorkflowByName(ctx, app, name, nativeWorkflowInput(input), sv.WithInstanceID(id))
			return err
		},
		describe: func(ctx context.Context, id string) (*sv.WorkflowDescription, error) {
			return sv.Describe(ctx, app, id)
		},
	}
}
