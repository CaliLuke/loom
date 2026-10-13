# Streaming interceptor result identity

Model for #859, initially reproduced at `8f1240831c2df98d80c4b65f760925d805188375`.
The implementation retains the checked shared-owner contract. A stream boundary has one effective attribute
identity. Validation, accessor field analysis and accessor nominal casts must
select that identity; ordinary endpoint result identity must stay unchanged.

The model distinguishes mixed, ordinary result streams, HTTP/gRPC final stream
results, unary methods and JSON-RPC upload request/reply methods. Two field
shapes expose both rejection of distinct authored fields and wrong nominal
casts despite equal field names. Stream eligibility remains the accepted rule;
this does not add unary interception to mixed methods.

| Configuration | Required outcome | Observed distinct states |
| --- | --- | ---: |
| current | AuthoredFieldAccepted violation | 3 |
| current-shared | AttributeIdentity violation | 3 |
| validation-only | AttributeIdentity violation | 4 |
| attributes-only | NominalIdentity violation | 4 |
| explicit-only | AuthoredFieldAccepted violation | 4 |
| owned | All five invariants pass | 21 |

`explicit-only` models ignoring the implicit/final-result fallback: it is an
alternative semantic design, not a claim that prepared MethodExpr currently
leaves StreamingResult nil. Prepare copies Result for ordinary streaming
methods; existing expr tests also validate unprepared final-result methods.
The owned resolver must retain both contracts.

Run each configuration from this directory with Java and TLA+ tools 1.7.4
(TLC 2.19), jar SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`:

```sh
java -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 1 -metadir /tmp/loom-stream-result-owned \
  -config owned.cfg StreamingResultIdentity.tla
```

Use distinct temporary metadirectories. Expected counterexamples exit 12; parse
or evaluation errors are not evidence.

## Implementation correspondence and limits

- `Expected` models the effective stream result attribute, owned
  by MethodExpr.EffectiveStreamingResult and reused by validation and service stream analysis.
- `validated` maps to expr/interceptor.go validateStreamingResultAccess.
- `attribute` maps to populateInterceptorAttributes in service_data_interceptors.go.
- `nominal` maps to MethodInterceptorData.StreamingResultRef and the generated
  accessor structs/casts. The source is canonical ClientStream.RecvTypeRef,
  which records the concrete value for both client access and server access.
  The JSON-RPC server event interface is not that concrete value identity.
- `unary` maps to unchanged Result/ResultRef and existing eligibility validation.
- Direct tests distinguish disjoint fields, string/int field analysis
  and Summary/Chunk nominal identities. The parent mounted HTTP and JSON-RPC SSE harness
  compiled but exposed nominal assertion panics; its unintercepted controls passed.
  The corrected harness checks accessor read/write, EOF and Close. HTTP server
  accessors, optional-field mutation and external-package multi-method dispatch
  add concrete controls. Those Go tests establish correspondence beyond
  this abstract selection model.

The model abstracts views, pointer layouts, shared-interceptor method aggregation,
custom Go codecs, transport framing and lifecycle. Bounded Go tests must cover
optional fields and canonical viewed final results. It does not fix #785's
server wrapper shapes, mixed unary/stream wrapper dispatch (#867), or #726's operation
cancellation. The #720 client-operation model and runtime owners remain valid;
this model adds result-identity agreement and makes no stronger lifetime claim.
