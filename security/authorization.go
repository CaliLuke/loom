package security

import (
	"context"
	"reflect"

	loom "github.com/CaliLuke/loom/pkg"
)

// EndpointCheck validates and authenticates the typed request and evaluates
// its application authorization requirements. A nil error permits execution
// using the returned context; every error prevents execution.
type (
	EndpointCheck func(context.Context, any) (context.Context, error)

	authorizationOriginKey struct{}

	// A nonzero size gives each middleware boundary a distinct context key.
	authorizationContextKey struct {
		_ byte
	}
)

// Protect runs check before invoking next. Both dependencies are mandatory.
// Middleware factories are constructed once, in order, with the last outermost.
// Checks run before middleware short circuits and repeat before service execution.
// Middleware must preserve the supplied context (or derive a child from it);
// discarding it fails closed. Original request cancellation remains authoritative.
// Checks must be read-only and safe to repeat within a single invocation.
func Protect(check EndpointCheck, next loom.Endpoint, middleware ...func(loom.Endpoint) loom.Endpoint) loom.Endpoint {
	endpoint := protectedEndpoint(check, next)
	for _, wrap := range middleware {
		endpoint = protectedMiddleware(check, endpoint, wrap)
	}
	return endpoint
}

// RequireAuthorizer rejects missing evaluator dependencies at construction,
// including typed nil implementations stored in an interface. It panics on
// invalid configuration, matching generated endpoint constructor semantics.
// Reflection is limited to detecting nil; requests are accessed by typed code.
func RequireAuthorizer(authorizer any) {
	if authorizer == nil {
		panic("authorization requires an evaluator")
	}
	value := reflect.ValueOf(authorizer)
	switch value.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		if value.IsNil() {
			panic("authorization requires a non-nil evaluator")
		}
	}
}

func protectedEndpoint(check EndpointCheck, next loom.Endpoint) loom.Endpoint {
	if check == nil || next == nil {
		panic("authorization requires a check and endpoint")
	}
	return func(ctx context.Context, req any) (any, error) {
		original, ok := ctx.Value(authorizationOriginKey{}).(context.Context)
		if !ok {
			original = ctx
		}
		if err := original.Err(); err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		requestCtx := ctx
		ctx, err := check(ctx, req)
		if err != nil {
			return nil, err
		}
		if err := original.Err(); err != nil {
			return nil, err
		}
		if err := requestCtx.Err(); err != nil {
			return nil, err
		}
		if ctx == nil {
			return nil, loom.Fault("authorization check returned a nil context")
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return next(context.WithValue(ctx, authorizationOriginKey{}, original), req)
	}
}

func protectedMiddleware(check EndpointCheck, next loom.Endpoint, wrap func(loom.Endpoint) loom.Endpoint) loom.Endpoint {
	if wrap == nil {
		panic("authorization requires non-nil middleware")
	}
	key := new(authorizationContextKey)
	wrapped := wrap(func(ctx context.Context, req any) (any, error) {
		original, ok := ctx.Value(key).(context.Context)
		if !ok {
			return nil, loom.Fault("middleware discarded the protected request context")
		}
		if err := original.Err(); err != nil {
			return nil, err
		}
		return next(ctx, req)
	})
	if wrapped == nil {
		panic("authorization middleware returned a nil endpoint")
	}
	return func(ctx context.Context, req any) (any, error) {
		// Authentication may replace its context. Install the marker afterward,
		// retaining the original context without rebuilding middleware state.
		return protectedEndpoint(check, func(authorized context.Context, req any) (any, error) {
			return wrapped(context.WithValue(authorized, key, ctx), req)
		})(ctx, req)
	}
}
