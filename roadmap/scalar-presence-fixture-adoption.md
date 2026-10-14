# Scalar presence fixture adoption (#932)

1. [x] Reproduce the four failing HTTP tests on current main with race checks.
2. [x] Make the shared multipart harness carry omission separately from the
   scalar contents. Keep missing-field assertions for absent carriers and add
   present-empty conversion failures for numeric/custom carriers.
3. [x] Update direct rendered-code checks to assert the shared raw/presence
   extraction while retaining mapped names and validation-placement checks.
4. [x] Rerun the exact failures and existing mapped generated wire controls.
   No generator implementation or production wire behavior changes.
5. [ ] Independently review the exact final diff, commit atomically, push and
   close the report; reconcile the central inventory.

Acceptance: the four exact parent failures and existing generated mapped-name
wire controls pass with race checks on current source (44.463s). The multipart
matrix retains callback and service invocation counts, status, problem code and
detail, decoded payload checks, constraints, and framing/body-limit controls.
Supplied empty query/header/cookie numeric or custom values are rejected as
invalid values; omission remains a missing-field error. No generated artifacts
or production implementation change.
