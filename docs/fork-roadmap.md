# Goa Light Fork Roadmap

This fork is intentionally opinionated. The goal is to reduce maintenance cost
by removing legacy compatibility layers and transports we do not plan to
support.

## Completed in this pass

- Removed OpenAPI v2 generation from the HTTP codegen path.
- Standardized generated OpenAPI output on `gen/http/openapi.{json,yaml}`.
- Removed deprecated `swagger:*` metadata support from the active OpenAPI
  generation path.

## Current transport direction

- Keep HTTP and JSON-RPC support.
- Open question: whether gRPC/protobuf support stays in the fork.

## Next high-value cuts

1. Decide whether gRPC stays.
   If HTTP-only is the target, removing gRPC and protobuf support will simplify
   the DSL, metadata model, codegen, release tooling, Makefile, and dependency
   graph the most.
2. Trim release plumbing tied to external repos.
   The current repo still assumes synced example and plugin repositories. That
   is unnecessary for a focused fork.
3. Re-audit metadata keys.
   After transport cuts, remove transport-specific meta keys and docs that no
   longer influence generation.

## Suggested strategy

1. Keep the fork buildable after each transport removal.
2. Delete dead tests and golden files in the same commit as the code change.
3. Prefer hard removals over deprecation shims.
4. Rename outputs and APIs to the new canonical behavior as soon as ambiguity
   disappears.
