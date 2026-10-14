# CLI declaration and example argument identity

`CLIArguments.tla` explores two services, independent multipart/interceptor/
streaming membership, and each one-service or two-service order. A leading
shared dialer and ordered per-service configurers precede service-local extras.
The invariant compares semantic dependency identities at every position.

The `current` strategy reproduces #697: declarations interleave multipart and
interceptors per service, while calls group all multipart arguments first.
`unfiltered` rejects a local fix that orders service extras correctly but ignores
server subset/order. `owned` uses the same selected ordered plan for both.

Run with pinned TLA+ tools 1.7.4 (TLC 2.19):

```sh
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -workers 1 \
  -metadir /tmp/loom-cli-arguments-owned -config owned.cfg CLIArguments.tla
```

Run each configuration with a distinct external metadir. Current and unfiltered
must fail `MatchingIdentity`; owned must pass. Keep source/configuration and
concise results here; remove checker state after independent review.

Correspondence: `buildCLIArgumentPlan` in `http/codegen/cli_arguments.go` owns
selected endpoint-bearing services and ordered semantic dependency identities.
`appendCLIArgumentParameters` resolves its entries against parser-local allocated
commands; `renderExampleCLIEnd` resolves them against example locals and encoder
functions. `TestCLIArgumentPlanPreservesSelectedIdentity` checks service/endpoint
identity through alias allocation, including per-service configurer types.
`TestCLIArgumentPlanExcludesEndpointFreeServices` checks the selection boundary.
`TestCLIArgumentOrder` compares actual emitted dependency order;
`TestGeneratedCLIArgumentOrder` compiles and vets complete generated example apps.

The model abstracts endpoint multiplicity, Go types, import renaming, codec
functions, transport validation and actual generated syntax. Those are bounded
Go-test and generated-build obligations. This is a finite ordering exploration,
not a protocol or runtime concurrency proof.

Readiness results: `current` and `unfiltered` exit 12 with the intended
`MatchingIdentity` counterexamples; `owned` exits 0, 512 generated / 256 distinct
states. Current fails even when the two features belong to different services.
