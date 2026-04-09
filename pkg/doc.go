/*
Package loom contains the core runtime types and helpers for the Loom
framework.

Loom is an AI-first service framework focused on stronger machine-consumable
contracts. It generates transport, endpoint, and service scaffolding from a
DSL so that business logic, client code, and published contracts stay aligned.
The loom package provides the runtime pieces shared by generated code,
including service errors, validation helpers, and transport-neutral interfaces
used across HTTP, gRPC, and JSON-RPC.

Loom intentionally puts more weight on OpenAPI 3.1 quality, reusable public
contract components, direct client-generation compatibility, and
framework-owned transport/auth glue that would otherwise be reimplemented in
applications. It was pushed in that direction in part to support the creation
of Auto-K and similar AI-assisted systems.

Visit https://github.com/CaliLuke/loom for the current repository, releases,
and migration notes.
*/
package loom
