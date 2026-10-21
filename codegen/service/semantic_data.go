package service

import (
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// SemanticData owns captured service value occurrences for one generation.
// Construct it after design preparation and before backend analysis. Backends
// share immutable occurrences and resolution caches, but own naming and example
// selection records. It must not be reused for a different or revised design.
type SemanticData struct {
	root        *expr.RootExpr
	context     *expr.ValueContext
	values      map[serviceValueKey]*ValueData
	nextBackend atomic.Uint64
}

// NewSemanticData captures the finalized method values without selecting examples
// or allocating Go names. Backend changes to the design cannot change these values.
func NewSemanticData(root *expr.RootExpr) *SemanticData {
	d := &SemanticData{root: root, context: expr.NewValueContext(), values: make(map[serviceValueKey]*ValueData)}
	for _, svc := range root.Services {
		for _, method := range svc.Methods {
			d.captureMethod(method)
		}
	}
	return d
}

// Services creates a backend analysis with independent Go naming and example
// selection and sampling state. Package aliases within that backend share its
// selected values. Each call creates a fresh synthesis domain; its identity does
// not affect random seeds or generated names.
func (d *SemanticData) Services() *ServicesData {
	promoteSharedTypeLocations(d.root)
	return &ServicesData{Root: d.root, Services: make(map[string]*Data), Ctx: codegen.NewContext(),
		values: &serviceValueStore{semantic: d, scope: strconv.FormatUint(d.nextBackend.Add(1), 10), values: make(map[serviceValueKey]*ValueData)}}
}

// Method returns unselected value records for a method belonging to this owner.
// Each call returns detached records over the same captured occurrences. It
// returns nil for a method from another design. No Go analysis is performed.
func (d *SemanticData) Method(method *expr.MethodExpr) *MethodData {
	if _, ok := d.values[serviceValueKey{method, "payload"}]; !ok {
		return nil
	}
	result := &MethodData{}
	result.PayloadValue = d.value(method, "payload")
	result.ResultValue = d.value(method, "result")
	result.StreamingPayloadValue = d.value(method, "streaming-payload")
	result.StreamingResultValue = d.value(method, "streaming-result")
	result.ErrorValues = make(map[string]*ValueData, len(method.Errors))
	for _, failure := range method.Errors {
		result.ErrorValues[failure.Name] = d.value(method, "error:"+failure.Name)
	}
	return result
}

func (d *SemanticData) captureMethod(method *expr.MethodExpr) {
	defer func() {
		if err := codegen.RecoverPanic(recover()); err != nil {
			context := (&codegen.Context{}).WithService(method.Service).WithMethod(method)
			panic(codegen.NewError(context, method, err))
		}
	}()
	for _, pair := range []struct {
		role      string
		attribute *expr.AttributeExpr
	}{
		{"payload", method.Payload}, {"result", method.Result},
		{"streaming-payload", method.StreamingPayload}, {"streaming-result", method.StreamingResult},
	} {
		d.values[serviceValueKey{method, pair.role}] = captureValueData(d.context, pair.attribute)
	}
	for _, failure := range method.Errors {
		d.values[serviceValueKey{method, "error:" + failure.Name}] = captureValueData(d.context, failure.AttributeExpr)
	}
}

func (d *SemanticData) value(method *expr.MethodExpr, role string) *ValueData {
	value, ok := d.values[serviceValueKey{method, role}]
	if !ok {
		panic(fmt.Errorf("service value %s.%s (%s) does not belong to this generation", method.Service.Name, method.Name, role))
	}
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func captureValueData(context *expr.ValueContext, attribute *expr.AttributeExpr) *ValueData {
	if attribute == nil || attribute.Type == expr.Empty {
		return nil
	}
	occurrence, err := context.NewOccurrence(attribute)
	if err != nil {
		panic(fmt.Errorf("capture service value occurrence: %w", err))
	}
	return &ValueData{Context: context, Occurrence: occurrence}
}
