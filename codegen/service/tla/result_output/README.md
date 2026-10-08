# Selected result output contract

This bounded model records the shared boundary fixed for #598: preserve service
errors, select and project a successful result, validate the selected view, then
allow transport emission. Unknown service-selected views and invalid selected
content are server faults. Fields excluded by the selected view do not count
against it. Nil object roots are invalid; nil and empty collection roots are
valid. Non-nullable nil collection elements are invalid.

## Candidate designs and checks

The legacy boundary reports an unknown view as a caller error, emits invalid
content, and panics while projecting nil collection elements. Adding validation
alone still panics before the validator runs. The checked design preserves nil
object values through projection and validates at the shared constructor. It
needs neither transport-specific validation policy nor synthetic replacement
values. Service-returned errors remain unchanged.

The model enumerates unary, raw-request, stream and interceptor paths, seven
source categories, three view selections and service success/error outcomes.
`Prepare` and `Validate` are separate steps. The checked configuration explores
384 distinct states, checks fault classification, no panic, no invalid emission,
exact acceptance and termination under weak fairness.

## Run and expected results

Use TLC 2.19 revision `5a47802` from the official TLA+ tools release, JAR SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
The recorded runs used Corretto Java 25.0.4.1. Set `TLA_TOOLS_JAR` to the installed
JAR. From this directory, run each configuration with a separate output directory:

```sh
for config in legacy-view legacy-content validation-only checked; do
  java -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 \
    -metadir "/tmp/loom-output-$config" -config "$config.cfg" ResultOutput.tla
done
```

- `legacy-view`: exit 12, `NoCallerBlame` fails on an unknown service-selected view.
- `legacy-content`: exit 12, `NoBadEmission` fails on invalid selected content.
- `validation-only`: exit 12, `NoPanic` fails on a nil collection element.
- `checked`: exit 0, all invariants and the termination property pass.

A parsing, evaluation or tool failure is not a successful negative control.
Remove task-owned checker output after validation and review; retain this model,
configurations and findings.

## Production correspondence and limitations

`service_data_views.go` passes the generated validator's allocated name to
`buildViewedResultInit` in `service_data_views_init.go`. Its renderer emits the
checked `NewViewed...` constructor. It wraps selection/validation errors with
`loom.NewServiceError`, preserving the cause and marking the error as a fault.
Object projection helpers preserve nil, allowing the existing collection
validator to report non-nullable nil elements. Reverse client conversion does
not acquire server-fault classification.

`TestViewedResultConstructorValidatesOutput` and
`TestResultProjectionPreservesNilObject` check those generated seams.
`TestResultViewPresenceGeneratedIntegration` compiles and vets the generated
module and runs it under the race detector. Its output harness exercises dynamic,
implicit-default and fixed views, nested constraints, excluded fields, nil roots,
nil elements, service errors, raw-request endpoints, result interception and
WebSocket sends. Unary HTTP checks actual 200/400/500 responses; runtime error
mapping checks gRPC Internal and JSON-RPC internal-error classifications and
preserved validation causes. Existing client presence checks remain in the same
module. Service goldens cover generated collection, nested, union and streaming
shapes. The final-result interception model in the parent directory separately
owns send/close ordering.

The model assumes all four paths enter the shared boundary; tests and code
inspection establish that correspondence. It does not model arbitrary constraints,
recursive or cyclic Go values, codecs, transport status encoders, concurrent
mutation or network delivery. The source categories abstract content validity;
they do not prove the generated validators themselves. A stream fault prevents
that send, but does not roll back earlier sends or an HTTP upgrade. The model
also does not cover client-side null collection elements (#617).
