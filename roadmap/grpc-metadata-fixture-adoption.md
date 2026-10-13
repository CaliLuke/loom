# gRPC metadata contract adoption (#885)

Delivered as `c01bd5447`, independently reviewed, pushed and confirmed closed.
Exact final/root integration verdict: `/tmp/loom-885-root-final-verdict.json`.

Owner: existing evaluated metadata binding (`expr.ensureMetadataBindings` and `normalizeGRPCMetadata`). Accepted contract: lowercase wire keys, binary `-bin` derivation, and rejection of distinct attributes sharing a complete canonical binding. No production or public policy change is proposed.

## Confirmed parent

Current pushed main 814a3a2e37361b3343dd048052295e122b8edcc3. Targeted current-source run in `/tmp/loom-705-adoption-parent.log` reproduces all metadata failures in `/tmp/loom-ci-155-{ci,grpc,race}.log`. The two YAML race failures remain separate #820. Local source is explicit through LOOM_DIR, Go experiment jsonv2, dedicated correctness GOCACHE.

## Consumer inventory and repair sequence

- [x] Steps 1–4: preserve the parent and repair existing fixtures/peers.
- [x] Step 5: affected packages, bounded generation, direct admission, lint and the CI security specimen pass.
- [x] Step 6: obtain fresh final review and deliver the atomic repair.

1. Preserve parent repro, review original #705 native/collision controls, then safely reuse the delivered #883 checkout.
2. Keep MessageWithMetadataDSL and MessageWithValidateDSL authored uppercase names unchanged. Update eight affected request/response codec goldens only from actual generator output; verify differences are wire-key lowercase normalization. Change service-data and transport-IR assertions to canonical wire keys while asserting original logical fields remain unchanged.
3. Keep the security fixture's JWT, OAuth2, API-key and Basic requirements, payload attributes, and OAuth message placement. Explicitly map API key to `x-api-key`; keep JWT authorization and Basic username/password. Assert all four schemes and message/metadata partition remain covered. Regenerate its affected codec goldens; compile the actual CLI/protobuf module and retain existing proto/CLI-example tests. Keep existing unmapped collision rejection tests.
4. Update only hardcoded bytes wire keys in metadata-defaults direct/native peer harnesses to `data-bin`. Preserve absence/default, explicit empty, supplied value, malformed, requiredness and no-service-call checks. Add explicit expected bytes at mounted native-peer boundary and binary raw value where meaningful: previous mounted assertions only compared limit/lang and echo consistency, so could miss using the fallback for supplied bytes.
5. Rerun the targeted failing tests, bounded actual security design compiler case, and existing direct metadata admission controls. Inspect all regenerated golden differences and require no production edits. Reuse #705 accepted model/native evidence; no new model or exhaustive generation corpus. Compare one unchanged metadata design and changed security specimen through process-isolated output capture when existing capture seams permit, compile affected outputs; no repeated build of identical repeat output.
6. Freeze one atomic CI adoption packet for fresh independent review. Root owns commit and ticket closure.

## Acceptance

All reported metadata CI failures pass with original semantic assertions retained or strengthened. Generated lowercase metadata keys and binary keys agree with native peers. Logical attribute names and service payload fields remain intact. Security specimen is unambiguous without dropping schemes; default colliding bindings still reject. No blanket golden rewrite, no normalization exception, and no runtime changes.

## Evidence

- Parent failures: `/tmp/loom-705-adoption-parent.log` covers all four reported consumer classes on 814a3a2e3; no production failure was established.
- Ten regenerated goldens: eight differ only in wire-key case, verified mechanically by `/tmp/loom-885-golden-classification.json`; two security codec expectations reflect the explicit API-key mapping and consequent metadata iteration/constructor argument order.
- Actual CLI/protoc generation from a physical `git archive` of the parent, current candidate and fresh repeat covers `MessageWithMetadataDSL`, `MessageWithValidateDSL`, and `MessageWithSecurityAttrsDSL`. The first two retain all 30 parent artifacts exactly. The security fixture correctly rejects at the parent and emits 15 compiling candidate artifacts. All 45 candidate artifacts match the independent repeat. Parent accepted modules and all candidate modules pass build/vet. Commands, source provenance and 120 file hashes: `/tmp/loom-885-artifact-receipt.json`.
- Full affected packages pass: `grpc/codegen` 534.840s and `grpc/codegen/internal/transportir` 0.130s. This includes the repaired direct/native peer harness and existing security/proto/example consumers.
- Existing canonical/implicit credential collision and scope controls pass with the race detector (1.566s); scoped gRPC codegen lint reports zero issues.
- The exact previously failing CI design-compilation case (`TestDesigns/grpc/codegen/MessageWithSecurityAttrsDSL`) now passes through gen, example, tidy, build and vet (14.648s), with explicit local source.
- The initial added structural assertion assumed Basic appeared in the bearer-scheme partition. Inspection confirmed Basic has its own credential path; the final assertion checks all four evaluated requirements, both metadata bearer schemes, OAuth message placement, and the four metadata credential bindings. No auth requirement was removed.

No new formal model is needed: this repair adopts the already-reviewed #705 binding contract in existing fixtures and their peers. Production, normalization and collision admission remain unchanged.
