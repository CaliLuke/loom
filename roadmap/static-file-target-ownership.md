# Static file target ownership (#769)

An exact `Files("/index.html", "./dist/index.html")` mapping currently uses
`http.FileServer` directly and redirects to an unmounted `./`. Wildcard targets
already use `http.NewStaticFileServer`, which owns target lookup and distinguishes
files from directories. All generated file handlers must use that existing owner.

## Sequence and acceptance

- [x] Extend the mounted file-serving fixture with exact index paths, aliases and
  independently named targets. Confirm the index redirect fails on the parent.
- [x] Route every generated file target through `NewStaticFileServer`; delete
  the generated prefix filesystem and shared basename remapping table. Let the
  runtime owner handle nil filesystems as it already does for wildcard targets.
- [x] Preserve directory and SPA routes, MIME type, range and conditional
  responses, missing-file errors and request immutability. Keep route admission
  and method mounting unchanged. Regenerate affected goldens.
- [x] Compare bounded representative file-only, mixed, wildcard and ordinary
  designs before/after; compile output and compare independent generation bytes.
  Intended changes are handler construction, removal of duplicated nil defaults
  and deletion of the obsolete generated adapter/imports. Ordinary non-file
  designs must remain identical.
- [ ] Obtain independent exact-diff review, commit one #769 repair, push, close
  the proven report and preserve compact evidence before scratch cleanup.

This is a stateless delegation correction: it introduces no new concurrency or
protocol lifecycle. Existing runtime HTTP file-serving semantics remain the
contract. Application code still chooses the filesystem and file contents.
Regenerate servers to adopt the correction; no new DSL or configuration is needed.

## Evidence before independent review

Parent `c441a62da` fails both exact index routes with HTTP 301 and serves the
second alias target for the first alias. These three failures share the generated
adapter; other mounted SPA and directory cases pass. The candidate removes that
adapter rather than adding request-name exceptions.

- Mounted fixture: 13 route/response cases, including exact index, colliding
  alias basenames, multiple service prefixes, SPA fallback, directory contents,
  range, conditional response and missing directory entry.
- Runtime owner: six existing cases plus five HTTP protocol cases covering GET,
  HEAD, valid/invalid ranges, conditional responses, MIME and request immutability.
- Focused runtime/generator race checks pass; relevant mount/CLI controls and
  scoped lint accompany the packet. Five affected initializer/render goldens are
  regenerated and the obsolete adapter golden is deleted.
- Five representative designs produce 61 artifacts per process. All build/vet;
  four server files differ exactly by construction/default/adapter removal.
  The other 57 artifacts are byte-identical, including every ordinary-design
  artifact. The second candidate process reproduces all 61 bytes exactly.

Compact validation record: `/tmp/loom-769-final-manifest.json`, with source hashes,
parent/candidate output comparison and exact diff. An initial combined command
passed the generator checks but supplied its `-update` flag to the runtime package;
that runtime invocation did not run tests. The subsequent focused race command
ran both packages successfully and supplies the runtime evidence.
