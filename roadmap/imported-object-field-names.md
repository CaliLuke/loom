# Imported object field identity — #789

Parent: `8375ce604`. JSON property identity and Go field identity are different
namespaces. The importer preserves property keys but currently only allocates
reserved error receiver names, so ordinary normalized-name collisions fail later
in DSL admission. No public contract choice is needed: emit distinct
`struct:field:name` metadata without changing JSON keys or schema constraints.

1. [x] Add failing direct and actual generated tests for the original normalized
   pairs, an occupied numeric suffix, reserved error methods, nested/inline body
   objects, and supported inherited/shared-reference objects.
2. [x] Replace the error-only override pass with one object-local field plan.
   Use the existing NameScope allocator. Reserve all natural field names and
   inherited physical names before allocating replacements. Keep inherited field
   identities unless an effective-object collision or error method requires an override; emit any
   such override on the derived declaration, never mutate a shared base schema.
   Local refinements retain the existing wire-key identity. Keep error-method
   reservations in this same allocation pass.
3. [x] Make schemaBlock consume this plan for named, inline, request, result and
   error objects. Keep Extend relationships, property order, requiredness,
   validation, references and JSON names unchanged; remove the old error-only
   algorithm. Reuse existing namespace allocation invariants, not a new allocator
   or model. Inheritance traversal follows only base references, not recursive
   property graphs.
4. [x] Verify generated DSL admission and Go compilation across the bounded
   family. Check exact JSON keys through mounted conversion/encoding and exported
   contracts, plus protobuf consumer compilation where the design has gRPC.
   Compare parent/candidate output and independent process repeats; ordinary
   output must remain byte-identical. Preserve existing error-field controls.
5. [ ] Document behavior, freeze an exact final packet for independent review,
   then let the orchestrator deliver one atomic commit and clean inactive output.

## Final acceptance

The parent fails the original `@type`/`type` generator witness, the direct
normalization/suffix cases, and the inherited/generated family. The corrected
family covers named, nested and inline request/result properties, shared base
identity, derived error-method reservations with occupied suffixes, preserved
inherited integer type/requiredness/minimum, and actual HTTP error encoding.
JSON keys remain distinct in mounted round-trips. The normal unmodified base
retains `Error`; its derived error uses `ErrorField3` without changing the base.
Rendering twice leaves the normalized source byte-identical.

The gRPC consumer uses `Reference(ImportedFields)` and explicit `Field` tags for
two selected properties. Its service fields keep the imported `Type3`/`Type2`
selectors while protobuf owns `Type`/`Type2`; compiled binary round-trip and
converter assertions check both directions. An initial attempted whole imported
type consumer lacked required protobuf tags and was rejected as expected by
existing admission; that attempt is not acceptance evidence. No protobuf naming
policy changed.

Two existing ordinary fixtures (`supported.yaml` and
`shared_status_different_schemas.yaml`) preserve all 32 generated files and
rendered DSL bytes against archived parent `8375ce604`. All three specimens
produce 62 identical generated files across fresh CLI processes. The only
adversarial DSL changes are collision metadata and derived-only inherited field
metadata; the parent cannot emit its generated module. Every candidate module
compiles and passes vet. The full family passes mounted race tests and libopenapi parsing. All three
exported specs pass the pinned Vacuum contract rules. Existing shared/error aliases and allOf-kind controls pass focused race
checks. Configured importer lint is clean.

This importer-only change does not affect regeneration of checked-in HTTP or
JSON-RPC fixtures, whose authored DSL never invokes the importer. The bounded
import specimens are the affected generation surface. Existing NameScope
reservation/suffix invariants are reused; the package-symbol model does not model
importer inheritance traversal, which the generated base/derived checks cover.
No new graph algorithm, concurrency protocol, allocator or formal model is added.

Evidence: `/tmp/loom-789-final-manifest.json`,
`/tmp/loom-789-output-comparison.json`, `/tmp/loom-789-artifact-ledger.json`, and
`/tmp/loom-789-rendered-family.diff`. Review and delivery remain pending.
