# Basic credential CLI defaults (#794)

Parent: `2e4c80226ad2f84c27d506558f34e961344bc908`.
Worktree: `/Users/luca/.codex/worktrees/basic-cli-defaults/loom`.

## Sequence

- [x] Inspect shared CLI argument analysis and its HTTP/JSON-RPC consumers.
- [x] Reproduce missing default metadata and uncompilable complete CLI packages.
- [x] Obtain orchestrator readiness approval before production edits.
- [x] Preserve effective defaults in the existing shared Basic argument owner.
- [x] Exercise plain/named, optional/required, inherited/occurrence and independent
  component defaults with direct analysis and generated CLI tests.
- [x] Compare bounded parent/current generated output, repeat generation in fresh
  processes, build and vet the complete generated modules, and check payload/wire
  values for omitted defaults and explicit credential overrides.
- [ ] Independent exact-diff review, root integration and reviewed atomic commit.

## Ownership and acceptance

`buildBasicAuthCLIArgs` constructs physical string arguments for both HTTP and
JSON-RPC. It retains the service scheme's pointer/value representation and the
credential occurrence's type, validation and example, but drops its default.
`makeFlags` passes this incomplete record to `cli.NewFlagData` and
`cli.FieldLoadCode`; the latter consequently treats the optional string as a
pointer while the former has no flag default. The payload field initializer
already owns conversion from physical strings to named service credentials.

Retain each credential's effective default from `selectedTransportConstraints`
in its `AttributeData.DefaultValue`, the same owner used by ordinary transport
arguments. This makes the existing flag declaration, field loader and service
conversion agree without transport-specific assignments or new presence policy.
Use the actual occurrence so a username's default never leaks to the password,
and inherited named defaults and local refinements retain expr's selection.

This repairs admitted designs' CLI compilation and default metadata. It does not
change the CLI's existing empty-flag/absence policy, service scalar presence, or
the pending M7 explicit-zero/empty contract. Basic defaults remain ordinary flag
defaults. Required credentials with defaults receive present payload values under
the existing generic CLI default policy; required credentials without defaults
still demand flags. No public DSL
or runtime API is added. Regeneration repairs already-generated CLI code.

The existing Basic presence model explicitly treats scalar defaults as present
but does not prove their initialization. The CLI identifier model does not own
metadata or pointer semantics. This repair is straight metadata preservation;
direct representation assertions and compiled payload/wire tests establish the
required correspondence without a new concurrency or state model.

## Bounded validation matrix

Reuse `BasicCredentialConstraintsDSL` for required username, required password,
optional named credentials, distinct defaulted named credentials, constraints,
field-name overrides and HTTP multipart controls. Extend coverage separately for
plain strings, inherited defaults, a default on only one component, a local
empty default, and required/default combinations. Compare HTTP and JSON-RPC
outputs: only defaulted CLI flag declarations/help/examples and payload loaders
may change; transport/server/service/OpenAPI artifacts must remain identical.
Both full client CLI trees must compile and vet, and independent generations
must match byte for byte. Generated runtime tests exercise the CLI parser (to
apply default flag values), builder field values and Basic Authorization bytes;
explicit nonempty overrides must survive unchanged. Existing #731 generated
constraint tests remain validation controls.

## Evidence and review handoff

Readiness approved by the orchestrator after the direct and compiled red tests.
The direct regression found `nil` instead of the username default `"alice"`.
Both full transport CLI builds failed because generated defaulted username and
password locals were `string`, but their loaders assigned `*string`. The broader
parent specimen also reproduces plain, one-component, empty and inherited
variants. Source stayed at the parent until those failures were captured.

The production diff is four additions in `service_data_init_args.go`: select the
username and password effective defaults independently and retain them in their
existing argument records. No declaration, conversion, encoder, expression or
runtime policy changed. Direct coverage now checks twelve method configurations,
both components' defaults/requiredness/pointer facts, physical string references
and downstream flag defaults. The runtime harness calls the real generated CLI
parser, observes the typed payload, then captures an outbound client request.
Twenty successful default/override cases per transport check Basic header values
and absence; required-without-default and invalid constrained-value cases still
fail. Empty defaults are covered as authored defaults, not a new explicit-empty
or absence policy.

Validation used Go 1.27.2 on darwin/arm64, explicit `LOOM_DIR` pointing at this
worktree, and the root's pinned dependencies:

- `go test ./http/codegen ./jsonrpc/codegen -run
  'TestBasicCLI|TestBasicCredentialValidationGenerated|TestBasicPresence' -count=1`
  passed (HTTP 8.797s, JSON-RPC 6.948s). This includes the existing server
  constraint and Basic presence wire controls.
- The final CLI harness, including missing-required-flag controls, passed both
  transports (HTTP 6.900s, JSON-RPC 6.797s). Each generated specimen ran full
  `go build ./...`, `go vet ./...` and `go test -race ./...`.
- The bounded broader selection `TestBasic|TestClientCLI|TestFieldLoad|TestPayload`
  passed in HTTP, JSON-RPC and CLI codegen, including existing checked-in CLI
  goldens. The service-codegen package had no matching tests, so that invocation
  is not claimed as service behavioral coverage. The complete `codegen/cli`
  package passed separately (0.157s).
- Pinned golangci-lint v2.14.0 reported zero issues in HTTP codegen, JSON-RPC
  codegen and the shared service testdata package. Repository gofmt, filesize,
  JSON-v2 and NameScope gates passed; filesize reported existing warnings only.

The bounded comparison generated and scaffolded each transport fixture at the
parent, the candidate and a fresh candidate process. Each module required the
verified pushed version
`v1.10.0-alpha.9.0.20261012181439-651372ce6ff0` and explicitly replaced it with
the selected source. The parent source was an immutable `git archive` of the
full SHA above. Both candidate full application modules built and vetted;
both parents failed only their defaulted Basic CLI assignments. No repeat
build/vet was needed for byte-identical source under the same toolchain.

| Specimen | Compared artifacts | Intended changed artifacts | Fresh process differences |
| --- | ---: | ---: | ---: |
| HTTP shared Basic constraints and defaults | 35 | 3 | 0 |
| JSON-RPC shared Basic constraints and defaults | 32 | 3 | 0 |

Every intended difference is confined to CLI output:

1. Each transport's `basic/client/cli.go` assigns defaulted physical strings by
   value and validates those values unconditionally, preserving named conversion.
2. Each transport's `defaults/client/cli.go` does the same for plain, empty,
   one-component, inherited and overridden defaults. Required value assignments
   were already correct and remain unchanged.
3. Each transport's `cli/basic/cli.go` retains defaults in command-field metadata,
   help text and initialization. It uses the existing generic CLI default
   mechanism, including defaults satisfying required payload values when a flag
   is omitted. Without a default, required flags still require explicit input.

All other service, server, client wire, OpenAPI, manifest, path and example files
were byte-identical, and no artifact was added or removed. No checked-in generated
fixture changed. Reproduction inputs are the committed shared fixture and test
harness; temporary comparison modules, logs and manifests are retained only until
independent review and integration complete. The orchestrator owns cleanup and
managed worktree archival after confirming no active process needs them.
