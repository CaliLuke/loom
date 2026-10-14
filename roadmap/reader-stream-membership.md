# Reader stream membership (#773)

## Contract and owner

A Reader identifies membership by the Redis stream key. Equivalent Stream handles
must be interchangeable for duplicate admission and removal. A registration
instance still identifies the lifetime of an admitted read: removing and re-adding
the same key creates a new registration whose cursor old replies cannot advance.
Removing membership does not drain already admitted delivery.

## Sequential implementation

- [x] Reproduce the retained equivalent-handle witness and extend direct membership
  controls for the same handle, an equivalent handle and an absent key.
- [x] Share one locked key lookup between duplicate admission and removal. Keep
  registration-reference comparison for read completion. Exercise equivalent-handle
  removal/re-add at the existing held-reply and fanout boundaries.
- [x] Run affected Reader races and the existing registration model; record evidence
  and update owning documentation.
- [ ] Obtain independent exact-diff review, then commit, push and close #773.

## Validation scope

No generator behavior changes. Existing `ReaderRegistration` models the selected
registration's removal and replacement, abstracting the public handle lookup;
its invariants and transitions remain unchanged. Direct tests expose the incorrect
pointer selection, while existing held-reply/fanout controls establish correspondence
with registration-lifetime ownership. No new scheduling or recovery mechanism is
introduced. Broader network shutdown is owned by #800.

## Evidence

Parent `4db060dfc` fails the same/equivalent removal and re-add regression
(`/tmp/loom-773-parent.log`, 0.653 s). Both direct membership and held-reply/fanout
controls pass after the shared lookup: affected Reader races 1.820 s; full streaming
race suite 5.590 s; configured scoped lint reports zero issues. Unchanged
`admission-identity.cfg` passes 384 generated / 192 distinct states. No generator,
wire protocol or dependency changes require regeneration. The exact final packet
is `/tmp/loom-773-final-manifest.json`.
