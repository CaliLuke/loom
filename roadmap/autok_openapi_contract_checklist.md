# Auto-K OpenAPI Contract Checklist

## Goal

Use the current Auto-K OpenAPI 3.1 document as the baseline and turn it into a
better machine-consumable contract for SDK generation, validation, and
automation.

This checklist separates:

- framework-wide contract rules that should apply everywhere
- cross-cutting sweeps that affect many endpoints at once
- operation-specific improvements for each published endpoint

All endpoint items below assume the global rules in this document also apply.

The checked items in the baseline section are already satisfied in the
framework. Keep them green during downstream review rather than reopening them
as new framework backlog.

## Baseline Rules For The Auto-K Team

- [x] Keep OpenAPI 3.1.1 plus JSON Schema 2020-12 as the only published HTTP
      contract format.
- [x] Treat the spec as an SDK input artifact, not just human documentation.
- [x] Replace Loom-flavored `operationId` values with SDK-safe stable IDs that do
      not contain `#`.
- [x] Make operation tags match the top-level tag objects exactly.
- [x] Reuse repeated `components.parameters`, `components.requestBodies`,
      `components.headers`, and `components.examples` instead of inlining the
      same shapes everywhere.
- [ ] Standardize errors on `application/problem+json` or a close RFC 9457 style
      profile with stable machine-readable error codes.
- [ ] Keep a single reusable error component per error shape instead of
      duplicating the same `Error` response payload across statuses.
- [x] Mark request-only secrets as `writeOnly` and response-only computed fields
      as `readOnly` where the framework metadata already models them.
- [ ] Add `deprecated` markers at the field, parameter, and operation level when
      behavior is being phased out.
- [x] Add named examples support to the published contract instead of relying
      only on anonymous `example`.
- [ ] Add OpenAPI `links` wherever a successful response naturally points to the
      next operation.
- [ ] Use explicit binary media types and response headers for downloads instead
      of vague JSON or implicit file semantics.
- [ ] Make pagination contracts explicit and uniform: cursor style, limit
      bounds, default sort, and next-page behavior.
- [ ] Make mutation contracts explicit and uniform: idempotency key semantics,
      conflict behavior, and optimistic concurrency where relevant.
- [ ] Use discriminated unions for payloads that already carry a type field.
- [ ] Publish AsyncAPI or a documented OpenAPI extension surface for SSE and
      WebSocket message contracts instead of stopping at handshake-only HTTP
      documentation.
- [ ] Add contract linting for tag mismatches, unsafe `operationId` values,
      inline component duplication, and missing `readOnly` / `writeOnly` on
      sensitive auth fields.

## Cross-Cutting Sweeps

### Sweep A: Naming And Tagging

- [x] Rename every `operationId` to a stable SDK-safe format such as
      `accounts.getProviders`, `auth.login`, or `projectGraph.search`.
- [x] Replace internal tag names such as `rest_auth`, `rest_admin`, and
      `project_graph` with the public tag names that already exist in the top
      level `tags` block.

### Sweep B: Shared Components

- [x] Hoist repeated `projectID`, `accountID`, `userID`, `threadID`, `taskID`,
      `nodeID`, and pagination query parameters into `components.parameters`
      where the framework can safely reuse them today.
- [ ] Hoist repeated request bodies for create, update, patch, and status-change
      operations into `components.requestBodies` whenever the public body
      identity is stable enough to share.
- [ ] Hoist common success and error responses into `components.responses`.
- [x] Hoist shared headers such as `X-Client-Mutation-Id`,
      `Content-Disposition`, cache headers, and rate-limit headers into
      `components.headers` where the framework already recognizes them as
      reusable shapes.

### Sweep C: Auth And Secret Handling

- [ ] Mark `password`, `current_password`, `new_password`, `api_key`,
      `client_secret`, `refresh_token`, `access_token`, `id_token`, `link_token`,
      and similar fields `writeOnly` or `readOnly` as appropriate.
- [ ] Distinguish clearly between browser session auth, Auto-K bearer auth, OIDC
      resource bearer auth, and WebSocket bootstrap auth in descriptions and
      examples.

### Sweep D: Error Semantics

- [ ] Replace generic reused Loom error payloads with typed problem documents that
      keep `type`, `title`, `status`, `detail`, `instance`, and an Auto-K code.
- [ ] Add endpoint-specific documented 409 / 422 / 429 / 503 cases only where
      the server actually guarantees them.
- [ ] Add named examples for auth failures, validation failures, not-found
      failures, and conflict failures.

### Sweep E: Async And Download Semantics

- [ ] Move SSE and WebSocket message-shape documentation into a formal async
      contract layer instead of relying only on the handshake contract.
- [ ] Make every download endpoint advertise exact response content type,
      filename rules, and `Content-Disposition`.

## Endpoint Checklist

## Discovery And Metadata

- [ ] `GET /.well-known/jwks.json`: add cache header documentation, reusable JWKS
      response component, and public `Discovery` tag alignment.
- [ ] `GET /.well-known/oauth-authorization-server`: add reusable metadata
      response component, stable examples for local vs deployed issuer URLs, and
      `Discovery` tag alignment.
- [ ] `GET /.well-known/oauth-protected-resource`: document resource indicator
      semantics and hoist the response into a reusable component.
- [ ] `GET /.well-known/oauth-protected-resource/{mcpPath}`: hoist `mcpPath`
      parameter, document path-specific resource semantics, and reuse the same
      metadata response component.
- [ ] `GET /.well-known/openid-configuration`: add reusable response component,
      stronger examples, and public `Discovery` tag alignment.
- [ ] `GET /docs`: describe whether this is HTML or a redirect contractually and
      use the public `Metadata` tag.
- [ ] `GET /meta/graph/schema`: document schema-versioning guarantees, add links
      to related graph metadata endpoints, and use shared parameter components if
      any query options are added later.
- [ ] `GET /meta/graph/types`: describe sort order and stability guarantees for
      type metadata, and align tags.
- [ ] `GET /openapi.json`: document this endpoint as the canonical contract
      artifact, add content hash / caching guidance, and align tags.

## Health And Debug

- [ ] `GET /health`: model readiness vs liveness explicitly, add a reusable
      health response component, and document whether degraded states are 200 or
      non-200.
- [ ] `GET /debug/lanes`: define this as an internal contract if it is not
      public, add explicit auth / visibility rules, and give the response a
      reusable component.

## Accounts

- [ ] `GET /accounts/{accountID}/providers`: hoist `accountID`, mark any secret
      provider fields `writeOnly` on input-only shapes and absent on read shapes,
      and use a reusable provider response component.
- [ ] `PUT /accounts/{accountID}/providers`: split request and response schemas,
      mark `api_key` `writeOnly`, add conflict / validation problem details, and
      reuse the request body component.
- [ ] `DELETE /accounts/{accountID}/providers`: clarify whether delete is
      idempotent, add a reusable delete response component, and document
      `404` vs `409` semantics.
- [ ] `POST /accounts/{accountID}/providers/refresh-models`: document whether
      this is synchronous or queued, add a response link back to provider status,
      and model provider refresh failures distinctly.
- [ ] `POST /accounts/{accountID}/providers/test`: document what is actually
      tested, add provider-test result codes, and separate connection/auth/model
      test failures in the error contract.

## Admin Accounts, Users, Invites, Projects, Variables

- [ ] `GET /admin/accounts`: define pagination and sort guarantees, hoist shared
      query parameters if present, and link to account detail operations.
- [ ] `POST /admin/accounts`: mark mutable input fields separately from server
      assigned output fields and add a link to the created account.
- [ ] `DELETE /admin/accounts/{accountID}`: clarify hard delete vs soft delete
      semantics and define conflict behavior if users or projects still exist.
- [ ] `PUT /admin/accounts/{accountID}`: distinguish full replace from partial
      update and use reusable request / response components.
- [ ] `GET /admin/accounts/{accountID}/users`: define pagination, sort order, and
      links to user detail or mutation actions.
- [ ] `POST /admin/accounts/{accountID}/users`: mark password fields `writeOnly`,
      add created-user links, and document invite vs immediate-create behavior.
- [ ] `DELETE /admin/accounts/{accountID}/users/{userID}`: clarify idempotency
      and what happens to owned resources or sessions.
- [ ] `PATCH /admin/accounts/{accountID}/users/{userID}`: define patch semantics
      precisely and mark computed fields `readOnly`.
- [ ] `POST /admin/accounts/{accountID}/users/{userID}/reset-password`: mark new
      password fields `writeOnly`, clarify whether the response includes a
      temporary password or only status, and define audit-safe examples.
- [ ] `GET /admin/email/preview/{name}`: document whether response is HTML, text,
      or JSON-wrapped markup, and advertise exact content types.
- [ ] `POST /admin/email/send-test`: make the request / result schema reusable
      and define provider failure semantics separately from template errors.
- [ ] `GET /admin/email/templates`: define template identifier stability and link
      previews to the preview endpoint.
- [ ] `GET /admin/invites`: define pagination, status filtering, and sort order.
- [ ] `POST /admin/invites`: define invite idempotency and duplicate-invite
      conflict semantics, then link to the created invite resource.
- [ ] `POST /admin/invites/{id}/resend`: document resend rate limits and whether
      this is synchronous or queued.
- [ ] `POST /admin/invites/{id}/revoke`: define revoke idempotency and whether
      already-redeemed invites return conflict or success.
- [ ] `GET /admin/projects`: define pagination and sort guarantees, and link
      listed items to project detail operations.
- [ ] `GET /admin/variables`: clarify scope, mutability, and secret redaction
      semantics for returned variables.

## Auth Core

- [ ] `POST /auth/login`: mark password `writeOnly`, define session creation vs
      token issuance explicitly, and add typed auth failure problem responses.
- [ ] `POST /auth/logout`: define whether logout is idempotent, whether it
      invalidates all sessions or only the current one, and use a minimal shared
      success response.
- [ ] `GET /auth/session`: mark server-generated session metadata `readOnly`,
      clarify expiration semantics, and define unauthenticated response shape.
- [ ] `GET /auth/profile`: keep profile output free of write-only auth fields and
      mark computed / immutable fields `readOnly`.
- [ ] `PATCH /auth/profile`: define patch semantics, validation failures, and the
      exact profile fields callers may mutate.
- [ ] `GET /auth/methods`: document method ordering, feature-flag behavior, and
      whether disabled methods are omitted or marked unavailable.
- [ ] `POST /auth/signup`: mark secrets `writeOnly`, define duplicate-account
      conflicts, and link success to profile or session retrieval.

## Auth Password And PATs

- [ ] `PUT /auth/password`: mark both password fields `writeOnly`, distinguish
      invalid-current-password from policy-validation failures, and reuse a
      problem response component.
- [ ] `POST /auth/password/reset/request`: define anti-enumeration behavior,
      whether success is intentionally opaque, and keep the response shape
      stable.
- [ ] `POST /auth/password/reset/confirm`: mark reset token and new password
      `writeOnly`, define expired-token vs invalid-token failures separately, and
      avoid echoing sensitive data in examples.
- [ ] `GET /auth/pats`: define pagination, sort order, and token redaction
      guarantees.
- [ ] `POST /auth/pats`: mark token output `readOnly`, mark any secret creation
      inputs `writeOnly`, and document one-time token visibility clearly.
- [ ] `DELETE /auth/pats/{tokenID}`: define revoke idempotency and whether
      revoking an already-deleted token is `204`, `404`, or `409`.

## Auth Consent, OAuth Identities, And External Login

- [ ] `GET /auth/oauth/identities`: define provider identifier stability and link
      each identity to unlink operations.
- [ ] `DELETE /auth/oauth/identities/{provider}`: define whether unlink is
      blocked when it would remove the last login method, and model that as a
      typed conflict.
- [ ] `POST /auth/oauth/link/confirm`: mark link tokens and passwords `writeOnly`
      and define expired / mismatched link-token failures.
- [ ] `GET /auth/github/login`: document redirect behavior formally and use the
      exact redirect response contract.
- [ ] `GET /auth/github/callback`: define callback success / failure redirect
      contracts instead of leaving them implicit.
- [ ] `POST /auth/github/link`: define whether this returns a URL, starts a
      browser redirect, or creates pending link state, then model that explicitly.
- [ ] `GET /auth/google/login`: document redirect behavior formally and use the
      exact redirect response contract.
- [ ] `GET /auth/google/callback`: define callback success / failure redirect
      contracts explicitly.
- [ ] `POST /auth/google/link`: define whether this returns a URL, starts a
      browser redirect, or creates pending link state, then model that explicitly.
- [ ] `GET /auth/consent/pending`: define the pending-consent resource shape and
      link it to the approval action.
- [ ] `POST /auth/consent/approve`: mark consent token fields `writeOnly`,
      document approval idempotency, and add a link to the resulting redirect or
      session state.

## Auth WebAuthn

- [ ] `GET /auth/webauthn/credentials`: define ordering, pagination if needed,
      and which returned fields are device metadata vs server metadata.
- [ ] `DELETE /auth/webauthn/credentials/{credentialID}`: define idempotency and
      last-credential conflict behavior.
- [ ] `POST /auth/webauthn/login/begin`: define challenge lifetime and whether
      opaque anti-enumeration behavior is enforced.
- [ ] `POST /auth/webauthn/login/finish`: document replay / expired challenge
      failures distinctly and link successful auth to session retrieval.
- [ ] `POST /auth/webauthn/register/begin`: document challenge lifetime and user
      verification requirements explicitly.
- [ ] `POST /auth/webauthn/register/finish`: define duplicate-credential and
      expired-challenge failures distinctly and mark attestation payloads
      `writeOnly` where applicable.

## OIDC

- [ ] `GET /oidc/authorize`: model redirect responses formally, define required
      query parameters precisely, and add named examples per grant flow.
- [ ] `GET /oidc/authorize/{callbackToken}`: clarify whether this is a resumable
      browser action or a backend callback resource and document redirect
      semantics explicitly.
- [ ] `POST /oidc/token`: use a discriminated request-body union on `grant_type`,
      mark `client_secret` and refresh tokens `writeOnly`, and mark issued tokens
      `readOnly`.
- [ ] `GET /oidc/userinfo`: define exact optional vs required claims and keep the
      schema aligned with discovery metadata.
- [ ] `POST /auth/register`: clarify whether this is dynamic client registration,
      mark generated secrets `readOnly`, and define secret rotation semantics if
      supported.

## Invites

- [ ] `GET /invites/validate`: define whether invalid invites are `404`, `410`,
      or `200` with state, and keep the chosen contract stable.
- [ ] `POST /invites/redeem`: define duplicate redemption, expired invite, and
      account-mismatch failures distinctly, and link success to session or
      profile retrieval.

## Chat

- [ ] `GET /chat`: define pagination, ordering, and whether archived / deleted
      conversations are included.
- [ ] `POST /chat`: document request / response streaming vs non-streaming
      semantics explicitly, and add links to the created or updated conversation.
- [ ] `GET /chat/{conversationID}`: define transcript completeness, ordering, and
      pagination if message lists can grow large.
- [ ] `GET /chat/{conversationID}/download`: advertise exact binary content type,
      filename behavior, and `Content-Disposition`.
- [ ] `POST /chat/{conversationID}/summary`: define whether summaries are
      ephemeral, persisted, or versioned, and link to the conversation resource.
- [ ] `DELETE /chat/{conversationID}/undo`: define undo scope precisely and model
      conflicts when no reversible action exists.

## Projects Core

- [ ] `GET /projects`: define pagination, ordering, filtering, and links to
      project detail resources.
- [ ] `POST /projects`: distinguish caller-provided fields from server-assigned
      fields and link to the created project.
- [ ] `GET /projects/reserved-slugs`: define whether this is complete or only
      advisory and how callers should refresh it.
- [ ] `GET /projects/{projectID}`: mark immutable identifiers and timestamps
      `readOnly` and add links to high-value subresources.
- [ ] `PATCH /projects/{projectID}`: define patch semantics and optimistic
      concurrency if the server supports it.
- [ ] `DELETE /projects/{projectID}`: define soft-delete vs hard-delete semantics
      and dependent-resource behavior.

## Project Documents

- [ ] `POST /projects/{projectID}/documents/generate`: define whether generation
      is synchronous, queued, or resumable, and add a link to the generated
      document artifact.
- [ ] `POST /projects/{projectID}/documents/download`: advertise exact binary or
      textual content type, filename behavior, and `Content-Disposition`.

## Project GitHub

- [ ] `GET /projects/{projectID}/github/status`: define connector state machine
      fields clearly and mark derived status flags `readOnly`.
- [ ] `GET /projects/{projectID}/github/mcp-status`: distinguish GitHub app state
      from MCP export state explicitly.
- [ ] `GET /projects/{projectID}/github/repos`: define pagination, filtering,
      install visibility, and sort behavior.
- [ ] `POST /projects/{projectID}/github/link`: define idempotency and conflict
      behavior when a repository is already linked.
- [ ] `DELETE /projects/{projectID}/github/link`: define unlink idempotency and
      what state remains after unlink.
- [ ] `POST /projects/{projectID}/github/export`: define whether export is full,
      incremental, or queued, and link to resulting artifacts or status.
- [ ] `POST /projects/{projectID}/github/mcp-export`: define how this differs
      from normal export and document MCP-specific failure modes.
- [ ] `PATCH /projects/{projectID}/github/tasks/{taskID}/sync`: define whether
      this is partial or full sync and what fields are authoritative.
- [ ] `DELETE /projects/{projectID}/github/tasks/{taskID}/link`: define unlink
      idempotency and resulting task state.

## Project Sources

- [ ] `GET /projects/{projectID}/sources`: define pagination, ordering, and
      filter behavior.
- [ ] `POST /projects/{projectID}/sources`: split input and output schemas,
      advertise supported content types clearly, and add a link to the created
      source resource.
- [ ] `PUT /projects/{projectID}/sources/{sourceID}`: clarify replace semantics
      and mark file content / body fields `writeOnly` where needed.
- [ ] `DELETE /projects/{projectID}/sources/{sourceID}`: define delete
      idempotency and whether downstream graph nodes are removed, detached, or
      preserved.
- [ ] `GET /projects/{projectID}/sources/{sourceID}/download`: advertise exact
      content type, `Content-Disposition`, and any content-encoding guarantees.

## Project Threads

- [ ] `GET /projects/{projectID}/artifacts/{artifactID}/threads`: define
      ordering, pagination, and whether deleted / resolved threads are included.
- [ ] `POST /projects/{projectID}/artifacts/{artifactID}/threads`: split request
      and response schemas, use typed unions if thread bodies can target multiple
      artifact kinds, and link to the created thread resource.
- [ ] `GET /projects/{projectID}/artifacts/{artifactID}/threads/count`: define
      the count scope precisely and whether it honors filters.
- [ ] `GET /projects/{projectID}/threads/{threadID}`: define embedded comment
      ordering and whether related artifact context is included inline.
- [ ] `PATCH /projects/{projectID}/threads/{threadID}`: define patch semantics
      and keep status values as an enum with named examples.
- [ ] `DELETE /projects/{projectID}/threads/{threadID}`: define delete
      idempotency and whether comments are hard-deleted or tombstoned.
- [ ] `POST /projects/{projectID}/threads/{threadID}/bring-forward`: define what
      bring-forward does semantically and how duplicates / conflicts are handled.
- [ ] `POST /projects/{projectID}/threads/{threadID}/comments`: link the result
      to the parent thread and define comment ordering guarantees.
- [ ] `PATCH /projects/{projectID}/threads/{threadID}/comments/{commentID}`:
      define patch semantics and optimistic concurrency if edits can conflict.
- [ ] `DELETE /projects/{projectID}/threads/{threadID}/comments/{commentID}`:
      define delete idempotency and whether deleted comments are removed or
      tombstoned.

## Project Graph Core

- [ ] `GET /projects/{projectID}/graph`: define what root graph payload includes,
      whether it is paginated, and whether it is intended for UI-only or SDK use.
- [ ] `GET /projects/{projectID}/graph/summary`: define summary freshness and
      whether counts are exact or approximate.
- [ ] `GET /projects/{projectID}/graph/search`: define ranking, pagination,
      filter semantics, and sort stability.
- [ ] `GET /projects/{projectID}/graph/search/formatted`: separate presentation
      formatting from canonical machine data, or mark this operation as UI-facing.
- [ ] `POST /projects/{projectID}/graph/similarity/search`: define ranking
      metrics, threshold semantics, and pagination.
- [ ] `GET /projects/{projectID}/graph/subgraph/formatted`: separate formatted
      presentation output from canonical graph resource output.
- [ ] `GET /projects/{projectID}/graph/review-queue`: define ordering and what
      transitions move items in or out of the queue.
- [ ] `GET /projects/{projectID}/graph/check`: define check execution mode,
      severities, and deterministic result ordering.
- [ ] `GET /projects/{projectID}/graph/check/rules`: define rule identifiers as
      stable public IDs.
- [ ] `GET /projects/{projectID}/graph/duplicates`: define ranking, confidence,
      and duplicate grouping semantics.
- [ ] `GET /projects/{projectID}/graph/fragments-by-type`: define type ordering
      and whether empty groups are included.
- [ ] `GET /projects/{projectID}/graph/task-spec/{taskID}`: link this result to
      the backing task node and define version / freshness semantics.
- [ ] `GET /projects/{projectID}/graph/traceability`: define traversal depth and
      ordering guarantees.
- [ ] `GET /projects/{projectID}/graph/visualization`: separate UI-specific
      layout fields from canonical graph structure if both are mixed today.
- [ ] `GET /projects/{projectID}/graph/artifact/{nodeID}`: define whether this is
      a typed union over artifact kinds and model it explicitly with a
      discriminator if so.

## Project Graph Nodes And Edges

- [ ] `GET /projects/{projectID}/graph/nodes`: define pagination, filtering, and
      ordering guarantees.
- [ ] `POST /projects/{projectID}/graph/nodes`: define upsert identity rules,
      partial success behavior, and conflict reporting.
- [ ] `PATCH /projects/{projectID}/graph/nodes`: define patch semantics for bulk
      mutation and partial failure reporting.
- [ ] `DELETE /projects/{projectID}/graph/nodes`: define whether delete is all or
      nothing and how missing IDs are reported.
- [ ] `POST /projects/{projectID}/graph/nodes/create`: clarify how this differs
      from node upsert and whether it guarantees creation-only semantics.
- [ ] `GET /projects/{projectID}/graph/nodes/formatted`: separate UI formatting
      from canonical node payloads.
- [ ] `DELETE /projects/{projectID}/graph/nodes/{nodeID}`: define idempotency and
      dependent-edge behavior.
- [ ] `PATCH /projects/{projectID}/graph/nodes/{nodeID}`: define patch semantics
      and conflict behavior precisely.
- [ ] `PUT /projects/{projectID}/graph/nodes/{nodeID}`: document full replace
      semantics and required field behavior.
- [ ] `POST /projects/{projectID}/graph/nodes/{nodeID}/propose-status`: define
      proposal lifecycle and link to pending status or promotions resources.
- [ ] `POST /projects/{projectID}/graph/edges`: define partial success behavior,
      duplicate-edge behavior, and conflict reporting.
- [ ] `POST /projects/{projectID}/design-fragments`: define whether this is a
      convenience alias over node creation and keep the contract consistent with
      graph node create / upsert behavior.

## Project Promotions And Status

- [ ] `GET /projects/{projectID}/graph/pending-status`: define ordering and
      whether items are deduplicated by node or proposal.
- [ ] `GET /projects/{projectID}/graph/promotions`: define queue ordering and the
      stable identifier for promotion items.
- [ ] `POST /projects/{projectID}/graph/promotions/{nodeID}/approve`: define
      approval idempotency and conflict behavior if state changed concurrently.
- [ ] `DELETE /projects/{projectID}/graph/promotions/{nodeID}`: define whether
      this dismisses one promotion item or all pending promotions for the node.
- [ ] `POST /projects/{projectID}/graph/relations`: define whether this is
      additive, replace-all, or patch behavior and model partial failures
      explicitly.

## Project Task Dependencies

- [ ] `GET /projects/{projectID}/graph/task-deps/get`: define returned graph
      shape, ordering, and whether transitive dependencies are included.
- [ ] `GET /projects/{projectID}/graph/task-deps/next`: define scheduling rules
      and tie-breaking semantics.
- [ ] `GET /projects/{projectID}/graph/task-deps/validate`: define validation
      severity levels and result ordering.
- [ ] `POST /projects/{projectID}/graph/task-deps/add`: define idempotency and
      duplicate-edge behavior.
- [ ] `POST /projects/{projectID}/graph/task-deps/add-bulk`: define partial
      success handling and per-item error reporting.
- [ ] `POST /projects/{projectID}/graph/task-deps/remove`: define idempotency and
      behavior when the dependency is already absent.

## Project Versions

- [ ] `GET /projects/{projectID}/graph/nodes/{displayID}/versions`: define
      ordering, pagination, and whether all version metadata is stable.
- [ ] `GET /projects/{projectID}/graph/nodes/{displayID}/versions/summary`:
      define aggregation semantics and pagination if summaries can grow.
- [ ] `GET /projects/{projectID}/graph/nodes/{displayID}/versions/{version}`:
      define version numbering stability and not-found behavior.
- [ ] `POST /projects/{projectID}/graph/nodes/{displayID}/rollback`: define
      rollback idempotency, conflict behavior, and link to the resulting latest
      version.
- [ ] `GET /projects/{projectID}/graph/nodes/{displayID}/similar`: define
      similarity scoring semantics and ordering.
- [ ] `GET /projects/{projectID}/graph/versions/latest-numbers`: define response
      key stability and whether values are exact snapshots.
- [ ] `GET /projects/{projectID}/graph/versions/needs-review`: define review
      state semantics and ordering.

## Event Stream And Realtime

- [ ] `GET /events`: keep the HTTP handshake in OpenAPI, but publish event
      message contracts in AsyncAPI or a documented extension and add named event
      examples per event type.
- [ ] `GET /ws/projects/{projectID}`: remove the fake JSON `101` response body,
      document the upgrade contract properly, and publish message envelopes in an
      async contract artifact shared with the SSE endpoint.

## Completion Criteria

- [ ] Every published operation has a public tag name that matches a top-level
      tag object.
- [ ] No published `operationId` contains `#`.
- [ ] Repeated path and query parameters are componentized.
- [ ] Repeated request bodies and responses are componentized.
- [ ] Secret request fields are `writeOnly`.
- [ ] Generated secret or token output fields are `readOnly`.
- [ ] Downloads advertise exact content types and `Content-Disposition`.
- [ ] Async endpoints have a truthful async contract story.
- [ ] Error responses are standards-first and typed.
- [ ] The spec passes parser validation, linting, and SDK smoke-generation for at
      least one TypeScript and one Go client target.
