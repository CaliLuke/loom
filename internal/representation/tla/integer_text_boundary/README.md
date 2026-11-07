# Integer text boundary model

This is an opt-in TLA+ design aid for a value whose physical Go storage differs from its integer wire type. The storage is set with `Meta("struct:field:type", …)`. It is not part of CI or any default gate.

## Contract

Storage overrides are admitted: narrow, cross-signed and opaque integer storage are all allowed. A value crosses an HTTP text boundary only when both the storage domain and the wire domain can represent it.

- **Encode:** convert the stored value to the wire carrier (`codegen.GoNativeTypeName`) with the round-trip and sign check, then format it. A failure is returned before any header, cookie, query or request is published.
- **Decode:** parse in the wire domain, then convert to storage with the same check before assigning.
- **Ordinary public path builders:** print the stored value as a faithful decimal and do not validate. `BuildRequest` preflight rejects unrepresentable values.

## Files

- `IntegerTextBoundary.tla` is the model. Domains are abstract integer intervals, and Go conversion is modular wrapping. `Mode = "unchecked"` reproduces the parent emitters; `Mode = "checked"` is the candidate.
- `MCIntegerTextBoundary.tla` sets the scaled domains: int8-like signed, uint8-like unsigned, and wider signed storage, each against a signed and an unsigned wire. Values include the domain edges and out-of-range values.
- `checked.cfg` and `unchecked.cfg` select the mode.

## Invariants

- `FaithfulPublication`: published text equals the stored value and lies in the wire domain.
- `ErrorBeforePublication`: a value outside the wire domain is never published.
- `DecodePreservesValue`: an accepted decode equals the published text.
- `RepresentableAccepted`: every representable value round-trips. This guards against over-rejection.

## Run

Use a local `tla2tools.jar` (TLC 2.19 was used); installation is not part of this repository:

```sh
java -cp tla2tools.jar tlc2.TLC -workers 2 -config checked.cfg MCIntegerTextBoundary.tla
java -cp tla2tools.jar tlc2.TLC -workers 2 -config unchecked.cfg MCIntegerTextBoundary.tla
```

Expected results:
- `checked`: no error; 138 distinct states.
- `unchecked`: `FaithfulPublication` is violated. Stored -512 in wider storage is published as wire text `0`. This is the parent's silent wrap.

Do not commit checker state or traces.

## Assumptions and limitations

- Domains are scaled intervals, not the real Go widths.
- `representation.ClassifyIntegerConversion` decides which pairs need the check, using real Go sizes on 386 and amd64. The model does not prove that decision.
- Opaque external storage is modeled as an interval. In code, the emitted `const _ = ^T(0)` proves integer storage at compile time.
- The model does not cover Go formatting interfaces. Ordinary path builders derive decimal text from numeric storage with strconv, using a sign split for opaque storage (`renderIntegerDecimal`), so `fmt.Formatter` and `fmt.Stringer` never run.
- Response publication safety relies on `HandlerLifecycle.EncodeResponse`, which restores headers when the encoder fails before `WriteHeader`. The model abstracts this as "not published".

Generated runtime tests prove that the implementation corresponds to the model. The model does not.

## Implementation seams

- Shared core: `internal/representation/integer.go`. It is consumed by `codegen/init_primitive.go` for constructor conversions.
- Encode formatter: `http/codegen/transport_value_conversion.go` (`renderIntegerText`, `renderIntegerValue`). Callers: `parameter_value.go`, the header and cookie templates, and the ordinary query template.
- Decode check: `integerDecodeCheck` in `transport_value_conversion.go`, used by the shared `scalar_element` partial.
- Faithful path builder: `paths.go` (`renderPathValue`). Its preflight is `parameter_preflight.go` (`integerPathChecked`).

Constructor adapters retain actual source and target attributes and contexts,
including viewed pointer layouts. Fallible field plans participate in the
existing TransformPlan error channel and CLI zero-result/first-error return.
Known noninteger storage is rejected when an integer conversion is required;
opaque replacement capability is established by generated Go compilation, not
by inspecting external type declarations. Safe native widening remains
infallible. This extraction reuses the reviewed arithmetic model unchanged;
the main-baseline and generated constructor tests check its concrete adapters.
