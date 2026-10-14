# Protobuf service declaration namespace — #736

Parent: `4db060dfc46a0addc619ab964260e540dd1b70fa`. The original
BillingClient design still generates a message that conflicts with the service
client interface emitted by protoc-gen-go-grpc. The ordinary-name control builds.
The current reproduction and owner inventory are preserved under
`/tmp/loom-proto-import-namespace-triage`.

The protobuf package scope must contain the service declarations before message
allocation. Existing NameScope allocation owns inferred message identities;
explicit `struct:name:proto` remains authored intent and must produce a contextual
error when its generated Go identity conflicts with a fixed service declaration.
The service name, RPC names and transport import aliases remain canonical.

1. Add failing direct and actual protoc compilation cases for the five reported
   public symbols and the additional emitted Unsafe server, descriptor, method
   constant and conditional stream aliases. Keep normal, explicit noncolliding,
   numbered competitor and repeated analysis controls.
2. Inventory declarations emitted by pinned protoc-gen-go-grpc v1.6.2 in one
   service package owner. Reserve these in the existing protobuf NameScope before
   message analysis; preserve conditional method/stream roles. At message
   collection, reject an explicit collision using that same recorded inventory.
   Delete no-op or duplicated policy rather than adding renderer exceptions.
3. Check inferred message names and every reference through the existing shared
   naming helpers. Preserve service type names, protobuf service/RPC wire names,
   explicit noncolliding names, imports and reanalysis determinism. Reuse the
   existing message-declaration model with fixed occupancy as an input; extend
   only the bounded reserved-owner obligation if needed.
4. Generate a bounded normal/collision/explicit-rejection/stream matrix against
   archived parent and candidate. Compare exact artifacts and repeat generation
   in a fresh process; compile/vet corrected output and run a conversion control.
   Check existing message reservation/declaration tests. HTTP/JSON-RPC fixture
   generators are unaffected; existing gRPC testdata is the relevant surface.
5. Update framework/public guidance for inferred message renaming and explicit
   diagnostics. Freeze exact source, intended output deltas and checks for a
   fresh independent final review; parent orchestrator owns the atomic commit.

This scope does not include importer object-field naming (#789), service
receiver symbols, transport aliases or a general replacement allocator.

## Final candidate evidence

- [x] Shared fixed-declaration inventory installed before message analysis;
  the same record guards explicit names at collection. No new allocator.
- [x] Six inferred collisions and ten authoritative-name collisions reproduce
  on parent; candidate direct checks pass. Public service/RPC names are retained.
- [x] Eleven existing-harness generated modules captured against archived parent,
  candidate and a fresh process repeat. All 157 artifacts per stage are captured;
  repeat matches exactly. Only 24 files in six colliding modules change (message
  declarations/references/descriptors and converter helper names). The 84 files
  in ordinary, streaming, explicit noncolliding and existing message reservation/
  declaration controls are byte-identical to parent.
- [x] Actual protoc AST declaration inventory agrees for unary and streaming
  services. Candidate modules build/vet; existing conversion fixtures and the
  new occupied-suffix roundtrip pass. Located service type imports are covered.
- [x] Focused direct naming/declaration race tests pass (3.312s); expanded generated
  matrix passes under race (11.641s), dedicated roundtrip passes (7.308s), capture
  with compilation/vet/conversion controls passes (17.914s). Scoped lint: 0 issues.
- [x] Existing bounded message-declaration model remains unchanged; its fixed
  occupancy assumption applies, with actual plugin inventory checked in Go.
- [ ] Independent exact-final review, atomic delivery, then inactive output cleanup.

Parent capture changes only test-source selection/capture and skips the new
candidate-only inventory assertion because the parent has no inventory field.
The same DSL/render/protoc/build input is otherwise used. Repeat capture skips
redundant build/vet/test subprocesses and performs fresh actual generation.
No HTTP or JSON-RPC generator/fixture source changed; the bounded existing gRPC
fixture designs are the affected checked-in regression surface.
