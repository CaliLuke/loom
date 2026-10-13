# Shared interceptor accessor coverage

The model separates generated declaration coverage from invocation applicability.
Three method identities enumerate every nonempty pair of server/client sets,
including single-side, disjoint, overlap and identical sets. Canonical service
order is represented by method numbers. Each method stands for all declared
payload/result/message accessor records; actual type correctness and native
carrier extraction are outside this set model and require generated Go tests.

`current` selects server declarations whenever that side exists, reproducing
#782's missing client coverage. `append` simply concatenates both lists and
violates declaration uniqueness on overlap. `broaden` uses the correct union
for declarations but also widens invocation sets, violating authored side
applicability. `owned` unions/deduplicates declarations in service order and
leaves invocation sets unchanged. The invariants require complete coverage,
unique declarations, stable order and exactly authored side-local calls.

Run each configuration using a separate temporary checker directory:

```sh
java -cp /Users/luca/.local/share/tlaplus/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 1 -metadir /tmp/loom-782-model-owned \
  -config owned.cfg InterceptorAccessorCoverage.tla
```

The implementation seam is service interceptor analysis, consumed by
`interceptors.go` and the public/private accessor renderers. Side-local
`InterceptorData.Methods`, stream wrapper collection and #867's client endpoint
bindings must retain their existing meaning. `TestInterceptorAccessorCoverage`
checks emitted type membership and unchanged method-side applicability;
`TestGeneratedInterceptorAccessorCoverage` exercises actual read/write access,
client/server stream operations and endpoint carrier extraction.

No claim about incompatible field schemas (#767), JSON-RPC carrier shape
(#785), user callback behavior or native transport cancellation is made. The
model does not prove Go compilation, exact-byte determinism or runtime casts.
The current, append and broaden configurations exit 12 with the named
invariants (73, 72 and 64 distinct states respectively); owned exits 0 with
126 distinct states. `interceptor_declarations.go` implements the shared
coverage owner, initialized in service analysis and rebuilt by file alias
analysis. The renderer consumes its two placement lists; original side-local
invocation records remain unchanged. Alias and nonapplication tests extend the
implementation checks without changing the finite set model.


## Signature, storage and server operation contract exploration

`InterceptorContract.tla` extends the investigation to #767 and #785 without
changing the existing declaration-coverage model. Shared signature equality and
method-local physical storage are separate decisions. The finite layouts model
optional versus required primitives and a renamed selector; incompatible logical
signatures are rejected. Server operations and incoming-message boundaries are
independent: JSON-RPC SSE sends contextual events, bidirectional WebSocket methods
send responses/notifications/errors, and request/reply messages enter endpoints.
The native stream case is the generic-operation control.

Run from this directory, substituting a configuration name for `contract-owned`:

```sh
java -cp /Users/luca/.local/share/tlaplus/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 1 -metadir /tmp/loom-interceptor-contract-owned \
  -config contract-owned.cfg InterceptorContract.tla
```

| Configuration | Result | Design consequence |
| --- | --- | --- |
| current-storage | LocalStorage counterexample, 5 distinct states | First-method physical layout cannot serve every method. |
| current-stream | AdapterImplemented counterexample, 4 states | Generic server operations do not implement RPC interfaces. |
| layout-only | LegalAcceptance counterexample, 8 states | Local bindings alone admit incompatible public signatures. |
| signature-only | LocalStorage counterexample, 6 states | Admission alone leaves optional/required and selector failures. |
| methods-only | MessageBoundary counterexample, 5 states | Renaming operations alone invents a missing receive carrier. |
| reject-variants | LegalAcceptance counterexample, 5 states | Rejecting optional/required or supported streams loses valid designs. |
| contract-owned | Pass, 24 generated / 16 distinct states | Shared signature admission, local bindings and recorded operations compose. |

Implementation correspondence is `buildInterceptorData`,
`collectInterceptorMethodBindings` / `collectInterceptorBindings`,
`admitInterceptorContracts`,
`buildInterceptorDeclarations`, `addAccessorMethods`, stream interface analysis
and the server wrapper renderers. `TestInterceptorContractMethodStorage` and
`TestInterceptorContractRejectsIncompatibleSignature` reproduce the binding and
admission defects; `TestInterceptorContractSignatureIdentity` reads actual emitted
Go getter/field types, including nominal, custom and nullable references.
The archived readiness `TestInterceptorContractServerOperations` compares actual
interfaces with wrapper operations and detects the missing request/reply carrier;
its still-failing probes remain outside the atomic #767 diff until #785 resumes.
The actual CLI storage specimens in `testdata/interceptor_contract_family.go`
expose the binding failures to the Go compiler. The #767 implementation separates
shared AttributeData from interceptorMethodBindings, admits signatures across
both sides before publication, and renders each binding's physical selector and
conversion. Existing declaration coverage and native stream tests remain controls.

The model assumes a correct canonical signature oracle; it does not implement
Go nominal identity, imported aliases, nullable carriers or assignability.
Error/context preservation is an explicit adapter obligation, represented as an
unchanged value, not proof of runtime callback behavior. The generic operation
set abstracts different native stream directions. It does not prove native I/O
lifetime, user callbacks, field mutation, projection conversion, actual Go
compilation or emitted declaration completeness. Those remain direct and mounted
Go test obligations. The family plan separates #767 and #785 atomic fixes; this
small joint model does not merge their ownership or claim the pending #785 fix.
