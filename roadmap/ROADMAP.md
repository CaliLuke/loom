# Loom Roadmap

This file contains only unresolved, framework-owned work. Completed work belongs
in the implementation, tests, user documentation, release notes, and Git
history—not in the roadmap.

## Priority 1: Correctness and Security

- Remove the legacy X-Ray, request-ID, logging, and tracing middleware surface in
  the next breaking release. Until removal, correct the reversed 4xx/5xx X-Ray
  classification and lock the behavior with focused tests.
- Define and test entropy-failure behavior for generated request IDs and other
  framework-generated identifiers.

## Priority 2: Reproducible Generation and Releases

- Make `loom gen` transactional: generate into a staging tree, validate the
  result, and replace `gen/` only after the complete run succeeds.
- Pin `protoc-gen-go` and `protoc-gen-go-grpc` in framework-owned tooling and
  publish the supported toolchain contract.
- Make the plugin boundary explicit: either document and test it as a public
  extension contract or internalize it.
- Make release preparation transactional and self-verifying, including version
  updates, fixture regeneration, tag publication, and matching GitHub Release
  creation.
- Centralize temp-module local-source selection so integration and release
  harnesses share one untracked, worktree-safe mechanism.

## Priority 3: Documentation and Skill Integrity

- Correct the canonical JSON-RPC documentation for SSE event names, final
  response suppression, eager GET listener behavior, mixed HTTP/SSE routing,
  and session-auth transport behavior.
- Document HTTP negotiation defaults, body-size and multipart limits, trusted
  proxy behavior, remediation metadata, and Pulse lifecycle guarantees.
- Document gRPC prerequisites, initial streaming envelopes, conversion failure
  behavior, and the supported interceptor DSL/runtime boundary.
- Keep observer reason and event lists synchronized with source by generating
  or validating them in CI.
- Add a documentation check that rejects legacy upstream naming, stale commands,
  broken relative links, and duplicated user guides in the Loom skill.

## Priority 4: Verification

- Finish direct seam coverage for HTTP endpoint validation and service-data
  assembly, especially optional-body origins, union collections, remediation
  metadata, raw-object wrapping, viewed-result deduplication, and forced types.
- Prove representative downstream generation in temporary modules pinned to an
  exact pushed Loom commit, then run compile and contract smoke tests without
  modifying the consumer repository.
- Keep the OpenAPI specimen matrix, Redocly validation, downstream client
  generation, generated-fixture regeneration, and full integration suite as
  release gates.

## Decision Rules

Add roadmap work only when it is unresolved, framework-owned, and backed by a
concrete defect, maintenance cost, or downstream consumer need. Remove an item
as soon as the implementation, tests, and durable documentation are complete.

Do not add compatibility work solely to preserve historical upstream behavior,
runtime security policy better owned by applications, or speculative DSL
surface without a current consumer.
