# Compact exact-number storage

`CompactNumber` models normalization of sign*C*2^E2*5^E5 and checked native
reconstruction. It informed the inline uint64 coefficient and signed int32
exponents in `internal/exactnum/compact.go`; numbers outside those storage bounds
retain immutable arbitrary-precision storage. The model checks 8-bit unsigned
reconstruction, coefficients 0..31, initial nonnegative exponents 0..4 and both
signs. Normalization may increase exponents. Overflow must choose the exact
large-number comparator, never truncate into a native magnitude.

## Results and reproduction

With TLA+ v1.7.4 (TLC 2.19), jar SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`, and
Corretto Java 25.0.4.1:

- `Checked.cfg` passes normalized meaning, exact native reconstruction, necessary
  fallback and completion: 7,096 generated states, 6,192 distinct, depth 14.
- `Wrapped.cfg` violates `ExactNative`: 5,381 generated, 4,477 distinct, depth 5.
  One counterexample reconstructs 11*5^2=275 as 19 after unsigned wraparound.

From this directory:

```sh
java -Xmx512m -cp "$TLA_JAR" tlc2.TLC -workers 1 \
  -metadir "$TLC_STATE" -config Checked.cfg CompactNumber.tla
```

Repeat with Wrapped.cfg, requiring the named invariant failure. Keep checker
states, trace output and logs outside the repository.

## Correspondence and limits

`compact_test.go` checks native extrema, compact/promoted equivalence, storage
boundaries, zero/sign identity, overflow fallback and IEEE float decoding against
independent big.Rat arithmetic, including random finite bit patterns, subnormal
values and adjacent extremes. Allocation tests require native integer conversion
and comparison to allocate no heap objects. Existing source/domain/boundary tests
cover authored grammar, arbitrary exponents, canonical keys/text and exact native
carrier endpoints. Generated/imported OpenAPI and runtime JSONShape checks retain
the same numeric contracts across consumers.

The model does not establish IEEE decoding, arbitrary-precision parsing or unique
zero storage. Its fallback state does not prove the large comparator. The Lean
value-contract proofs establish their mathematical premises; they do not prove
this Go storage implementation. Direct oracle and consumer tests establish that
correspondence within their tested domains. No numeric rounding or admitted-value
policy changes are intended.
