package log

import (
	"context"
	"fmt"

	"github.com/aws/smithy-go/logging"
	"github.com/go-logr/logr"
)

type (
	// StdLogger implements an interface compatible with the stdlib log package.
	StdLogger struct {
		ctx context.Context
	}

	// AWSLogger returns an AWS SDK compatible logger.
	AWSLogger struct {
		context.Context
	}

	// LogrSink returns a logr LogSink compatible logger.
	LogrSink struct {
		name string
		context.Context
	}

	// middlewareLogger is a Loom middleware compatible logger.
	middlewareLogger struct {
		context.Context
	}

	// MiddlewareLogger is the minimal structured logger interface used by Loom
	// middleware adapters.
	MiddlewareLogger interface {
		Log(keyvals ...any) error
	}
)

// NameKey is the key used to log the name of the logger.
const NameKey = "log"

// AsLoomMiddlewareLogger creates a middleware-compatible logger that can be used
// when configuring generated HTTP or gRPC servers.
//
// Usage:
//
//	// HTTP server:
//	import loomhttp "github.com/CaliLuke/loom/http"
//	import httpmdlwr "github.com/CaliLuke/loom/http/middleware"
//	...
//	mux := loomhttp.NewMuxer()
//	handler := httpmdlwr.LogContext(log.AsLoomMiddlewareLogger)(mux)
//
//	// gRPC server:
//	import "google.golang.org/grpc"
//	import grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
//	import grpcmdlwr "github.com/CaliLuke/loom/grpc/middleware"
//	...
//	srv := grpc.NewServer(
//	    grpcmiddleware.WithUnaryServerChain(grpcmdlwr.UnaryServerLogContext(log.AsLoomMiddlewareLogger)),
//	)
func AsLoomMiddlewareLogger(ctx context.Context) MiddlewareLogger {
	return middlewareLogger{ctx}
}

// AsStdLogger adapts a Loom logger to a stdlib compatible logger.
func AsStdLogger(ctx context.Context) *StdLogger {
	return &StdLogger{ctx}
}

// AsAWSLogger returns an AWS SDK compatible logger.
//
// Usage:
//
//	import "github.com/aws/aws-sdk-go-v2/config"
//	import "github.com/CaliLuke/loom/clue/log"
//
//	ctx := log.Context(context.Background())
//	httpc := &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
//	cfg, err := config.LoadDefaultConfig(ctx,
//	    config.WithHTTPClient(httpc),
//	    config.WithLogger(log.AsAWSLogger(ctx)))
func AsAWSLogger(ctx context.Context) *AWSLogger {
	return &AWSLogger{ctx}
}

// ToLogrSink returns a logr.LogSink.
//
// Usage:
//
//	import "github.com/CaliLuke/loom/clue/log"
//
//	ctx := log.Context(context.Background())
//	sink := log.ToLogrSink(ctx)
//	logger := logr.New(sink)
func ToLogrSink(ctx context.Context) *LogrSink {
	return &LogrSink{Context: ctx}
}

// Fatal is equivalent to l.Print() followed by a call to os.Exit(1).
func (l *StdLogger) Fatal(v ...any) {
	l.Print(v...)
	osExit(1)
}

// Fatalf is equivalent to l.Printf() followed by a call to os.Exit(1).
func (l *StdLogger) Fatalf(format string, v ...any) {
	l.Printf(format, v...)
	osExit(1)
}

// Fatalln is equivalent to l.Println() followed by a call to os.Exit(1).
func (l *StdLogger) Fatalln(v ...any) {
	l.Println(v...)
	osExit(1)
}

// Panic is equivalent to l.Print() followed by a call to panic().
func (l *StdLogger) Panic(v ...any) {
	l.Print(v...)
	panic(fmt.Sprint(v...))
}

// Panicf is equivalent to l.Printf() followed by a call to panic().
func (l *StdLogger) Panicf(format string, v ...any) {
	l.Printf(format, v...)
	panic(fmt.Sprintf(format, v...))
}

// Panicln is equivalent to l.Println() followed by a call to panic().
func (l *StdLogger) Panicln(v ...any) {
	l.Println(v...)
	panic(fmt.Sprintln(v...))
}

// Print print to the logger. Arguments are handled in the manner of fmt.Print.
func (l *StdLogger) Print(v ...any) {
	Printf(l.ctx, "%s", fmt.Sprint(v...))
}

// Printf prints to the logger. Arguments are handled in the manner of fmt.Printf.
func (l *StdLogger) Printf(format string, v ...any) {
	Printf(l.ctx, format, v...)
}

// Println prints to the logger. Arguments are handled in the manner of fmt.Println.
func (l *StdLogger) Println(v ...any) {
	Printf(l.ctx, "%s", fmt.Sprintln(v...))
}

func (l *AWSLogger) Logf(classification logging.Classification, format string, v ...any) {
	fn := Infof
	switch classification {
	case logging.Debug:
		fn = Debugf
	case logging.Warn:
		fn = Warnf
	}
	fn(l, format, v...)
}

func (l *AWSLogger) WithContext(ctx context.Context) logging.Logger {
	return &AWSLogger{Context: WithContext(ctx, l)}
}

func (l *LogrSink) Init(info logr.RuntimeInfo) {}

func (l *LogrSink) Enabled(level int) bool {
	return true
}

func (l *LogrSink) Info(level int, msg string, keysAndValues ...any) {
	kvs := make([]KV, len(keysAndValues)/2+1)
	kvs[0] = KV{K: "msg", V: msg}
	for i := 0; i < len(keysAndValues); i += 2 {
		kvs[i/2+1] = KV{K: fmt.Sprint(keysAndValues[i]), V: keysAndValues[i+1]}
	}
	if level == 0 {
		Info(l, kvList(kvs))
	} else {
		Debug(l, kvList(kvs))
	}
}

func (l *LogrSink) Error(err error, msg string, keysAndValues ...any) {
	kvs := make([]KV, len(keysAndValues)/2+1)
	kvs[0] = KV{K: "msg", V: msg}
	for i := 0; i < len(keysAndValues); i += 2 {
		kvs[i/2+1] = KV{K: fmt.Sprint(keysAndValues[i]), V: keysAndValues[i+1]}
	}
	Error(l, err, kvList(kvs))
}

func (l *LogrSink) WithValues(keysAndValues ...any) logr.LogSink {
	kvs := make([]KV, len(keysAndValues)/2)
	for i := 0; i < len(keysAndValues); i += 2 {
		kvs[i/2] = KV{K: fmt.Sprint(keysAndValues[i]), V: keysAndValues[i+1]}
	}
	return &LogrSink{Context: With(l, kvList(kvs)), name: l.name}
}

func (l *LogrSink) WithName(name string) logr.LogSink {
	cur := l.name
	if cur != "" {
		cur += "/"
	}
	cur += name
	return &LogrSink{Context: With(l, KV{NameKey, cur}), name: cur}
}

// Log creates a log entry using a sequence of key/value pairs.
func (l middlewareLogger) Log(keyvals ...any) error {
	n := (len(keyvals) + 1) / 2
	if len(keyvals)%2 != 0 {
		keyvals = append(keyvals, "MISSING")
	}
	kvs := make([]KV, n)
	for i := range n {
		k, v := keyvals[2*i], keyvals[2*i+1]
		kvs[i] = KV{K: fmt.Sprint(k), V: v}
	}
	Print(l, kvList(kvs))
	return nil
}
