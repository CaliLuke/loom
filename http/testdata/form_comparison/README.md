# Form codec library investigation (#615)

**Decision: do not adopt go-playground/form v4.5.0 for request decoding.**
Its ownership boundary is promising, but two reproduced parser panics block
adoption of this release. Do not put a recovery wrapper, a second key parser,
or a bracket-translation fallback in front of it. Reconsider a corrected
release or another library. Backward compatibility is not the reason to defer.

This is an isolated, opt-in comparison module, not a new runtime dependency or
CI gate. It pins Loom to pushed commit `bd83a10dfccce82177f47fd7538f6849fcccb88e`
and the candidate to v4.5.0. Loom's `http/form.go` is unchanged from the ticket's
original `88aa08d7` baseline. The evaluated library version was also the latest
version returned by the Go module proxy during this investigation.

## Reproduce

Requires Go 1.27, Python 3, network access on the first run, and a working Go
race toolchain. From this directory:

```sh
go mod download
go test -race -count=1 -v ./...
go vet ./...
python3 run_oauth.py
```

The last command extracts the existing OAuth design and tests from the pinned
Loom module, regenerates client and server code in a temporary module, and adds
the typed adapter experiment. It runs tests with the race detector and `go vet`.
The OAuth client dependency is pinned to `golang.org/x/oauth2 v0.37.0`.
No generated source is edited. Temporary outputs are removed by the script.

Validated with Go 1.27.1 on darwin/arm64. The comparison tests and generated
OAuth tests passed with the race detector; both modules passed `go vet`.
Passing comparison tests mean the documented differences and failures were
reproduced, **not** that the candidate is safe to deploy. `observe` recovers
panics solely to assert the library's unsafe behavior in the experiment.

## Observed contract differences

The library configurations use `ModeExplicit` to honor only tagged fields,
matching generated transport types. Both the default dot namespace and the
library's supported `[` / `]` namespace settings are exercised.

| Input or shape | Loom | Candidate v4.5.0 |
| --- | --- | --- |
| Nested object | `child[name]=a` | Default `child.name=a`; namespace settings support the bracket form directly |
| Map of objects, including empty nested map key | `items[k][name]=b`, round-trips | Default `items[k].name=b`; bracket configuration also round-trips |
| Repeated strings | Repeated key; order preserved | Same |
| Absent optional pointer | Remains nil | Same |
| `number=` into `*int` | Conversion error | Success with nil pointer, as if absent |
| `flag=` into `*bool` | Conversion error | Success with a present false value |
| `text=` into `*string` | Present empty string | Same |
| `number=0`, `flag=false` | Present zero/false | Same |
| Invalid/overflow integer | Error | Error, with field namespace in `form.DecodeErrors` |
| Duplicate scalar key | First value | First value |
| Unknown ordinary key | Ignored | Ignored |
| Byte field `[]byte("hi")` | `data=hi` | `data=104&data=105`; a small registered `[]byte` callback can retain raw bytes |
| Root map `{"k":"v"}` | `k=v` | `[k]=v` in both namespace modes |
| Empty root map key | Encode error | Encodes `[]=empty` and round-trips |
| Array of objects | Encode error | Indexed object fields round-trip |
| `tags[2]=x` | Ignored for a string slice | Allocates three entries, including two empty gaps |

The byte callback probe covers concrete `[]byte`, not arbitrary named byte
aliases. The empty root-map key and object-array results expose limitations of
the current codec; they are not evidence that its behavior should be preserved.
No claim is made about all supported DSL shapes or custom Go types.

## Blocking failures and bounds

A destination with a slice or map invokes the library's shared key parser.
It scans all supplied keys, including unknown ones:

- `unknown[=x` or `unknown]=x` causes `log.Panicf` in `parseMapData`, rather than
  returning a decoding error. `TestMalformedKeySurvivesHTTPParsing` confirms
  that `unknown%5B=x` survives Loom's bounded HTTP form parsing and reaches this
  panic. The input is much smaller than the request-body limit.
- `tags[9223372036854775807]=x` on 64-bit Go overflows `sliceLen + 1` and panics
  with `reflect.MakeSlice: negative len`. The test derives the platform's max
  integer rather than assuming 64 bits.

`tags[10000]=x` correctly returns the default 10,000-element limit error.
However, `SetMaxArraySize` only bounds allocation caused by indexed keys:
setting it to 2 still accepts three repeated `tags` values. Neither this option
nor an HTTP byte limit is a complete aggregate allocation bound. Loom must keep
its request-body byte limit; any adoption must explicitly assess sparse indices,
nesting and total decoded allocation. This experiment does not claim an OOM or
production incident, nor test adversarial throughput.

Sources at the evaluated version:
[decoder parser and allocation](https://github.com/go-playground/form/blob/v4.5.0/decoder.go),
[decoder API and callback scope](https://github.com/go-playground/form/blob/v4.5.0/form_decoder.go),
[encoder API](https://github.com/go-playground/form/blob/v4.5.0/form_encoder.go).
The pinned module source and runnable tests establish these findings. No
upstream issue or pull request was published as part of this investigation.

## Architectural fit and deletion boundary

The library can own reflection-based traversal of ordinary structs, maps,
sequences, pointers, scalar conversion and field tags. These responsibilities
currently occupy the 426-line implementation tail of `http/form.go`, beginning
at `encodeFormValue`. That is a candidate deletion boundary, not a measured net
reduction from a completed replacement. HTTP request construction, byte limits,
error normalization and DSL validation still belong to Loom.

The library's custom-type decoder receives only the strings at one exact key.
It is not called for sibling fields such as `grant.type` and `grant.code`, and
cannot select flattened union branches by itself. The experiment verifies that
limitation. A new generic reflection walker to discover unions would reproduce
the architecture we want to remove.

Instead, generated typed adapters should own the discriminator and branch
selection, and let the library encode/decode the selected branch's ordinary
wire struct. The OAuth experiment needs two small adapter functions and a typed
branch DTO, with no custom traversal. It proves both directions:

- Library plus adapter output is accepted by the unchanged generated server.
- Unchanged generated client output is decoded by the library plus adapter.

Both authorization-code and empty refresh-token branches match exactly. The
existing `x/oauth2` HTTP exchange, generated-client exchange, and discriminator-
only refresh tests also pass against regenerated code.

This does **not** prove a complete replacement: the adapter is a happy-path
root-union prototype. Generated request validation and error normalization stay
in Loom. Scalar union branches, nested unions, unions inside maps/arrays,
recursive union-containing types and custom byte aliases still need a bounded
prototype before claiming that the whole generic engine can be deleted. That
prototype must walk only union-bearing typed paths; if it grows into a second
reflection codec, reject that design.

## Decision required before any later adoption

If the parser defects are corrected, prefer the library's native dot namespace
for ordinary nested objects and retain the standard flat OAuth discriminator
contract through generated adapters. Bracket settings are supported and need
no shim, but compatibility alone does not justify selecting them.

Root-map syntax, empty numeric/boolean semantics, sparse indices, and byte-field
representation are public contract choices, not automatic consequences to hide
inside a dependency upgrade. Native root maps would change `k=v` to `[k]=v`;
empty numbers would become absent and empty booleans false. Raw bytes can remain
a narrow typed codec policy. Record and accept those decisions before adoption;
do not maintain dual grammars or silently weaken authored constraints.

A bounded scan of the sibling `loom-mcp` design sources found no authored
`FormRequest()` use. The existing OAuth fixture is the concrete consumer contract
used here. That scan does not establish the absence of external consumers.
