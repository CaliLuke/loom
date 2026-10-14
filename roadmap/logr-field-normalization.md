# Logr adapter field normalization — #747

Parent: `8375ce604`. Info, Error and WithValues each implement the same unguarded
alternating-pair conversion, so a trailing key panics before logging. The shared
owner is the adapter boundary that converts untyped logr arguments into ordered
Loom fields, not the log writer or formatter.

1. [x] Add public logr adapter regressions for Info, verbose Info, Error and
   WithValues. Include empty/even/odd arguments, repeated keys, non-string keys,
   explicit nil values, inherited fields and caller-slice isolation. Confirm the
   odd inputs fail on the parent before changing production.
2. [x] Replace all three pair loops with one private normalization helper.
   Preserve ordering, fmt.Sprint key conversion and explicit values; give an odd
   trailing key the `<no-value>` marker, as pinned logr's funcr adapter does.
   Normalize bound fields separately from later event fields; never pair a bound
   orphan with an event argument. Do not mutate caller slices or logger parents.
3. [ ] Run direct regressions, existing adapter tests, full clue/log races and
   scoped lint. Document fallback and shared ownership. Obtain fresh exact-diff
   review, then commit/push and close the owning report; clean temporary logs.

This stateless conversion does not change logger queue or output ownership.
No formal state model, dependency change or generated-code audit is needed.

All eight odd-argument entry cases fail on the parent; the separate bound-field
probe also panics there. The 20 empty/even/duplicate/non-string/nil controls pass.
The candidate passes all 28 adapter cases and independent bound/event/parent
isolation, with full clue/log race checks (1.944s) and scoped lint (zero issues).
An initial isolation test used the default buffered logger and therefore saw no
immediate output; its final form uses the existing WithDebug test setup. That
initial failure is not counted as a production regression. No generated output
or asynchronous logging owner changed. Final review and delivery remain pending.
