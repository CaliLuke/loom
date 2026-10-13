# Public interceptor package symbols (#770)

Status: approved implementation resumed on `2a98e4123`, after the separately reviewed
constructor-scope prerequisite #894 was delivered. The exact prior twelve-file
candidate was preserved, restored without conflicts and hash-verified. Retain the
scalar alias competitor; response-header array alias issue #895 is independent.

1. [x] Reproduce current original method payload/result collision and authored Info collision; retain ordinary compile control using existing generated-module harness.
2. [x] Review this owner/consumer plan independently before production.
3. [x] Allocate public interceptor declaration identities once in service package symbol analysis, record them in interceptor data, and make emitters consume those identities.
4. [x] Verify direct declaration/reference consistency and bounded generated compilation/runtime/output comparison; reuse existing package-symbol allocation invariant.
5. [ ] Freeze final diff for independent review and atomic delivery.

## Cause and correction

`serviceSymbols` reserves fixed declarations and allocates method helpers/private
interceptor adapters before nominal user types. Public Info and four accessor
interfaces bypass this owner: `interceptor_sections` concatenates interceptor
name with role. `buildInterceptorDeclarations` correctly unions coverage across
server/client files but cannot prevent names from colliding with nominal payload,
result or user types. The original Audit/audit design still generates duplicate
AuditPayload and AuditResult declarations; an authored AuditInfo also collides.

Extend the existing package symbol owner with one public declaration record per
applicable interceptor identity. Allocate Info unconditionally for emitted
interceptors and accessor roles only when actually emitted. Use existing NameScope
Unique for each concrete role, in deterministic interceptor order across both
sides, before nominal type allocation. Reuse that record for server/client data;
do not allocate again in declaration union or file-local reanalysis. Preserve
interceptor method names (a distinct interface-method namespace), DSL names,
method identities, private wrapper names, field bindings and operation plans.
Canonical names remain unchanged without collisions. Cache and consume allocated
public names in declarations, method signatures, Info receivers/construction,
accessor returns and helper references, including application example code.

The two public role bases can themselves coincide (AuditStreaming.Payload versus
Audit.StreamingPayload); allocate them in the same scope rather than reserving a
string set and reconstructing names. A nominal suffix competitor such as
AuditPayload2 must also pass through cached type identity. Do not unconditionally
reserve absent accessor roles or import aliases in unrelated package scopes.

## Consumer inventory and boundaries

- `package_symbols.go` / `service_data_analysis.go`: package allocation before
  `collectServiceAnalysisData`, separate method/member namespaces.
- `service_data_interceptors.go` / `InterceptorData`: one reusable public record
  independent of method-specific storage and side invocation membership.
- `interceptor_declarations.go`: existing union/file-placement owner copies names;
  it does not decide namespace policy.
- `interceptor_sections.go`: Info/accessor definitions, interceptor interface
  method signatures, Info accessor receiver and return types.
- Interceptor wrappers and client invocation emitters: every Info construction
  and accessor assertion/reference must use the same record.
- `fileData` reanalysis and located user types: allocated declaration names and
  import aliases must remain consistent when regenerating a file-specific view.
- Example interceptor implementation/CLI consumers must follow recorded Info
  types while preserving canonical interceptor method identity.

#767 typed shared accessor admission, #782 declaration coverage and #785 stream
operation/carrier contracts remain separate and unchanged. #786 private helper
concatenation is already delivered. Protobuf #736 and importer #856 are independent
namespace owners, not part of this correction.

## Evidence and bounded acceptance

Read-only test overlay began on main `3aa0562b2`; the manifest was captured after
HEAD advanced to `43edbf9fe`. Every recorded owner file is byte-identical across
those commits (verified with their scoped Git diff).
- direct service data: authored AuditInfo collides; the method subtest passes
  because its userTypes-only assertion does not inspect method-local payload/result
  declarations. The generated module is the method collision evidence;
- actual generated build: original AuditPayload/AuditResult redeclarations and
  authored AuditInfo redeclaration plus downstream wrong-field errors;
- ordinary service compiles and vets (0.88 seconds); full selected generator
  probe2.086 seconds. Both server/client sides are present, exercising shared
  declaration placement. Inputs/logs retained under /tmp/loom-770-*.

Before final implementation review cover the original pair, explicit Info and
all four accessor-role nominal collisions, helper/helper suffix overlap,
preoccupied numeric suffix, ordinary/disabled-role names, server-only/client-only
and shared/disjoint method coverage. Reuse interceptor harnesses for unary and
streaming runtime behavior plus one located-type/import reanalysis control.
Select only representative collision/unary/streaming modules for parent versus
candidate output/build/vet and independent generation repeat; directly test the
remaining identity permutations. Existing successful accessor-coverage and
server-operation tests cover behavior; do not rebuild the entire design corpus.

The existing `codegen/service/tla/package_symbols/PackageSymbols.tla` models
missing versus reserved declarations, and NamespaceOwnership models independent
method and package scopes. Their documented Missing/Reserved and Split/Owned
counterexamples cover this unchanged allocation policy. Reuse those invariants;
the missing public-role inventory and renderer correspondence are executable
obligations, not claims established by the finite model. No new name allocator
or broad naming model is needed.

## Constructor prerequisite and resumed validation

The streaming/helper-cross specimen includes scalar aliases whose allocated
names differ from authored names. It exposed the independent constructor scope
defect #894; the exact unfinished candidate and witness were archived before
that shared owner was corrected separately. This candidate resumes on delivered
`2a98e4123`, retains the scalar competitor, and now compiles both unary and
streaming all-role modules. The standalone #895 response-header array conversion
witness remains outside this naming correction.

Public identities are stored once in `InterceptorData`; the existing declaration
union copies them without allocation. Direct identity cases and compiled examples
check definitions and references. Existing multiple-method unary/streaming,
located-import reanalysis and RPC bidi runtime tests retain their assertions.


## Final validation and intended differences

The four actual CLI specimens use an archived physical `2a98e4123` parent,
including #894. Original method, Info and helper-cross cases fail compilation on
that parent; the ordinary case builds and vets. All four candidate modules,
including generated examples, build and vet. The 106 generated/example artifacts
match an independent repeat exactly. Ordinary output is unchanged. Intended
changes are the original method's payload/result nominal declarations and their
references (11 files), the authored Info nominal declaration and references
(6 files), and helper/helper plus scalar suffix identities and references
(4 files). Public role allocation uses the existing suffix algorithm, including
preoccupied numeric suffixes; wire names and interceptor method names are unchanged.
Applications regenerate code and update references to previously colliding types.

All 55 generated files across HTTP ticktock, JSON-RPC ticktock and mixedtick match
parent/candidate/repeat exactly; parent and candidate modules build and vet.
Direct role/name and expanded generated compile checks pass under race (service
1.692s, generator 7.563s). Existing accessor, wrapper, multiple-method unary and
streaming, located alias and RPC bidi runtime controls pass (service 1.922s,
generator 11.085s). Scoped configured lint reports zero issues.

The full service-package race run failed only three pre-existing collection/view
snapshots: `service-result-collection-multiple-views`,
`service-result-with-result-collection`, and `service-result-with-dashed-mime-type`.
All three independently fail on the archived parent too (0.553s). Their expected
collection transformation placement is stale; this candidate leaves those
unrelated snapshots unchanged and does not claim a green full-package gate.

The frozen packet is `/tmp/loom-770-final-manifest.json`; the exact CLI output
changes and per-artifact hashes are in `/tmp/loom-770-final-artifact-comparison.json`
and `/tmp/loom-770-final-artifact-differences.patch`. Fixture hashes and commands
are in `/tmp/loom-770-final-fixture-ledger.json`. Model evidence remains the approved
readiness evidence; only the executable correspondence note changed.
