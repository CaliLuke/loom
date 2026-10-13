# HTTP wire-name admission — #710

Readiness independently approved on `34d04a0e1`; implementation starts on
`9a4f1c2f3` after shared request-cookie publication #715. Completion was
transferred to the managed implementation worktree on `227ea92be`; the accepted
shared owner and first-colon contract remain unchanged.

All 60 HTTP/JSON-RPC request, success and error header/cookie name cases fail
solely because admission is missing (0.503s; independent 0.174s). The initial
JSON-RPC fixture placed its route at method scope; its unrelated rejection was
corrected before these results. Readiness approval is
`/tmp/loom-710-independent-readiness-verdict.json`.

1. Reproduce malformed effective header/cookie names in HTTP and JSON-RPC request,
   success and error bindings; cover inherited and session-derived bindings.
2. Use the existing shared expression admission owners. The request wire-binding
   plan already includes explicit and inferred credential names; response bindings
   and response-cookie validation already cover both successes and errors.
3. Preserve the entire suffix after the first colon in `ElementName`. The DSL
   defines attribute:element mapping and no third component; the old second cut
   silently hid invalid header/cookie names and lost valid query/body names.
   Verify full-suffix mappings, distinct query/body names, and gRPC
   metadata preservation (gRPC has no key grammar admission today). Add one private
   name-admission function using the existing pinned
   golang.org/x/net/http/httpguts.ValidHeaderFieldName token grammar for header and
   cookie names. Invoke it for final physical bindings and response cookie names.
   Do not validate logical attribute names, trim authored wire names, or apply
   token grammar to query/form/path/body names. No new dependency or transport patch.
4. Keep response-cookie duplicate-source/name diagnostics and binding occupancy
   unchanged. Expanding that plan solely to rewrite unrelated duplicate diagnostics
   is unnecessary: both existing admission paths call the same grammar owner.
5. Prove invalid names fail during evaluation across consumers, while all token
   punctuation and ordinary/inherited/security mappings remain valid. Compare a
   bounded representative valid generated output set byte for byte across parent,
   candidate and independent-process repeat, compile it, and verify an invalid CLI
   design leaves prior generated output intact. No exhaustive corpus.
6. Independently review the exact final change, deliver one atomic commit and
   preserve concise evidence before cleaning inactive scratch/build files.

This corrects a protocol admission defect; it does not change the supported HTTP
name grammar. Request cookie *values* remain the distinct runtime publication
owner in #715. Query names may contain characters that need URL escaping and must
not be rejected by the header/cookie rule. Existing case-insensitive header slot
occupancy and case-sensitive cookie slot occupancy remain unchanged.

No concurrency, ordering protocol or ownership transition changes. Existing
binding-plan model covers occupancy; direct admission and generated rejection/
parity tests cover this grammar predicate without a new model.

## Completion evidence

- [x] Shared effective HTTP header/cookie admission and full-suffix mapping.
- [x] Parent failures retained: 60 invalid-name admission cases, plus colon query
  and explicit-body cases that previously collapsed distinct names and failed
  evaluation. The latter use the actual parent versions of all three changed
  production files through a Go overlay; their renderer is in-process, with no
  external CLI source-mode ambiguity.
- [x] Full expression and DSL race suites pass (7.338s and 1.348s), including
  inherited/session names, primitive response cookies, complete token punctuation,
  empty-name admission, unchanged occupancy checks and gRPC suffix preservation.
- [x] Existing mapped query/body fixtures now accept configurable element names.
  Their ordinary configurations remain unchanged. Their existing mounted
  harnesses check complete colon query and body names, required fields, validation
  errors, invocation, decoded payloads and exact response JSON (7.601s).
- [x] Ordinary mapped and cookie modules compile/vet and pass mounted controls
  (HTTP 12.914s, JSON-RPC 5.479s). All 45 cookie artifacts match the independently
  reviewed #715 candidate. No checked-in transport fixture has a changed mapping.
- [x] Thirty ordinary parent/candidate artifacts are byte-identical; all 60
  candidate artifacts repeat exactly in an independent process, including both
  OpenAPI formats. The only intended behavior change in these specimens is that
  previously colliding full-suffix query/body designs now evaluate and preserve
  their distinct wire names. Four colon OpenAPI documents pass pinned Vacuum
  with zero errors; JSON documents also pass the existing libopenapi checks.
- [x] Actual CLI rejection reports the complete invalid `X:Colon` header name
  and leaves the prior generated tree byte-identical.
- [x] Scoped production/testdata lint, formatting and JSON-v2 checks.
- [ ] Independent exact final review, atomic delivery and inactive-artifact cleanup.

The initial new response-schema assertion dereferenced a reference proxy rather
than looking up the component as the existing test does. That test-only mistake
was corrected before the passing wire/spec result; no production compensation was
made. gRPC key grammar remains a separate existing admission gap: this change
preserves its complete metadata key and does not introduce a new grammar policy.
Compact comparison and CLI receipts are `/tmp/loom-710-full-comparison.json`,
`/tmp/loom-710-cookie-parity.json` and `/tmp/loom-710-invalid-cli-receipt.json`.
