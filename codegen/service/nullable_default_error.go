package service

import (
	"fmt"
	"strings"
)

// renderNullableDefaultError stores one native occurrence. Public Nullable values
// are snapshots; Unwrap exposes the same pointer used by later projection.
func renderNullableDefaultError(data *Data, owner *nullableErrorData) string {
	failure := owner.error
	name := data.Scope.GoTypeName(owner.owner)
	ref := data.Scope.GoTypeRef(owner.owner)
	constructor := nullableErrorConstructor(data.Scope, owner.owner)
	description := failure.Description
	if description == "" {
		description = failure.Name
	}
	var b strings.Builder
	fmt.Fprintf(&b, `// %s owns the %q error independently of its nullable native occurrence.
type %s struct {
 payload loom.Nullable[*loom.ServiceError]
}

// %s constructs a nullable error and checks its concrete declared identity.
func %s(payload loom.Nullable[loom.ServiceError]) (%s, error) {
 e := &%s{}
 if payload.IsNull() {
  e.payload.SetNull()
 } else if value, ok := payload.Value(); ok {
  e.payload.SetValue(&value)
 }
 if _, err := e.Payload(); err != nil {
  return nil, err
 }
 return e, nil
}

// Payload snapshots the native occurrence after checking its declared identity.
// Absence and null remain distinct. A nil owner or invalid occurrence is rejected.
func (e %s) Payload() (loom.Nullable[loom.ServiceError], error) {
 var payload loom.Nullable[loom.ServiceError]
 if e == nil {
  return payload, fmt.Errorf(%q)
 }
 if e.payload.IsNull() {
  payload.SetNull()
 } else if value, ok := e.payload.Value(); ok {
  if value == nil || value.Name != %q {
   return payload, fmt.Errorf(%q)
  }
  payload.SetValue(*value)
 }
 return payload, nil
}

// LoomErrorName returns the declared identity, including for null payloads.
func (e %s) LoomErrorName() string {
 return %q
}

// LoomServiceError returns only an agreeing concrete occurrence for response resolution.
func (e %s) LoomServiceError() *loom.ServiceError {
 if e != nil {
  if value, ok := e.payload.Value(); ok && value != nil && value.Name == %q {
   return value
  }
 }
 return nil
}

// Unwrap exposes the authoritative concrete occurrence for diagnostic errors.As.
// Mutation through that pointer is checked again by Payload and response resolution.
func (e %s) Unwrap() error {
 if e != nil {
  if value, ok := e.payload.Value(); ok && value != nil {
   return value
  }
 }
 return nil
}

// Error returns the agreeing native message or the declared description.
func (e %s) Error() string {
 if value := e.LoomServiceError(); value != nil {
  return value.Error()
 }
 return %q
}

// LoomErrorRemedy preserves an agreeing occurrence's remedy, including nil.
// Without a concrete occurrence it supplies only the authored declaration remedy.
func (e %s) LoomErrorRemedy() *loom.ErrorRemedy {
 if value := e.LoomServiceError(); value != nil {
  return value.Remedy
 }
`, name, failure.Name, name, constructor, constructor, ref, name,
		ref, "nil "+name+" error owner", failure.Name, "concrete ErrorName does not match declared error "+failure.Name,
		ref, failure.Name, ref, failure.Name, ref, ref, description, ref)
	if failure.Remedy == nil {
		b.WriteString("return nil\n}\n")
	} else {
		fmt.Fprintf(&b, "return &loom.ErrorRemedy{Code: %q, SafeMessage: %q, RetryHint: %q}\n}\n", errorRemedyCode(failure), errorSafeMessage(failure), errorRetryHint(failure))
	}
	return b.String()
}
