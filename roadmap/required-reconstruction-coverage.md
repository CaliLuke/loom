# Required value reconstruction coverage (#780)

Status: implementation and bounded validation complete against parent `92e232503`; exact final review pending.

- [x] Four current-behavior admission failures and sixteen controls retained.
- [x] Root approved shared expression owner and bounded validation scope.
- [x] Complete shared admission implementation and phase/source controls.
- [x] Compare bounded generated outputs and actual CLI rejection; run affected gates.
- [ ] Fresh independent exact-final review, delivery and cleanup.

1. Reproduce the three original admission failures on clean delivered source and retain mapped/optional controls.
2. Map each reconstruction source to its existing expression owner and actual constructor consumer. Add bounded authentication, JSON-RPC ID, Tag, view, raw-stream and decoder-default controls. Preserve required-but-unmapped defaults as a negative.
3. Propose one private expression coverage query: logical required keys of the actual reconstructed value minus effective body fields and explicit reconstructed bindings. Feed request, success and error admission from it, without renderer-specific validation or invented defaults.
4. Review phase correspondence, inherited mappings, view projection and error-name evidence; retain any accepted-expression/emission defect separately.
5. After independent readiness approval, implement and run direct admission plus bounded generated HTTP/JSON-RPC request/error round trips. No corpus or new concurrency model is justified by this static coverage work.

Coverage is a necessary admission condition, not proof that every renderer emits the admitted mapping. The independently retained #774 JSON-RPC mapped-error-header omission is a separate emission defect; #780 cannot close it.

A default is not automatically a reconstruction source. `buildTransformObjectInit` and `buildTransformObjectFieldCode` visit matching fields only; `DefaultForField` then governs absent source storage. A wholly omitted service field never enters that conversion. Existing mapped/default decoder paths must stay accepted and retain absence/zero semantics.

A custom ErrorName field is not reconstructed from Loom-Error today: the header chooses the error case; `errorBuilder.buildResultInit` receives body/header/cookie arguments only. Reject its uncovered required field rather than silently adding a constant assignment. Default ErrorResult has a distinct existing problem projection and inferred headers owner that must remain valid.

## Shared owner and phase

`expr/http_reconstruction.go` owns one private reconstruction-coverage helper, next to HTTP body/wire binding admission. Its input is a semantic object occurrence, the effective selected/listed/inferred body and the explicit reconstructed logical keys; output is the ordered uncovered required keys. `AttributeName`/`FindAttribute` resolve declared identities, never `Goify`, JSON wire names or generated fields. A selected body covers its `origin:attribute`; a listed body covers its actual declared members, not its Reference/Bases graph. Inferred bodies derive from the existing request/response body queries. Scalar/collection roots keep their current transport admission; this repair does not invent an object-key obligation for them.

Request validation obtains payload sources from `ensureWireBindings`: ordinary params/header/cookie entries and Basic's two source keys; only `sourceKind=payload` counts. Add the existing JSONRPCID binding separately (it is a body exclusion, not an ordinary wire record), and named MapParams. Transport-owned Session cookies do not satisfy an unrelated service field. Inherited mappings are already merged in endpoint Prepare. There must be no downstream reimplementation of this set subtraction.

Success response validation uses the same query with response headers/cookies and the existing typed Tag binding, only where it actually reconstructs the client value. Each response must cover its own representation; one response's header cannot satisfy another. For result types, use the existing `Project`/view requiredness owner to determine the actual selected view's obligations (each allowed view for dynamic results), not the original result's union of required fields. This preserves intentionally hidden required fields. A Tag is an existing HTTP status-selected reconstruction source; do not invent JSON-RPC status selection to exempt a field that its consumer never reconstructs.

Error admission resolves `designError()` before finalization, then uses the same query after `prepareErrorWireBindings` adds the existing default-problem field headers. For custom errors, explicit body/header/cookie bindings count; the static Loom-Error dispatch header does not. JSONRPCErrorSelector similarly selects candidates from present data and is not a missing-field assignment.

Run coverage after each existing wire/body validity check, with nil/invalid references guarded so existing useful diagnostics survive. The query should not mutate or finalize a body just to discover its fields. A bounded phase check must confirm identical key coverage before and after normal Finalize; before-finalization bodies already retain selected origin/member names. Requiredness must come from the service occurrence or projected view, not an uninitialized body clone.

## Limits and acceptance

This is structural coverage, not a new generic field-presence validator. It does not change optional omission, credential alternatives, defaults, custom codecs/full tags, raw stream access, or view semantics. An authored opaque Go codec remains responsible for its admitted representation. OpenAPIRequestBody/OpenAPIBody are documentation only and cannot satisfy a runtime field obligation. Skip-body paths keep their existing all-typed-fields-mapped checks; raw readers are separate service arguments. Nested codec transformations and collection roots keep their existing owners; no recursive graph walk is needed to detect the reported top-level omissions.

Current bounded evidence on clean `19cf194ad4bf0d56e2e2fe9d85e845c702b8a215`:
- Original selected request/listed request/listed custom error admission expectations fail; omitted required field with a default also fails the proposed admission expectation.
- Sixteen acceptance controls pass: four original mapped/optional controls, eight Basic/JWT/ID/Tag/fixed+dynamic-view/raw-request/mapped-default controls, four full/mapped-error/default-problem controls.
- Existing default-application source-carrier tests pass under race (1.881s). `walkMatches` iterates the source members; there is no conversion visit or default application for a wholly absent field.
- The first added problem control incorrectly used the obsolete name `message`; it was corrected to the existing problem field `detail`. Its failure is test setup, not evidence for #780.

Final assertions name the missing logical field and its diagnostic context.
Inherited/suffixed/MapParams and pre/post-Finalize controls are implemented.
Existing typed Tag, viewed-result, selected-problem and optional-body/default
owners retain their contracts; bounded mounted checks and actual CLI rejection
are recorded below.

The #774 JSON-RPC mapped-error-header witness is retained at `/tmp/loom-774-r1-mapped-header-server-residual_test.go`. That field has an admitted explicit mapping but its server omits emission; coverage admission alone cannot repair it. Reconcile that witness separately against the existing issue before filing or claiming closure. No policy question appears necessary for rejecting the wholly unmapped required values in #780; response/JSON-RPC consumer scope was checked before implementation.

## Implemented ordering and consumer correspondence

`HTTPEndpointExpr.OrderedResponses` now owns the existing stable partition:
tagged responses retain their authored order, followed by untagged fallbacks.
Transport IR consumes that query instead of performing its own partition. HTTP
coverage validates every success representation; JSON-RPC validates only the
first effective representation, matching its existing generated client. The
regression authors the fallback first, so selecting the first authored response
would fail the control. This move changes no response ordering or tag policy.
The existing response-order TLA+ model is unchanged: the historical swap fails
`SelectsFirstMatch`, while the stable partition checks 576 generated / 384
distinct states. Its README links the new owner and both consumers.

Request coverage consumes the prepared wire binding plan, selected/listed or
inferred runtime body, JSON-RPC payload ID and named MapParams. Response/error
coverage consumes the existing body projection, headers, cookies and typed Tag.
Result views retain projected requiredness. Diagnostics use the prepared response
context; the error's embedded design expression is not populated until Finalize.
References never add absent body members, and defaults never invent a conversion
visit for an omitted field. The phase test checks both rules before and after
Finalize, including suffixed logical names.

## Bounded validation and intended changes

- Full expression, DSL and transport IR race suites pass (7.096s, 1.530s,
  1.505s). Scoped production/test and explicit testdata lint report zero issues.
- The exact archived parent plus the two added regression files reproduces seven
  missing-admission cases (0.498s), including the effective-first JSON-RPC case;
  the accepted source/phase controls pass in the candidate.
- Existing HTTP selected-body defaults, custom error projection and mapped-name
  modules pass; JSON-RPC mapped-name and error-discriminator modules pass
  (13.296s). These harnesses compile complete generated code and exercise mounted
  decoding with supplied values. No generated transport implementation changed.
- The existing response-selection fixture had required tag fields missing from
  some branches. Each response now maps them to two headers, preserving authored
  Required declarations; its sanctioned golden update adds exactly eighteen
  header assignments. Mounted decoding checks both reconstructed tag values.
- The typed-tag carrier fixture now gives required fallback cases a complete body,
  and its bodyless response maps required message through a header. Optional
  fallback kind remains absent. Mounted required/optional/nullable/tag controls
  pass; the final affected ordering/tag race run passes in 5.728s.
- Two expression selected-body controls now map the other required field to a
  header. The IR response-contract fallback now uses its complete inferred body.
  These are fixture corrections for the newly enforced coverage rule, not weaker
  Required declarations or renderer exceptions.
- Actual CLI parent/candidate/independent-repeat generation uses the same launcher
  name, toolchain and inputs. All fifty artifacts across HTTP mapped-body/custom
  error and JSON-RPC mapped-body designs are byte-identical. Parent and candidate
  complete modules build and vet. The physical parent is an exact git archive;
  no dropped GOFLAGS overlay can substitute candidate source for it.
- Six actual CLI designs (HTTP selected/listed/default/custom error; JSON-RPC
  selected/custom error) are accepted by that parent and rejected by the candidate
  with the missing logical field diagnostic. Every prior generated file remains
  unchanged after each rejection.

The first comparison used different launcher basenames, which only changed
header comments; it was superseded by process-isolated regeneration with identical
`loom` basenames. Complete build/vet evidence remains valid because all intervening
changes were verified comment-only. An initial parent test overlay did not add
new files and ran no tests; it was superseded by adding the two exact regression
files to the archived parent and running them there. Initial lint identified two
new test-style issues, now corrected; a wildcard testdata lint matched no files,
so the explicit package invocation is the authorizing check.

Evidence and exact inputs are frozen in `/tmp/loom-780-final-manifest.json` before
review. No full exported-design corpus, unrelated fixture regeneration, new
concurrency model or overall CI claim is part of this acceptance.
