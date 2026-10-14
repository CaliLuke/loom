# Concrete client receiver roles

`ClientMembers.tla` explores #879 receiver ownership. A fixed transport member,
an endpoint factory, a Doer field, a request builder and a mixed HTTP stream
factory can request the same Go receiver member. `current` reproduces unchecked
naming. `fixed_only` reserves fixed members but ignores collisions between derived
roles; this partial correction also fails. `owned` allocates every role against
fixed members and previously allocated roles. `ordinary` includes Close and
IsClosed without WebSocket lifecycle methods, enforcing conditional reservation.

Run using Corretto 25 and TLA+ tools 1.7.4 (TLC 2.19), jar SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`:

```sh
java -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -workers 1 \
  -metadir /tmp/loom-client-members-owned -config owned.cfg ClientMembers.tla
```

Use separate external metadirs per configuration. Readiness results:

| Configuration | Exit | Generated / distinct | Result |
| --- | --- | --- | --- |
| current | 12 | 7 / 7 | Fixed collision reproduced |
| fixed_only | 12 | 13 / 13 | Derived collision remains |
| owned | 0 | 30 / 24 | All six bounded specimens pass |

`http/codegen.finalizeClientMembers` runs in service analysis shared by JSON-RPC, after
all endpoints are analyzed and before constructor/group reservations consume the
plan. Each sequence position is a distinct semantic role; renderers must look up
that role rather than recompute a name. Fixed registrations depend on transport
and WebSocket mode. The finite candidate suffixes abstract NameScope, not its
exact suffix choice. This model does not prove complete registration, request
wiring, import qualification, CLI identity, group delegation or Go syntax.

`TestHTTPClientMemberIdentity` and `TestJSONRPCClientMemberIdentity` inspect actual
emitted fields and receiver methods across generated client files, including
operation groups. Generated module checks pair these with Go compilation and
include the transport CLI. Mounted checks exercise endpoint,
Doer, builder, stream, group and CLI bindings; `TestClientMembersRetainFileImportIdentity` establishes
that names are independent of imports. Server, gRPC and package namespaces are
outside this model.

The `streams` specimen also represents #883's service Client collision:
canonical `Watch` and authored `WatchStream` precede the synthetic stream role.
`service.assignClientMembers` reserves all authored methods, records the mixed
stream method identity and then allocates endpoint fields in that receiver.
The same uniqueness strategy applies; no second allocator or model is needed.
`TestServiceClientMemberIdentity` checks actual emitted roles, order/fallback
competitors and import reanalysis; the original `http-mixed-stream-collision`
module now compiles and checks authored/synthetic calls and canonical interceptor
identity through a mounted peer. The model does not establish completeness of
these emitted roles or their Go method signatures; those remain test obligations.

Keep this correspondence current. Preserve model sources,
configs, pinned toolchain and concise results; remove external checker states
and raw logs after final validation and independent review.
