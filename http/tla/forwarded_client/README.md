# Forwarded client trust boundary

`ForwardedClient.tla` models the ordered trust walk owned by
`http/request_metadata.go`. The direct network peer must be trusted before any
forwarding entry participates. Walking from right to left, a trusted address
allows traversal to continue, an untrusted address selects the client, and an
invalid entry stops selection with the direct-peer fallback. Invalid entries
must never be removed before walking.

## Design controls and bounds

The finite alphabet has one trusted address, two distinguishable untrusted
addresses and one invalid entry. A chain has zero to four entries. Both trusted
and untrusted direct peers are explored. Each chain is paired with every
same-length chain that has the same suffix beginning at the rightmost
untrusted or invalid entry; entries before that boundary may differ arbitrarily.
This exposes a selected client that depends on an attacker-controlled prefix.

`legacy.cfg` models the previous eager-filter behavior by ignoring invalid
entries while traversing. TLC fails `BoundarySelection` in three states for
`<<clientA, invalid>>`: ignoring the nearest invalid entry allows `clientA` to
be selected. The paired `<<trusted, invalid>>` chain has the same boundary but
has not selected a client. This is the same failure as `6.6.6.6, unknown` in
#708, and informed removing filtering rather than patching individual clients.

`checked.cfg` makes invalid entries terminal. It passes 56,880 distinct states
(85,790 generated), checking:

- `TypeOK`: cursor and results stay within their domains.
- `BoundarySelection`: each completed selection is the rightmost untrusted
  address or the direct peer, according to the nearest boundary and peer trust.
- `PrefixIndependent`: completed selections agree when only text before the
  boundary differs.

The empty/all-trusted chain returns the direct peer. A non-trusted direct peer
never reads a forwarding entry. Terminal states intentionally disable further
reads; deadlock checking is disabled because termination is intended. No
fairness or concurrent behavior is claimed.

## Run

Use TLA+ tools release 1.7.4 (which contains TLC 2.19), with jar SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
The recorded runs used Amazon Corretto Java 25.0.4.1 on Darwin/arm64. Run from
this directory with external checker-state directories:

```sh
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -workers 1 -metadir /tmp/forwarded-client-legacy -config legacy.cfg ForwardedClient.tla
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -workers 1 -metadir /tmp/forwarded-client-checked -config checked.cfg ForwardedClient.tla
```

The legacy command must fail with exit 12 and an invariant violation. The
checked command must exit 0. Keep each run's state directory separate; do not
commit checker traces or state. Remove task-owned outputs after independent
review when no active process needs them.

## Implementation correspondence and limits

| Model | Runtime/test seam |
| --- | --- |
| `trustedPeer` | `snapshotRequestMetadata` checks the parsed direct peer before calling `forwardedClient`. |
| `chain` | Ordered raw `Header.Values("X-Forwarded-For")` and comma entries; the Go implementation walks both levels backward. |
| `ReadHop` | Parse and unmap one raw IP, return fallback on parse error, select an untrusted IP, or continue over a trusted IP. |
| `Fallback` | `snapshotRequestMetadata` keeps its original direct-peer `ClientAddr` when the helper returns an invalid address. |
| `alternate` and `PrefixIndependent` | `TestRequestMetadataForwardedBoundary` changes left prefixes, including their lengths, and covers repeated header lines. |
| Selected result | The same test checks immutable metadata and `EffectiveClientAddress`; `clue/log.TestFrom` checks the logging consumer. |

The model abstracts IP syntax, CIDR membership, whitespace and IPv4 unmapping;
Go tests cover IPv4, IPv6, mapped IPv4, whitespace, malformed/empty/obfuscated
entries and rejected port-bearing values. HTTP parsing preserves repeated
header-value order, which the model treats as one flattened sequence; direct
Go tests cover the two-level traversal. Model prefix substitution preserves
length; Go tests also vary prefix length. Neither the model nor these tests
establish the trustworthiness of application-selected proxy CIDRs or proxy
header construction. `X-Forwarded-Host` and `X-Forwarded-Proto` processing remain
separate and unchanged. No generated program, concurrent mutation, or unbounded
chain theorem is claimed.

Before changing runtime source at parent
`23f2af5d2e4ad9c9b659309839af4f3b25b3f8e3`, the new Go matrix failed twelve
invalid-boundary cases; `TestFrom/invalid_boundary_retains_direct_peer` failed
as well. Valid chains and nearer untrusted boundaries passed. These tests pass
with the corrected shared traversal. Application upgrades require the runtime
fix, not regeneration.
