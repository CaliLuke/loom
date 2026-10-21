// Package adapter implements the type-erased adapter boundary for typed activities.
package adapter

import (
	"context"
	"errors"
	"fmt"
	"reflect"
)

// Handler retains the concrete function signature required by backend codecs.
type Handler struct {
	fn reflect.Value
}

// New validates a handler once, before backend registration.
func New(fn any) (Handler, error) {
	v := reflect.ValueOf(fn)
	if !v.IsValid() || v.Kind() != reflect.Func || v.IsNil() {
		return Handler{}, errors.New("activity handler is required")
	}
	t := v.Type()
	if t.IsVariadic() || t.NumIn() != 2 || t.In(0) != reflect.TypeFor[context.Context]() || (t.NumOut() != 1 && t.NumOut() != 2) || t.Out(t.NumOut()-1) != reflect.TypeFor[error]() {
		return Handler{}, errors.New("activity handler must be func(context.Context, I) (O, error) or func(context.Context, I) error")
	}
	return Handler{fn: v}, nil
}

// Wrap preserves the native signature while adapting context and errors.
func (h Handler) Wrap(ctxFn func(context.Context) context.Context, errFn func(error) error) any {
	return reflect.MakeFunc(h.fn.Type(), func(args []reflect.Value) []reflect.Value {
		args[0] = reflect.ValueOf(ctxFn(args[0].Interface().(context.Context)))
		out := h.fn.Call(args)
		last := len(out) - 1
		if !out[last].IsNil() {
			err := errFn(out[last].Interface().(error))
			out[last] = reflect.ValueOf(&err).Elem()
		}
		return out
	}).Interface()
}

// Call invokes a validated handler without a serialization round trip.
func (h Handler) Call(ctx context.Context, input any) (any, error) {
	v := reflect.ValueOf(input)
	if !v.IsValid() || !v.Type().AssignableTo(h.fn.Type().In(1)) {
		return nil, fmt.Errorf("activity input must have type %s", h.fn.Type().In(1))
	}
	out := h.fn.Call([]reflect.Value{reflect.ValueOf(ctx), v})
	var result any = struct{}{}
	if len(out) == 2 {
		result = out[0].Interface()
	}
	if err := out[len(out)-1]; !err.IsNil() {
		return result, err.Interface().(error)
	}
	return result, nil
}

// Assign copies a result into its typed destination without a serialization round trip.
func Assign(dst, src any) error {
	v := reflect.ValueOf(dst)
	if !v.IsValid() || v.Kind() != reflect.Pointer || v.IsNil() {
		return errors.New("activity result must be a non-nil pointer")
	}
	out := reflect.ValueOf(src)
	// A nil interface has no reflect type; typed nil pointers and maps do.
	if !out.IsValid() && v.Elem().Kind() == reflect.Interface {
		v.Elem().SetZero()
		return nil
	}
	if !out.IsValid() || !out.Type().AssignableTo(v.Elem().Type()) {
		return fmt.Errorf("activity result cannot be assigned to %s", v.Elem().Type())
	}
	v.Elem().Set(out)
	return nil
}

// ValidateCall rejects malformed invocations before scheduling commands.
func ValidateCall(name string, input any) error {
	if name == "" {
		return errors.New("activity name is required")
	}
	v := reflect.ValueOf(input)
	if !v.IsValid() || (v.Kind() == reflect.Pointer && v.IsNil()) {
		return errors.New("activity input is required")
	}
	return nil
}
