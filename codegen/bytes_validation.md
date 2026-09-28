# Named byte validation: issue #576

The shared validation renderer must dereference a physically pointer-valued
named byte slice before converting it to native bytes. Forced-pointer contexts
emit those pointers; ordinary optional byte aliases remain slice values.
Native Bytes and Any, nullable/optional extracted values, and collection
elements retain their existing layouts. The fix changes validation expressions
only, without changing declarations, codecs, schemas, names, or wire behavior.

## Reproduce

Run the focused compiler/runtime checks from the repository root:

    LOOM_DIR="$PWD" go test ./codegen ./http/codegen ./jsonrpc/codegen \
      -run '^(TestBytesValidation|TestNullableAliasValidation|TestCollectionEnumValidation)' -count=1

The generated HTTP and JSON-RPC tests also run build, vet and race checks in
their temporary modules. The direct layout matrix compiles 80 independently
emitted field layouts and exercises absent, empty, short, valid and long values.
Additional controls cover Any and extracted Optional/Nullable values.

The [comparison manifest](bytes_validation_manifest.json) retains all 17 common
designs, exact expected parent failures, and every intended artifact change.
With the prerequisites in the [comparison harness](../internal/valuecontract/README.md):

    LOOM_VALUE_BASE=5c260e95abcc5448623e0d1e0761e6c90b8fe462 \
    LOOM_VALUE_CANDIDATE="$PWD" \
    LOOM_VALUE_MANIFEST="$PWD/codegen/bytes_validation_manifest.json" \
    LOOM_VALUE_RESULTS=/tmp/loom576-new-unique-run \
    go test ./internal/valuecontract -run '^TestCompareRevisions$' -count=1 -timeout=90m

The results directory must not already exist. The manifest describes this
repair against its published parent; later generator changes need their own
reviewed expectations.

## Recorded findings

| Cases | Parent | Candidate | Intended differences |
| --- | --- | --- | --- |
| 13 existing designs and fixtures listed in the manifest | Pass | Pass | None |
| Canonical named Bytes HTTP | Build fails on named byte pointers | Pass | Server validators and service views |
| Unpreserved Bytes HTTP with views | Build fails on named byte pointers in views | Pass | Service views only |
| Canonical named Bytes JSON-RPC | Build fails on named byte pointers | Pass | Server validators and service views |
| Minimal unpreserved HTTP without views | Pass | Pass | None |

There are five changed artifact paths across the three repaired probes. Every
change replaces a conversion of body.Data or result.Data with a conversion of
its dereferenced value, inside the existing nil guard. All other artifact
bytes, including declarations, schemas and codecs, are unchanged. HTTP runtime
checks cover both request validation and client response validation; JSON-RPC
checks cover request validation. Both compile service-view validators.

Acceptance used two immutable source captures:

- Initial capture: 78cdbc9b2912a143abd214467ce9653f92a48d0b75080ec41a9060afaac8c320.
- Final capture: 6c4054515dbb6f84f38271a40731ed61062859ff3a952b3c294d5b67af6b1332.

The initial 16-case discovery command did **not** pass: it rejected the five
then-unlisted intended changes and the incorrect assumption that an unpreserved
body also removed named pointers from service views. Its 13 existing cases
each completed two parent and two candidate runs with matching input and
artifact hashes and passing generation, example generation where applicable,
build and vet.

The final four-case command passed, including the minimal passing control.
Every case again ran twice per revision with byte-identical repetitions. Thus
the accepted 17-case evidence contains 68 independent generation runs. Between
captures, every production generator file and every input in the existing
HTTP/gRPC specimen packages and both ticktock fixtures was unchanged. Changes
were confined to this new fixture/test and concurrent Lean proof sources.
The documentation and manifest were added afterward, outside those input
packages; they do not change generator execution or specimen inputs.

Full repository tests, full lint, generated-code quality, and the final focused
transport tests passed. This evidence establishes the tested Go layouts and
generated modules; it is not a universal theorem about generated Go or byte
wire/schema equivalence.

## Separate obligations

Byte defaults expose the independent transformer comparability bug
[#577](https://github.com/CaliLuke/loom/issues/577): native, named and alias-chain
byte slices are compared with a zero slice using equality in the both-default
context. The transport fixture uses a String default until that repair;
the 80-case validator matrix still includes named byte defaults. Byte schema
lengths and lexical acceptance remain
[#574](https://github.com/CaliLuke/loom/issues/574) obligations.

Keep these sources, the manifest and concise findings. Remove temporary
comparison snapshots, generated modules, binaries and logs after validation
and independent review finish, following AGENTS.md.
